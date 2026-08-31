#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

echo "Regenerating code from openapi/openapi.yaml..."
rm -rf ./gen
(cd openapi/config &&
  go tool oapi-codegen -config=health.yaml ../openapi.yaml &&
  go tool oapi-codegen -config=api.yaml ../openapi.yaml) >/dev/null

# Compare against HEAD, not the index, and count untracked files: staging a stale gen/
# or generating a new file both have to fail here, not slip through.
if [ -n "$(git status --porcelain --untracked-files=all -- ./gen)" ]; then
  git --no-pager diff HEAD -- ./gen
  git status --short --untracked-files=all -- ./gen
  echo "gen/ does not match openapi/openapi.yaml. It was regenerated in place: review the diff above and commit gen/ together with the spec." >&2
  exit 1
fi

echo "✓ Checked-in generated code matches OpenAPI spec"
