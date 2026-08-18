# Deployment

Single small VPS (2 GB RAM, Debian 13), Docker Compose. CI builds the image, pushes it to `ghcr.io`, then
runs [`deploy/site.yml`](deploy/site.yml) against the server as `root`.

That playbook is the description of the box: Docker, swap, the firewall, sshd, the admin account, unattended
upgrades, `/srv/knobel-manager`, the backup cron and the container update itself are all tasks in it, and
they run on every deploy. Read it for *what* is on the server. This file holds only what the tasks cannot
say — why a few things are the way they are, the parts that live outside this repo, and the operations you
still run by hand.

## Bootstrap a new VPS

1. Debian 13 image with a root SSH key installed — the provider's "SSH key" field at create time
2. DNS `A` record pointing at it (see [DNS](#dns-inwx))
3. GitHub: the variables and secrets [below](#github-secrets-and-variables)
4. Push to `main`

Database migrations run at app startup (goose), so there is no migration step.

CI authenticates as `root`: configuring a host cannot be pinned to a forced `command=` in `authorized_keys`
the way the old restart-only deploy key was. Use a dedicated key pair for it
(`ssh-keygen -t ed25519 -C github-ci`) so revoking CI is one line, and treat push access to `main` as root
access to the server — protect the branch and the `production` environment, not just the key. Your own login
is the `admin_user` account, whose keys the playbook takes from the GitHub account in `admin_keys`.

Running the playbook by hand is safe; it is idempotent and has no deploy-specific step:

```bash
pipx install --include-deps ansible
export ACME_EMAIL=... DB_PASSWORD=... FIREBASE_SECRET=...
ansible-playbook -i "<vps>," -u root deploy/site.yml
```

Everything configurable is a `var` at the top of the playbook; override one for a single run with `-e`, e.g.
`-e image_tag=main-1a2b3c4`.

## DNS (INWX)

The API lives on a subdomain, so it is a plain A record — no forwarding, no CNAME. In the INWX web panel:
*Domains → knobel-manager.de → Nameserver/DNS*, then **Neuer Record**:

| Type | Hostname | Value        | TTL  |
|------|----------|--------------|------|
| A    | `api`    | VPS IPv4     | 3600 |
| AAAA | `api`    | VPS IPv6     | 3600 |

Skip the AAAA record unless the VPS actually has a working IPv6 address — a dangling AAAA makes clients
(and the Let's Encrypt challenge) hang on v6 before falling back.

Do **not** use INWX *Weiterleitung* (redirect): that is an HTTP forward through their proxy, which breaks
the ACME challenge and mangles API requests.

Cutting over from another host: set the TTL to `300` a day in advance, switch the record, restore `3600`
afterwards. Verify before starting Caddy — a failed challenge counts against Let's Encrypt rate limits:

```bash
dig +short api.knobel-manager.de
```

Optional: a CAA record on the apex (`knobel-manager.de`, type `CAA`, value `0 issue "letsencrypt.org"`) so
no other CA can issue for the domain. If you add one, it must include `letsencrypt.org` or issuance fails.

## TLS / Let's Encrypt

Caddy handles it: HTTP-01 challenge on first start, renewal ~30 days before expiry. No certbot, no cron, no
renewal hook. Port 80 has to stay reachable — it is the challenge, not just the redirect.

Certificates live in the `caddy-data` volume. Don't delete that volume casually: Let's Encrypt rate-limits
new certs (5 per domain per week). Backing it up is optional, a lost volume just means a re-issue.

HTTP→HTTPS and HTTP/2 are Caddy defaults, and HTTP/3 comes along for free (hence `443/udp`). To serve
strictly HTTP/1.1 + HTTP/2, add `servers { protocols h1 h2 }` to the global block in the `Caddyfile`.

HSTS is set in the `Caddyfile`, not by the app: `SecurityHeaders` only emits it when `r.TLS != nil`, and
behind the proxy the app speaks plain HTTP, so that condition is never true in production.

## Secrets

The playbook renders `/srv/knobel-manager/.env` from `deploy/env.j2`, so the values come from GitHub and the
file on the box is disposable — editing it there does not survive the next deploy, which leaves a timestamped
backup beside it. Rotate by changing the variable or secret in GitHub and re-running the pipeline.

- The pipeline sees the app's secrets now. The old restricted key only restarted containers that read a file
  already on the box.
- `.env` is readable by root and the `docker` group through `docker inspect` or `/proc/<pid>/environ`. That
  group is root-equivalent anyway.
- Keep `DB_PASSWORD` alphanumeric. `.env` is also Compose's interpolation source, so a `$` reads as a
  variable reference and truncates the value — `pa$word` reaches Postgres as `pa`.
- `DATABASE_URL` is deliberately **not** in `.env`: Compose assembles it from the `DB_*` values, so
  credentials are written once and Postgres and the app cannot drift apart. A `DATABASE_URL` in `.env` would
  be ignored anyway — `environment:` wins over `env_file:`.

**Postgres gotcha:** `DB_USER`, `DB_PASSWORD` and `DB_NAME` only create anything on the first start, when the
data volume is initialised. Changing them later renames nothing — it only changes what the app connects with,
which then fails. Change the password in Postgres first, then in GitHub:

```bash
cd /srv/knobel-manager
docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
# at the psql prompt:  ALTER USER "knobel-manager" PASSWORD 'new-password';
```

**Firebase:** the app only calls `VerifyIDToken`, which validates signatures against Google's public certs —
it never reads or writes Firebase data, so use a *dedicated* service account; there is no IAM role it needs.
Any service account key of the project can mint custom tokens for arbitrary users, so the key is still worth
protecting: if the VPS is ever compromised, revoke it under Firebase Console → Project settings → Service
accounts, issue a new one, update the secret.

No Docker secrets, Vault or SOPS here: the app reads plain env vars, so a secrets store would mean code
changes plus another moving part on a 2 GB box.

## Ports

**ufw does not protect published container ports.** Docker writes its own iptables rules in the `DOCKER`
chain, which are evaluated before ufw's, so a port published as `"5432:5432"` is open to the internet even
with `ufw deny` in place. Binding to `127.0.0.1:` is what keeps it private — never drop that prefix from a
`ports:` entry and assume the firewall covers you.

Metrics (9090) are therefore not published at all — only reachable inside the compose network, via
`docker compose exec app curl localhost:9090/metrics`. Postgres is published to `127.0.0.1:15432` for the
SSH tunnel below.

## Database access

No pgAdmin on the box: it costs ~300 MB of a 2 GB server and puts a database admin panel on the public
internet. `db` publishes `127.0.0.1:15432:5432`, bound to the VPS's loopback interface, so the only way in is
an SSH session. Tunnel from your machine and point the GUI you already have (DBeaver, DataGrip, pgAdmin) at
`localhost:15432` with the `DB_*` values from `.env` and SSL disabled — SSH already encrypts the hop:

```bash
ssh -N -L 15432:127.0.0.1:15432 <admin>@<vps>
```

For a quick query, skip the tunnel entirely — `psql` inside the container connects over the unix socket,
which the Postgres image trusts, so it needs no password:

```bash
ssh -t <admin>@<vps> 'cd /srv/knobel-manager && docker compose exec db sh -c "psql -U \$POSTGRES_USER -d \$POSTGRES_DB"'
```

Use the admin account for both, not the CI key — that one is `root`'s and belongs in GitHub.

## Backups

Nightly dumps in `/var/backups/knobel-manager`, 7 kept. A backup on the same disk is not a backup — copy that
directory somewhere else; nothing in the playbook does, and nothing tells you when a dump fails. Restore:

```bash
cd /srv/knobel-manager
gunzip -c /var/backups/knobel-manager/knobel-manager-2026-08-17.sql.gz \
  | docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## Upgrades

Host packages come from `unattended-upgrades`, container images from Renovate plus the pipeline. Postgres
majors in `deploy/compose.yaml` are excluded from Renovate: the new major refuses to start on the old data
directory, so it needs a dump, restore and volume swap, done by hand.

## Rollback

```bash
export ACME_EMAIL=... DB_PASSWORD=... FIREBASE_SECRET=...
ansible-playbook -i "<vps>," -u root deploy/site.yml -e image_tag=main-1a2b3c4
```

This lasts until the next push to `main`, which redeploys the `main` tag. To make a rollback stick, revert
the commit.

## GitHub secrets and variables

| Name              | Kind     | Value                                              |
|-------------------|----------|----------------------------------------------------|
| `VPS_HOST`        | variable | server IP or hostname                              |
| `VPS_HOST_KEY`    | variable | `ssh-keyscan -t ed25519 <host>` output, one line   |
| `VPS_SSH_KEY`     | secret   | private key for `root` on the VPS                  |
| `ACME_EMAIL`      | variable | contact address for Let's Encrypt                  |
| `DB_PASSWORD`     | secret   | Postgres password (must match the existing volume) |
| `FIREBASE_SECRET` | secret   | base64-encoded Firebase service account JSON       |
