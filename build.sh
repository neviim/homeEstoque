#!/usr/bin/env bash
# build.sh — Compila backend + frontend com versão sincronizada.
# Uso: ./build.sh [--patch|--minor|--major]
#   --patch  (padrão) incrementa X.Y.Z → X.Y.(Z+1)
#   --minor  incrementa X.Y.Z → X.(Y+1).0
#   --major  incrementa X.Y.Z → (X+1).0.0

set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"

BUMP="${1:-}"
case "$BUMP" in
  --minor) BUMP_TYPE="minor" ;;
  --major) BUMP_TYPE="major" ;;
  *)       BUMP_TYPE="patch" ;;
esac

# ── 1. Lê e bumpa a versão ────────────────────────────────────────────────────
CURRENT="$(cat "$ROOT/VERSION" | tr -d '[:space:]')"
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT"

case "$BUMP_TYPE" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

VERSION="${MAJOR}.${MINOR}.${PATCH}"
printf '%s\n' "$VERSION" > "$ROOT/VERSION"
echo "▶ Versão: $CURRENT → $VERSION"

# ── 2. Sincroniza frontend/package.json ──────────────────────────────────────
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$VERSION\"/" "$ROOT/frontend/package.json"

# ── 3. Compila backend ────────────────────────────────────────────────────────
echo "▶ Compilando backend..."
cd "$ROOT/backend"
mkdir -p "$ROOT/bin"
go build \
  -ldflags "-X main.version=${VERSION}" \
  -o "$ROOT/bin/api" \
  ./cmd/api
cd "$ROOT"

# ── 4. Compila MCP ───────────────────────────────────────────────────────────
echo "▶ Compilando MCP..."
HOMEESTOQUE_VERSION="$VERSION" bash "$ROOT/tools/build-mcp.sh"

# ── 5. Compila frontend ───────────────────────────────────────────────────────
echo "▶ Compilando frontend..."
cd "$ROOT/frontend"
VITE_APP_VERSION="$VERSION" npm run build
cd "$ROOT"

echo ""
echo "✔ Build v${VERSION} concluído"
echo "  Backend : $ROOT/bin/api"
echo "  Frontend: $ROOT/frontend/dist"
