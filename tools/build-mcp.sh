#!/usr/bin/env bash
# Compila o servidor MCP do HomeEstoque.
# Saída: ./bin/homeestoque-mcp (na raiz do repo).

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/backend"

mkdir -p "$ROOT/bin"

MCP_VERSION="${HOMEESTOQUE_VERSION:-$(cat "$ROOT/VERSION" | tr -d '[:space:]')}"

go build -ldflags "-X main.version=${MCP_VERSION}" -o "$ROOT/bin/homeestoque-mcp" ./cmd/mcp

echo "✔ Binário gerado em $ROOT/bin/homeestoque-mcp"
ls -lh "$ROOT/bin/homeestoque-mcp"
