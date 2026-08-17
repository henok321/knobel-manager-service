# Deployment

Single small VPS (2 GB RAM, Debian 13), Docker Compose, no Coolify. CI builds the image and pushes it to
`ghcr.io`; the VPS only pulls and restarts.

Server-side files live in [`deploy/`](deploy): `compose.yaml`, `Caddyfile`, `.env.example`, `deploy.sh`,
`backup.sh`. The first three are copied to `/srv/knobel-manager` on the VPS, the scripts to
`/usr/local/bin`.

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

- `DOMAIN` in `.env` resolves (A/AAAA record) to the VPS
- ports 80 **and** 443 reachable from the internet — port 80 is the ACME challenge, not just a redirect
- `ACME_EMAIL` set, so Let's Encrypt can warn about expiry problems

Certificates live in the `caddy-data` volume. Don't delete that volume casually — Let's Encrypt rate-limits
new certs (5 per domain per week). Backing it up is optional; a lost volume just means a re-issue.

HTTP→HTTPS redirect and HTTP/2 are Caddy defaults, nothing to configure. HTTP/3 comes along for free
(hence `443/udp` in the compose ports and the firewall); to serve strictly HTTP/1.1 + HTTP/2, add
`servers { protocols h1 h2 }` to the global block in the `Caddyfile`.

HSTS is set in the `Caddyfile`, not by the app: `SecurityHeaders` only emits it when `r.TLS != nil`, and
behind the proxy the app speaks plain HTTP, so that condition is never true in production.

## First-time server setup

```bash
# 1. Docker from the official apt repo (signed, and covered by unattended-upgrades)
apt install -y ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  >/etc/apt/sources.list.d/docker.list
apt update && apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 2. Deploy user (CI logs in as this user)
adduser --disabled-password --gecos "" deploy
usermod -aG docker deploy
mkdir -p /home/deploy/.ssh && chmod 700 /home/deploy/.ssh
install -m 755 deploy/deploy.sh /usr/local/bin/knobel-manager-deploy
# paste the CI public key, prefixed with the forced command (see "CI key" below):
nano /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh

# 3. App directory
mkdir -p /srv/knobel-manager && chown deploy:deploy /srv/knobel-manager
# copy deploy/compose.yaml, deploy/Caddyfile and .env (from deploy/.env.example) there
chmod 600 /srv/knobel-manager/.env

# 4. ghcr access (only if the package is private)
docker login ghcr.io -u <github-user> --password-stdin   # PAT with read:packages

# 5. Start
cd /srv/knobel-manager && docker compose up -d --wait
```

Database migrations run automatically on app startup (goose), so there is no separate migration step.

## CI key

`deploy` is in the `docker` group, which is root-equivalent on the host — a stolen CI key could otherwise
mount `/` into a container. Pin the key to the deploy script so it can do nothing else; sshd runs the
forced command and ignores whatever the client sends:

```text
# /home/deploy/.ssh/authorized_keys
command="/usr/local/bin/knobel-manager-deploy",no-pty,no-port-forwarding,no-agent-forwarding,no-X11-forwarding ssh-ed25519 AAAA... github-ci
```

Use a dedicated key pair for CI (`ssh-keygen -t ed25519 -C github-ci`), private half in `VPS_SSH_KEY`.
Your own admin access stays on a separate user and key, unrestricted.

Key-only auth is what makes the username itself uninteresting to an attacker — turn password logins off:

```bash
# /etc/ssh/sshd_config.d/hardening.conf
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
```

```bash
systemctl restart ssh
```

## Secrets

All of them live in one file on the VPS: `/srv/knobel-manager/.env`, mode `600`, owner `deploy`. Compose
reads it for interpolation (`DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DOMAIN`, `ACME_EMAIL`) and hands it to
the app as `env_file`.

`DATABASE_URL` is **not** in `.env` — compose assembles it from the `DB_*` values, so credentials are
written once and Postgres and the app cannot drift apart. A `DATABASE_URL` in `.env` would be ignored
anyway: `environment:` wins over `env_file:`.

- **Not in git.** `deploy/.env` is gitignored; `deploy/.env.example` is the committed template.
- **Not in CI.** GitHub holds only `VPS_HOST` and `VPS_SSH_KEY`. The pipeline never sees or injects app
  secrets — it restarts containers that read the file already on the box.
- **Readable by root and the `docker` group** via `docker inspect` or `/proc/<pid>/environ`. That group is
  root-equivalent anyway, which is why the CI key is pinned to a single command.

Rotate by editing `.env` and running `docker compose up -d` — changed env recreates the container.

**Postgres gotcha:** `DB_USER`, `DB_PASSWORD` and `DB_NAME` only create anything on first start, when the
data volume is initialised. Editing them later does not rename the user or change the password on an
existing database — it only changes what the app tries to connect with, which then fails. To rotate the
password, change it in Postgres first, then in `.env`:

```bash
docker compose exec db sh -c 'psql -U "$POSTGRES_USER"'
# at the psql prompt:  ALTER USER knobel PASSWORD 'new-password';
# then update DB_PASSWORD in .env, and:
docker compose up -d
```

**Firebase:** the app only calls `VerifyIDToken`, which validates signatures against Google's public certs
— it never reads or writes Firebase data. So use a *dedicated* service account for this deployment rather
than a shared one; there is no IAM role it needs. Note that any service account key of the project can mint
custom tokens for arbitrary users, so the key is still worth protecting: if the VPS is ever compromised,
revoke it under Firebase Console → Project settings → Service accounts, issue a new one, update `.env`.

No Docker secrets, Vault or SOPS here: the app reads plain env vars, so a secrets store would mean code
changes plus another moving part on a 2 GB box, for one file that already never leaves the server.

## Firewall

```bash
apt install ufw
ufw allow 22/tcp
ufw allow 80,443/tcp
ufw allow 443/udp     # HTTP/3
ufw allow 51820/udp   # WireGuard
ufw enable
```

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

Use your own admin login for both, not `deploy` — that account is pinned to the deploy script and has port
forwarding and PTY allocation disabled.

## Swap

2 GB RAM with no swap kills the container on the first spike:

```bash
fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
echo '/swapfile none swap sw 0 0' >>/etc/fstab
sysctl -w vm.swappiness=10 && echo 'vm.swappiness=10' >>/etc/sysctl.conf
```

## Backups

```bash
install -m 755 deploy/backup.sh /usr/local/bin/knobel-manager-backup
echo '15 3 * * * root /usr/local/bin/knobel-manager-backup' >/etc/cron.d/knobel-manager-backup
```

Keeps 7 daily dumps in `/var/backups/knobel-manager`. A backup on the same disk is not a backup — rsync
that directory somewhere else. Restore:

```bash
gunzip -c /var/backups/knobel-manager/knobel-manager-2026-08-17.sql.gz \
  | docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## Host updates

```bash
apt install unattended-upgrades
dpkg-reconfigure -plow unattended-upgrades
```

Container images are updated by Renovate + the CI pipeline.

## Rollback

Set `IMAGE_TAG` in `.env` to a specific build (e.g. `main-1a2b3c4`), then `docker compose up -d --wait`.

## GitHub secrets

| Name          | Value                                                            |
|---------------|------------------------------------------------------------------|
| `VPS_HOST`    | server IP or hostname                                            |
| `VPS_SSH_KEY` | private key whose public half is in `deploy`'s `authorized_keys` |
