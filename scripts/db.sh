#!/bin/bash
set -e

echo "Setup database..."
docker-compose up -d

export DATABASE_URL="postgres://postgres:secret@localhost:5432/postgres?sslmode=disable"
export GOOSE_DRIVER=postgres
export GOOSE_MIGRATION_DIR="./db_migration"
export GOOSE_DBSTRING="postgres://postgres:secret@localhost:5432/postgres?sslmode=disable"

goose validate
goose up
