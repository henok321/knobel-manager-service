# Deployment

Single small VPS (2 GB RAM, Debian 13), Docker Compose. CI builds the image, pushes it to `ghcr.io`, then
runs [`deploy/site.yml`](deploy/site.yml) against the server as `root`.

That playbook is the server documentation. Docker, swap, the firewall, sshd, unattended upgrades, the files
in `/srv/knobel-manager`, the backup cron and the container update itself are all tasks in it, and they run
on every deploy — so this file does not repeat them. What is left here is what the code cannot say: the
reasoning, the parts that live outside this repo, and the few things you still do by hand.

## Bootstrap a new VPS

1. Debian 13 image with a root SSH key installed — the provider's "SSH key" field at create time
2. DNS `A` record pointing at it (see [DNS](#dns-inwx))
3. GitHub: the variables and secrets in [the table below](#github-secrets-and-variables)
4. Push to `main`

Database migrations run at app startup (goose), so there is no migration step. The playbook is idempotent
and has nothing deploy-specific in it, so running it yourself is safe:

```bash
pipx install --include-deps ansible
export ACME_EMAIL=... DB_PASSWORD=... FIREBASE_SECRET=...
ansible-playbook -i "<vps>," -u root deploy/site.yml
```

Domain, DB user and name, swap size and image tag are `vars:` at the top of the playbook; override one for a
single run with `-e`, e.g. `-e image_tag=main-1a2b3c4`.

CI authenticates as `root`: configuring a host cannot be pinned to a forced `command=` in `authorized_keys`
the way the old restart-only deploy key was. So use a dedicated CI key pair
(`ssh-keygen -t ed25519 -C github-ci`) — revoking it is then one line and your own admin key is unaffected —
and treat push access to `main` as root access to the server. Protect the branch and the `production`
environment, not just the key.

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

Cutting over from the old host: set the TTL to `300` a day in advance, switch the record, restore `3600`
afterwards. Verify before starting Caddy — a failed challenge counts against Let's Encrypt rate limits:

```bash
dig +short api.knobel-manager.de
```

Optional: a CAA record on the apex (`knobel-manager.de`, type `CAA`, value `0 issue "letsencrypt.org"`)
so no other CA can issue for the domain. If you add one, it must include `letsencrypt.org` or issuance
fails.

## TLS / Let's Encrypt

Caddy handles it: HTTP-01 challenge on first start, renewal ~30 days before expiry. No certbot, no cron, no
renewal hook. It needs the domain to resolve to the VPS and ports 80 **and** 443 reachable from the internet
— port 80 is the challenge, not just the redirect.

Certificates live in the `caddy-data` volume. Don't delete that volume casually — Let's Encrypt rate-limits
new certs (5 per domain per week). Backing it up is optional; a lost volume just means a re-issue.

HTTP→HTTPS redirect and HTTP/2 are Caddy defaults, nothing to configure. HTTP/3 comes along for free (hence
`443/udp` in the compose ports and the firewall); to serve strictly HTTP/1.1 + HTTP/2, add
`servers { protocols h1 h2 }` to the global block in the `Caddyfile`.

HSTS is set in the `Caddyfile`, not by the app: `SecurityHeaders` only emits it when `r.TLS != nil`, and
behind the proxy the app speaks plain HTTP, so that condition is never true in production.

## Secrets

`DB_PASSWORD` and `FIREBASE_SECRET` are GitHub secrets, `ACME_EMAIL` a plain variable — it is a contact
address, not a credential. The playbook renders all three into `/srv/knobel-manager/.env` from
`deploy/env.j2`, mode `600`. Compose reads that file for interpolation
(`DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DOMAIN`, `ACME_EMAIL`) and hands it to the app as `env_file`.

Rotate by changing the GitHub secret and re-running the pipeline. Editing `.env` on the VPS is pointless —
the next deploy overwrites it (the playbook keeps a timestamped backup next to it). That is the trade for
having no drift.

Keep `DB_PASSWORD` alphanumeric. `.env` is also Compose's interpolation source, so a `$` is read as a
variable reference and silently truncates the value the app receives — `pa$word` reaches Postgres as `pa`.

`DATABASE_URL` is **not** in `.env` — compose assembles it from the `DB_*` values, so credentials are
written once and Postgres and the app cannot drift apart. A `DATABASE_URL` in `.env` would be ignored
anyway: `environment:` wins over `env_file:`.

- **Not in git.** Only the `env.j2` template is committed.
- **In CI now.** The playbook needs the values to render the file, so the pipeline sees them. Previously it
  did not — the restricted `deploy` key restarted containers that read a file already on the box.
- **Readable by root and the `docker` group** via `docker inspect` or `/proc/<pid>/environ`. That group is
  root-equivalent regardless.

**Postgres gotcha:** `DB_USER`, `DB_PASSWORD` and `DB_NAME` only create anything on first start, when the
data volume is initialised. Changing `DB_PASSWORD` later does not change the password on an existing
database — it only changes what the app connects with, which then fails. Change it in Postgres first, then
in the GitHub secret:

```bash
cd /srv/knobel-manager
docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
# at the psql prompt:  ALTER USER "knobel-manager" PASSWORD 'new-password';
```

`db_user` and `db_name` are playbook vars rather than secrets, but the same rule applies: they name a role
and a database that only exist because the volume was initialised with them, so changing either breaks the
connection instead of renaming anything.

**Firebase:** the app only calls `VerifyIDToken`, which validates signatures against Google's public certs
— it never reads or writes Firebase data. So use a *dedicated* service account for this deployment rather
than a shared one; there is no IAM role it needs. Note that any service account key of the project can mint
custom tokens for arbitrary users, so the key is still worth protecting: if the VPS is ever compromised,
revoke it under Firebase Console → Project settings → Service accounts, issue a new one, update the secret.

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
internet. Use the GUI you already have on your laptop (DBeaver, DataGrip, pgAdmin) over an SSH tunnel.

The `db` service publishes `127.0.0.1:15432:5432` — bound to the VPS's loopback interface, so the only way
in is an SSH session. Open the tunnel from your machine and leave it running:

```bash
ssh -N -L 15432:127.0.0.1:15432 <admin>@<vps>
```

Connect the GUI to:

| Field    | Value                                         |
|----------|-----------------------------------------------|
| Host     | `localhost`                                   |
| Port     | `15432`                                       |
| Database | `DB_NAME` from `/srv/knobel-manager/.env`     |
| User     | `DB_USER` from `/srv/knobel-manager/.env`     |
| Password | `DB_PASSWORD` from `/srv/knobel-manager/.env` |
| SSL      | disable — SSH already encrypts the hop        |

For a quick query, skip the tunnel entirely — `psql` inside the container connects over the unix socket,
which the Postgres image trusts, so it needs no password:

```bash
ssh -t <admin>@<vps> 'cd /srv/knobel-manager && docker compose exec db sh -c "psql -U \$POSTGRES_USER -d \$POSTGRES_DB"'
```

Use your own admin login for both, not the CI key — that one is `root`'s and should stay in GitHub.

## Backups

Nightly dumps in `/var/backups/knobel-manager`, 7 kept. A backup on the same disk is not a backup — rsync
that directory somewhere else. Restore:

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

## Migrating the existing VPS

The box was set up by hand before the playbook existed. To hand it over:

1. Copy the live values out of `/srv/knobel-manager/.env` into the `DB_PASSWORD` and `FIREBASE_SECRET`
   secrets and the `ACME_EMAIL` variable, and check `DB_USER` and `DB_NAME` against `db_user`/`db_name` in the
   playbook. All three DB values **must** match what the data volume was initialised with — the first
   deploy rewrites `.env` and recreates the app container, so a mismatch is an outage, not a warning.
2. Let root in. The box still carries `PermitRootLogin no` from the old runbook, and the playbook cannot
   change that itself — it needs the login first:

   ```bash
   mkdir -p /root/.ssh && chmod 700 /root/.ssh
   printf '%s\n' 'ssh-ed25519 AAAA... github-ci' >>/root/.ssh/authorized_keys
   chmod 600 /root/.ssh/authorized_keys
   sed -i 's/^PermitRootLogin no$/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config.d/hardening.conf
   systemctl restart ssh
   ```

   Then set `VPS_SSH_KEY` to that key pair's private half. Verify from your machine before pushing:
   `ssh -i <ci-key> root@<vps> true`.
3. Remove the hand-made files the playbook now owns under different names:

   ```bash
   rm /etc/cron.d/knobel-manager-backup      # any whitespace drift from the Ansible-rendered line
                                             # duplicates it, and two concurrent dumps share one .part file
   rm /etc/apt/sources.list.d/docker.list    # replaced by docker.sources, else apt warns on every update
   ```

4. Push. The playbook adopts the existing containers and volumes — the compose project name comes from the
   directory, which does not change, so `compose.yaml`, the `Caddyfile` and the data survive.
5. Retire the old path: `userdel -r deploy`, `rm /usr/local/bin/knobel-manager-deploy` and
   `rm /etc/ssh/sshd_config.d/hardening.conf` — the playbook writes `10-hardening.conf`, which wins on
   precedence, but the old file lingers as drift. The playbook chowns `/srv/knobel-manager` to `root`, so
   confirm with `ls -l` that nothing there is still owned by `deploy` before you delete the user.

## GitHub secrets and variables

| Name              | Kind     | Value                                              |
|-------------------|----------|----------------------------------------------------|
| `VPS_HOST`        | variable | server IP or hostname                              |
| `VPS_SSH_KEY`     | secret   | private key for `root` on the VPS                  |
| `ACME_EMAIL`      | variable | contact address for Let's Encrypt                  |
| `DB_PASSWORD`     | secret   | Postgres password (must match the existing volume) |
| `FIREBASE_SECRET` | secret   | base64-encoded Firebase service account JSON       |
