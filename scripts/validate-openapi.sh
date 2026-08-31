#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "Regenerating code from openapi/openapi.yaml..."
(cd openapi/config &&
  go tool oapi-codegen -config=health.yaml ../openapi.yaml &&
  go tool oapi-codegen -config=api.yaml ../openapi.yaml) >/dev/null

if ! git diff --exit-code -- ./gen; then
  echo "gen/ does not match openapi/openapi.yaml. It was regenerated in place: review the diff above and commit gen/ together with the spec." >&2
  exit 1
fi

echo "✓ Checked-in generated code matches OpenAPI spec"
