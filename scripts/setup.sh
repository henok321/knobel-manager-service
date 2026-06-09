#!/bin/bash
set -e

echo "Install git hooks..."

pre-commit install --hook-type pre-commit --hook-type pre-push

echo "Setup database..."
docker-compose up -d

DATABASE_URL=postgres://postgres:secret@localhost:5432/postgres

echo "Init .env..."

{
  echo "ENVIRONMENT=local"
  echo "DB_MIGRATION_DIR=db_migration"
  echo "FIREBASE_SECRET=$(jq -c . ./firebaseServiceAccount.json | base64)"
  echo "DATABASE_URL=$DATABASE_URL"
} >.env
