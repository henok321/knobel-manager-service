#!/usr/bin/env bash
set -euo pipefail

cd /srv/knobel-manager

docker compose pull
docker compose up -d --wait
docker image prune -f
