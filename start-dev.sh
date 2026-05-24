#!/usr/bin/env bash
# Sobe backend + frontend em paralelo para desenvolvimento.
# Use Ctrl+C para encerrar ambos.
set -e
cd "$(dirname "$0")"

free_port() {
  local port=$1
  local pids
  pids=$(lsof -ti tcp:"$port" 2>/dev/null || true)
  if [ -n "$pids" ]; then
    echo "⚠ Porta $port em uso por PID(s): $pids — encerrando..."
    kill $pids 2>/dev/null || true
    for _ in 1 2 3 4 5; do
      sleep 0.2
      pids=$(lsof -ti tcp:"$port" 2>/dev/null || true)
      [ -z "$pids" ] && break
    done
    if [ -n "$pids" ]; then
      echo "⚠ Forçando kill -9 em PID(s): $pids"
      kill -9 $pids 2>/dev/null || true
    fi
  fi
}

trap 'kill $(jobs -p) 2>/dev/null; exit' INT TERM

free_port 8080
free_port 5173

echo "▶ Iniciando backend (Go) em :8080 com hot reload (Air)..."
(cd backend && GOROOT=/home/neviim/go GOPATH=/home/neviim/go /home/neviim/go/bin/air) &

echo "▶ Iniciando frontend (Vite) em :5173..."
(cd frontend && npm run dev) &

wait
