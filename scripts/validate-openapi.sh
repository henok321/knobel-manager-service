#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Validating: Checked-in generated code matches OpenAPI spec${NC}"
echo ""

# Check if gen/ exists
if [ ! -d "./gen" ]; then
	echo -e "${RED}✗ gen/ directory not found${NC}"
	echo "Run: make openapi-generate"
	exit 1
fi

# Check for uncommitted changes in gen/ (need clean slate)
if ! git diff --quiet HEAD ./gen 2>/dev/null; then
	echo -e "${RED}✗ Uncommitted changes in gen/${NC}"
	echo "Commit or stash changes first, then run validation"
	exit 1
fi

# Regenerate in-place to compare with checked-in code
echo "→ Regenerating code from openapi/openapi.yaml..."
cd openapi/config
go tool oapi-codegen -config=health.yaml ../openapi.yaml >/dev/null
go tool oapi-codegen -config=games.yaml ../openapi.yaml >/dev/null
go tool oapi-codegen -config=teams.yaml ../openapi.yaml >/dev/null
go tool oapi-codegen -config=players.yaml ../openapi.yaml >/dev/null
go tool oapi-codegen -config=tables.yaml ../openapi.yaml >/dev/null
go tool oapi-codegen -config=scores.yaml ../openapi.yaml >/dev/null
cd ../..

echo "→ Comparing regenerated code with checked-in gen/..."

# Check if regeneration changed anything
if ! git diff --quiet ./gen 2>/dev/null; then
	echo -e "${RED}✗ VALIDATION FAILED${NC}"
	echo ""
	echo -e "${YELLOW}Checked-in generated code does NOT match OpenAPI spec${NC}"
	echo ""
	echo "The OpenAPI spec has changed but generated code wasn't updated."
	echo ""
	echo "To fix:"
	echo "  1. Regenerate from spec:  make openapi-generate"
	echo "  2. Review changes:        git diff gen/"
	echo "  3. Commit both together:  git add openapi/ gen/ && git commit"
	echo ""
	# Restore original
	git checkout ./gen 2>/dev/null
	exit 1
fi

echo -e "${GREEN}✓ Checked-in generated code matches OpenAPI spec${NC}"
