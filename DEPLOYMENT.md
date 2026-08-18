# Deployment

One VPS (2 GB RAM, Debian 13), Docker Compose. CI builds the image, pushes it to `ghcr.io`, then runs
[`deploy/site.yml`](deploy/site.yml) against the server as `root`. The playbook is the server: read it for
what is installed. Below is only what it cannot say.

## Bootstrap a new VPS

1. Debian 13 image, root SSH key in the provider's "SSH key" field
2. DNS `A` record for `api` → the IPv4 (INWX: *Domains → knobel-manager.de → Nameserver/DNS*)
3. GitHub: the variables and secrets [below](#github-secrets-and-variables)
4. Push to `main`

Migrations run at app startup (goose). Your own login is the `admin_user` account; its keys come from the
GitHub account in `admin_keys`. CI is `root` because config management cannot run behind a forced
`command=` — so push access to `main` is root access to the box.

To run it yourself:

```bash
pipx install --include-deps ansible
export ACME_EMAIL=... DB_PASSWORD=... FIREBASE_SECRET=...
ansible-playbook -i "<vps>," -u root deploy/site.yml
```

Settings are `vars` at the top of the playbook, overridable per run: `-e image_tag=main-1a2b3c4`.

## Traps

- **Add no AAAA record** unless the VPS has working IPv6 — a dangling one hangs clients and the ACME
  challenge on v6 first. Never use INWX *Weiterleitung*: it proxies HTTP and breaks the challenge.
- **Keep port 80 open.** It is the ACME challenge, not just the redirect. Certs live in the `caddy-data`
  volume; deleting it means re-issuing against a limit of 5 per domain per week.
- **ufw does not cover published container ports.** Docker's iptables rules in the `DOCKER` chain are
  evaluated first, so `"5432:5432"` is world-open even with `ufw deny`. The `127.0.0.1:` prefix is what
  keeps a port private.
- **`DB_PASSWORD` is restricted to `A-Za-z0-9_.:@+~-`** and the playbook refuses anything else. `.env` is
  Compose's interpolation source: `$` expands, a leading `"` swallows the rest of the file, and a space
  followed by `#` truncates — all silently, with the right value still sitting in the file.
- **HSTS belongs in the `Caddyfile`.** The app only emits it when `r.TLS != nil`, which is never true behind
  the proxy.

## Secrets

The playbook renders `/srv/knobel-manager/.env` from `deploy/env.j2`. Rotate in GitHub and re-run the
pipeline; editing the file on the box does not survive the next deploy. `DATABASE_URL` stays out of it —
Compose builds it from the `DB_*` values so the app and Postgres cannot drift apart.

`DB_USER`, `DB_PASSWORD` and `DB_NAME` only create anything when the data volume is initialised. Changing
them later renames nothing, it just breaks the connection. To rotate the password:

1. Change it in Postgres first — the unix socket inside the container needs no password:

   ```bash
   cd /srv/knobel-manager
   docker compose exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
   # ALTER USER "knobel-manager" PASSWORD 'new-password';
   ```

2. Update the `DB_PASSWORD` secret in GitHub.
3. Re-run the pipeline. Until it finishes, new DB connections fail — the app still holds the old password.
   Then `rm /srv/knobel-manager/.env.*~`, those backups carry the old value.

Use a dedicated Firebase service account with no IAM role — the app only verifies ID tokens. Any key of the
project can mint tokens for arbitrary users, so revoke it if the VPS is compromised.

## Database access

`db` publishes `127.0.0.1:15432:5432`, so access needs an SSH session. Tunnel, then point a GUI at
`localhost:15432` with the `DB_*` values from `.env`, SSL off:

```bash
ssh -N -L 15432:127.0.0.1:15432 <admin>@<vps>
```

Or skip the tunnel — the unix socket inside the container needs no password:

```bash
ssh -t <admin>@<vps> 'cd /srv/knobel-manager && docker compose exec db sh -c "psql -U \$POSTGRES_USER -d \$POSTGRES_DB"'
```

## Backups

Nightly dumps in `/var/backups/knobel-manager`, 7 kept. Nothing copies them off the disk and nothing reports
a failed one. Restore:

```bash
cd /srv/knobel-manager
gunzip -c /var/backups/knobel-manager/knobel-manager-2026-08-17.sql.gz \
  | docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## Rollback

```bash
ansible-playbook -i "<vps>," -u root deploy/site.yml -e image_tag=main-1a2b3c4
```

Lasts until the next push to `main`. To make it stick, revert the commit.

Postgres major bumps in `deploy/compose.yaml` are excluded from Renovate: the new major refuses to start on
the old data directory, so it needs a dump, restore and volume swap by hand.

## GitHub secrets and variables

| Name              | Kind     | Value                                              |
|-------------------|----------|----------------------------------------------------|
| `VPS_HOST`        | variable | server IP or hostname                              |
| `VPS_HOST_KEY`    | variable | `ssh-keyscan -t ed25519 <host>` output, one line   |
| `VPS_SSH_KEY`     | secret   | private key for `root` on the VPS                  |
| `ACME_EMAIL`      | variable | contact address for Let's Encrypt                  |
| `DB_PASSWORD`     | secret   | Postgres password (must match the existing volume) |
| `FIREBASE_SECRET` | secret   | base64-encoded Firebase service account JSON       |
