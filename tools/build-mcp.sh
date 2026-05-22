#!/usr/bin/env bash
# Compila o servidor MCP do HomeEstoque.
# Saída: ./bin/homeestoque-mcp (na raiz do repo).

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/backend"

mkdir -p "$ROOT/bin"

GOROOT=/home/neviim/go \
GOPATH=/home/neviim/go \
GOMODCACHE=/home/neviim/go/pkg/mod \
  /home/neviim/go/bin/go build -o "$ROOT/bin/homeestoque-mcp" ./cmd/mcp

echo "✔ Binário gerado em $ROOT/bin/homeestoque-mcp"
ls -lh "$ROOT/bin/homeestoque-mcp"
