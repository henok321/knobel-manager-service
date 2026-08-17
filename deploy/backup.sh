#!/usr/bin/env bash
set -euo pipefail

dir=/var/backups/knobel-manager
mkdir -p "$dir"

out="$dir/knobel-manager-$(date +%F).sql.gz"

cd /srv/knobel-manager
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB"' | gzip >"$out.part"
mv "$out.part" "$out"

find "$dir" -name 'knobel-manager-*.sql.gz' -mtime +7 -delete
