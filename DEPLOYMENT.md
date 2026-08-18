# Deployment

Single small VPS (2 GB RAM, Debian 13), Docker Compose, no Coolify. CI builds the image and pushes it to
`ghcr.io`, then runs the Ansible playbook in [`deploy/`](deploy) against the VPS. The playbook is the whole
server configuration — the box is not edited by hand.

Everything server-side lives in [`deploy/`](deploy): `site.yml` (the playbook), `compose.yaml`, `Caddyfile`,
`env.j2` (the `.env` template) and `backup.sh`.

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

Caddy handles it. On first start it requests a certificate from Let's Encrypt via the HTTP-01 challenge, then
renews it automatically (~30 days before expiry). No certbot, no cron, no renewal hook.

Requirements:

- `domain` in the playbook resolves (A/AAAA record) to the VPS
- ports 80 **and** 443 reachable from the internet — port 80 is the ACME challenge, not just a redirect
- the `ACME_EMAIL` secret set, so Let's Encrypt can warn about expiry problems

Certificates live in the `caddy-data` volume. Don't delete that volume casually — Let's Encrypt rate-limits
new certs (5 per domain per week). Backing it up is optional; a lost volume just means a re-issue.

HTTP→HTTPS redirect and HTTP/2 are Caddy defaults, nothing to configure. HTTP/3 comes along for free
(hence `443/udp` in the compose ports and the firewall); to serve strictly HTTP/1.1 + HTTP/2, add
`servers { protocols h1 h2 }` to the global block in the `Caddyfile`.

HSTS is set in the `Caddyfile`, not by the app: `SecurityHeaders` only emits it when `r.TLS != nil`, and
behind the proxy the app speaks plain HTTP, so that condition is never true in production.

## Provisioning

`deploy/site.yml` is the server: Docker from the official apt repo, swap, firewall, sshd hardening,
unattended upgrades, the files in `/srv/knobel-manager`, the backup cron — and the container update itself.
CI runs it on every deploy, so the VPS cannot drift from the repo: anything changed by hand on the box is
put back on the next push.

It is idempotent and has no deploy-specific step, so running it yourself is safe:

```bash
pipx install --include-deps ansible
export ACME_EMAIL=... DB_PASSWORD=... FIREBASE_SECRET=...
ansible-playbook -i "<vps>," -u root deploy/site.yml
```

Non-secret settings (domain, DB user and name, swap size, image tag) are `vars:` at the top of the
playbook; override one for a single run with `-e`, e.g. `-e image_tag=main-1a2b3c4`.

### Bootstrap a new VPS

1. Debian 13 image with a root SSH key installed — the provider's "SSH key" field at create time
2. DNS `A` record pointing at it (see [DNS](#dns-inwx))
3. GitHub: `VPS_HOST` variable plus the `VPS_SSH_KEY`, `ACME_EMAIL`, `DB_PASSWORD` and `FIREBASE_SECRET`
   secrets
4. Push to `main`

That is the complete manual part. Database migrations run at app startup (goose), so there is no separate
migration step.

## SSH access

CI authenticates as `root`, because configuring a host is not something that can be pinned to one command.
The old `deploy` user with a forced `command=` in `authorized_keys` is gone, and with it the guarantee that
a stolen CI key could only restart containers. What replaces it:

- a dedicated key pair for CI (`ssh-keygen -t ed25519 -C github-ci`), private half in `VPS_SSH_KEY`, public
  half in `/root/.ssh/authorized_keys` — revoking CI is then one line, and your own admin key is unaffected
- branch protection on `main` and the `production` GitHub environment, since push access is now root access

Key-only auth is what makes the login name itself uninteresting to an attacker. The playbook writes
`/etc/ssh/sshd_config.d/hardening.conf`:

```text
PermitRootLogin prohibit-password
PasswordAuthentication no
KbdInteractiveAuthentication no
```

## Secrets

Three secrets live in GitHub (`ACME_EMAIL`, `DB_PASSWORD`, `FIREBASE_SECRET`); the playbook renders them
into `/srv/knobel-manager/.env` from `deploy/env.j2`, mode `600`. Compose reads that file for interpolation
(`DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DOMAIN`, `ACME_EMAIL`) and hands it to the app as `env_file`.

Rotate by changing the GitHub secret and re-running the pipeline. Editing `.env` on the VPS is pointless —
the next deploy overwrites it. That is the trade for having no drift.

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

**Firebase:** the app only calls `VerifyIDToken`, which validates signatures against Google's public certs
— it never reads or writes Firebase data. So use a *dedicated* service account for this deployment rather
than a shared one; there is no IAM role it needs. Note that any service account key of the project can mint
custom tokens for arbitrary users, so the key is still worth protecting: if the VPS is ever compromised,
revoke it under Firebase Console → Project settings → Service accounts, issue a new one, update the secret.

No Docker secrets, Vault or SOPS here: the app reads plain env vars, so a secrets store would mean code
changes plus another moving part on a 2 GB box.

## Firewall

The playbook enables ufw with a default-deny inbound policy and opens `22/tcp`, `80/tcp`, `443/tcp`,
`443/udp` (HTTP/3) and `51820/udp` (WireGuard).

Metrics (9090) are not published at all — only reachable inside the compose network, via
`docker compose exec app curl localhost:9090/metrics`. Postgres is published to `127.0.0.1:15432` for the
SSH tunnel below.

**ufw does not protect published container ports.** Docker writes its own iptables rules in the `DOCKER`
chain, which are evaluated before ufw's, so a port published as `"5432:5432"` is open to the internet even
with `ufw deny` in place. Binding to `127.0.0.1:` is what keeps it private — never drop that prefix from a
`ports:` entry and assume the firewall covers you.

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

## Swap

2 GB RAM with no swap kills the container on the first spike, so the playbook creates `/swapfile`
(`swap_size`, default 2G), adds the `fstab` entry and sets `vm.swappiness=10` in
`/etc/sysctl.d/99-knobel-manager.conf`. It is created once; changing `swap_size` later does not resize an
existing file.

## Backups

The playbook installs `deploy/backup.sh` as `/usr/local/bin/knobel-manager-backup` and a `03:15` cron
entry. It keeps 7 daily dumps in `/var/backups/knobel-manager`. A backup on the same disk is not a backup —
rsync that directory somewhere else. Restore:

```bash
cd /srv/knobel-manager
gunzip -c /var/backups/knobel-manager/knobel-manager-2026-08-17.sql.gz \
  | docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## Host updates

`unattended-upgrades` is installed and enabled by the playbook (`/etc/apt/apt.conf.d/20auto-upgrades`).
Container images are updated by Renovate plus the pipeline. Postgres major bumps in `deploy/compose.yaml`
are excluded from that: the new major refuses to start on the old data directory, so it needs a dump,
restore and volume swap, done by hand.

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
| `VPS_SSH_KEY`     | secret   | private key for `root` on the VPS                  |
| `ACME_EMAIL`      | secret   | contact address for Let's Encrypt                  |
| `DB_PASSWORD`     | secret   | Postgres password (must match the existing volume) |
| `FIREBASE_SECRET` | secret   | base64-encoded Firebase service account JSON       |

## Migrating the existing VPS

The box was set up by hand before the playbook existed. To hand it over:

1. Copy the live values out of `/srv/knobel-manager/.env` into the `ACME_EMAIL`, `DB_PASSWORD` and
   `FIREBASE_SECRET` GitHub secrets. `DB_PASSWORD` **must** be the one the data volume was initialised
   with, otherwise the app can no longer connect.
2. Put the CI public key in `/root/.ssh/authorized_keys` and change `VPS_SSH_KEY` to the root key.
3. `rm /etc/cron.d/knobel-manager-backup` on the box. The playbook writes its own entry with an Ansible
   marker and would otherwise leave the hand-written line next to it, running the backup twice.
4. Push. The playbook adopts the existing containers and volumes — the compose project name comes from the
   directory, which does not change, so `compose.yaml`, the `Caddyfile` and the data survive.
5. Retire the old path: `userdel -r deploy` and `rm /usr/local/bin/knobel-manager-deploy`.
