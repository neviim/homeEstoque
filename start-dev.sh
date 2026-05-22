#!/usr/bin/env bash
# Sobe backend + frontend em paralelo para desenvolvimento.
# Use Ctrl+C para encerrar ambos.
set -e
cd "$(dirname "$0")"

trap 'kill $(jobs -p) 2>/dev/null; exit' INT TERM

echo "▶ Iniciando backend (Go) em :8080 com hot reload (Air)..."
(cd backend && GOROOT=/home/neviim/go GOPATH=/home/neviim/go /home/neviim/go/bin/air) &

echo "▶ Iniciando frontend (Vite) em :5173..."
(cd frontend && npm run dev) &

wait
