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

until docker exec knobel-manager-service-db-1 pg_isready -h localhost -p 5432; do
	echo "Waiting for PostgreSQL to start..."
	sleep 2
done

goose validate
goose up

echo "Init .env..."

echo "ENVIRONMENT=local" >.env
echo "FIREBASE_SECRET=$(jq -c . ./firebaseServiceAccount.json | base64)" >>.env
echo "DATABASE_URL=$DATABASE_URL" >>.env
