#!/bin/bash
set -e

echo "Starting server..."

export DATABASE_URL="postgres://postgres:secret@localhost:5432/postgres?sslmode=disable"
export FIREBASE_SECRET=$(jq -c . ./firebaseServiceAccount.json)

./knobel-manager-service
