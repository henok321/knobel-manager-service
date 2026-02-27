#!/bin/bash
set -e

echo "Install git hooks..."

pre-commit install --hook-type pre-commit --hook-type pre-push

echo "Setup database..."
docker-compose up -d

echo "Init .env..."

echo "ENVIRONMENT=local" >.env
echo "DB_MIGRATION_DIR=db_migration" >.env
echo "FIREBASE_SECRET=$(jq -c . ./firebaseServiceAccount.json | base64)" >>.env
echo "DATABASE_URL=$DATABASE_URL" >>.env
