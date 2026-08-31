#!/bin/bash
set -euo pipefail

if [ ! -f ./firebaseServiceAccount.json ]; then
  echo "firebaseServiceAccount.json is missing: download it from the Firebase Console into the project root." >&2
  exit 1
fi

echo "Install git hooks..."

pre-commit install --hook-type pre-commit --hook-type pre-push

echo "Setup database..."
docker compose up -d

DATABASE_URL=postgres://postgres:secret@localhost:5432/postgres

# Assigned, not inlined into the heredoc below: a failure inside a command substitution used
# as an argument does not stop the script, and a silent empty secret fails much later.
firebase_secret=$(jq -c . ./firebaseServiceAccount.json | base64)

echo "Init .env..."

{
  echo "ENVIRONMENT=local"
  echo "DB_MIGRATION_DIR=db_migration"
  echo "FIREBASE_SECRET=$firebase_secret"
  echo "DATABASE_URL=$DATABASE_URL"
} >.env
