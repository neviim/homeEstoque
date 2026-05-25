#!/usr/bin/env bash
# Redefine a senha de um usuário diretamente no banco.
# Uso: ./tools/reset-password.sh <email> <nova-senha>
#
# Exemplo:
#   ./tools/reset-password.sh jaime@test.com minhasenha123

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

EMAIL="${1:-}"
SENHA="${2:-}"

if [[ -z "$EMAIL" || -z "$SENHA" ]]; then
  echo "Uso: $0 <email> <nova-senha>" >&2
  exit 1
fi

if [[ ${#SENHA} -lt 6 ]]; then
  echo "Erro: senha deve ter pelo menos 6 caracteres" >&2
  exit 1
fi

# Ativa o toolchain do mise (Go por projeto). Fallback: usa o 'go' do PATH.
if [[ -x "$HOME/.local/bin/mise" ]]; then
  eval "$("$HOME/.local/bin/mise" activate bash)"
  eval "$("$HOME/.local/bin/mise" hook-env -s bash 2>/dev/null || true)"
elif command -v mise >/dev/null 2>&1; then
  eval "$(mise activate bash)"
  eval "$(mise hook-env -s bash 2>/dev/null || true)"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Erro: 'go' não encontrado no PATH. Instale via mise (mise install) ou ajuste o PATH." >&2
  exit 1
fi

# Cria utilitário temporário dentro do módulo do backend para reaproveitar
# o go.mod (bcrypt + modernc.org/sqlite já são deps do projeto).
WORKDIR="$(mktemp -d "$ROOT/backend/cmd/resetpwd-tmp-XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

cat > "$WORKDIR/main.go" << 'GOEOF'
package main

import (
	"database/sql"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "uso: resetpwd <db-path> <email> <senha>")
		os.Exit(1)
	}
	dbPath, email, senha := os.Args[1], os.Args[2], os.Args[3]

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer db.Close()

	var id int64
	var name string
	err = db.QueryRow("SELECT id, name FROM users WHERE email = ?", email).Scan(&id, &name)
	if err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "Usuário não encontrado: %s\n", email)
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, "query:", err)
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash:", err)
		os.Exit(1)
	}

	if _, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hash), id); err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		os.Exit(1)
	}

	fmt.Printf("✔ Senha redefinida — id=%d nome=%q email=%s\n", id, name, email)
}
GOEOF

cd "$ROOT/backend"
go run "./cmd/$(basename "$WORKDIR")" \
  "$ROOT/backend/data/homeestoque.db" \
  "$EMAIL" \
  "$SENHA"
