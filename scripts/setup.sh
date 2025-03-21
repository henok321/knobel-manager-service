#!/bin/bash
set -e

echo "Install git hooks..."

pre-commit install --hook-type pre-commit --hook-type pre-push


echo "Setup database..."
docker-compose up -d

export DATABASE_URL="postgres://postgres:secret@localhost:5432/postgres?sslmode=disable"
export GOOSE_DRIVER=postgres
export GOOSE_MIGRATION_DIR="./db_migration"
export GOOSE_DBSTRING="postgres://postgres:secret@localhost:5432/postgres?sslmode=disable"

goose validate
goose up

echo "Init .env..."

echo "ENVIRONMENT=local" > .env
echo "FIREBASE_SECRET=$(jq -c . ./firebaseServiceAccount.json | base64)" >> .env
echo "DATABASE_URL=$DATABASE_URL" >> .env