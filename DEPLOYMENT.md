# Deployment

CI builds the image, pushes it to `ghcr.io`, then runs [`deploy/deploy.yml`](deploy/deploy.yml) against the server as
`root`. That playbook deploys *this service only* — three files into `/srv/knobel-manager`, one into the shared Caddy
drop-in directory, then `docker compose up`.

The host itself — Docker, firewall, swap, SSH, backups and the Caddy that terminates TLS — belongs to
[**henok321/homelab**](https://github.com/henok321/homelab). Run that playbook against a box before deploying here; this
one asserts `/srv/edge` exists and stops if it does not.

## How it attaches

| File                  | Lands at                               | Does                                               |
|-----------------------|----------------------------------------|----------------------------------------------------|
| `deploy/compose.yaml` | `/srv/knobel-manager/compose.yaml`     | `app` joins the `edge` network as `knobel-manager` |
| `deploy/site.caddy`   | `/srv/edge/sites/knobel-manager.caddy` | `api.knobel-manager.de` → `knobel-manager:8080`    |
| `deploy/env.j2`       | `/srv/knobel-manager/.env`             | secrets, rendered per run                          |

`db` has no `networks:` key, so it stays on the project's default network and is unreachable from
`edge`. The compose project name comes from the directory, so the data volume is
`knobel-manager_db-data` regardless of what changes around it.

Migrations run at app startup (goose).

To run it yourself:

```bash
pipx install --include-deps ansible
export DB_PASSWORD=... FIREBASE_SECRET=...
ansible-playbook -i "<vps>," -u root deploy/deploy.yml
```

Settings are `vars` at the top of the playbook, overridable per run: `-e image_tag=main-1a2b3c4`.

## Traps

- **`DB_PASSWORD` is restricted to `A-Za-z0-9_.:@+~-`** and the playbook refuses anything else. `.env` is
  Compose's interpolation source: `$` expands, a leading `"` swallows the rest of the file, and a space
  followed by `#` truncates — all silently, with the right value still sitting in the file.
- **HSTS comes from `import common`** in `site.caddy`. The app only emits it when `r.TLS != nil`, which is never true
  behind the proxy.
- **The alias must stay unique across the box.** Caddy addresses `knobel-manager:8080`, not `app:8080` — every service
  on `edge` shares one DNS namespace.
- **A broken `site.caddy` fails loudly but harmlessly.** `caddy reload` validates first and keeps the running config, so
  the other services stay up.

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

The homelab playbook dumps every `/srv/*` stack that has a service named `db`, so this one is covered with no
configuration here. Nightly in `/var/backups/knobel-manager`, 7 kept, nothing copied off the disk and nothing reporting
a failure. Restore:

```bash
cd /srv/knobel-manager
gunzip -c /var/backups/knobel-manager/knobel-manager-2026-08-17.sql.gz \
  | docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## Rollback

```bash
ansible-playbook -i "<vps>," -u root deploy/deploy.yml -e image_tag=main-1a2b3c4
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
| `DB_PASSWORD`     | secret   | Postgres password (must match the existing volume) |
| `FIREBASE_SECRET` | secret   | base64-encoded Firebase service account JSON       |

`ACME_EMAIL` moved to the homelab repo along with Caddy.
