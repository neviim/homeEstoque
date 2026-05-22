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
  echo "Uso: $0 <email> <nova-senha>"
  exit 1
fi

if [[ ${#SENHA} -lt 6 ]]; then
  echo "Erro: senha deve ter pelo menos 6 caracteres"
  exit 1
fi

cat > /tmp/reset_pwd.go << 'GOEOF'
package main

import (
	"database/sql"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := os.Args[1]
	email  := os.Args[2]
	senha  := os.Args[3]

	db, err := sql.Open("sqlite", dbPath+"?_time_format=sqlite")
	if err != nil { fmt.Fprintln(os.Stderr, "db:", err); os.Exit(1) }
	defer db.Close()

	var id int64
	var name string
	if err := db.QueryRow("SELECT id, name FROM users WHERE email = ?", email).Scan(&id, &name); err == sql.ErrNoRows {
		fmt.Fprintf(os.Stderr, "Usuário não encontrado: %s\n", email)
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, "query:", err); os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil { fmt.Fprintln(os.Stderr, "hash:", err); os.Exit(1) }

	if _, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hash), id); err != nil {
		fmt.Fprintln(os.Stderr, "update:", err); os.Exit(1)
	}

	fmt.Printf("✔ Senha redefinida para o usuário %q (id=%d, email=%s)\n", name, id, email)
}
GOEOF

GOROOT=/home/neviim/go \
GOPATH=/home/neviim/go \
GOMODCACHE=/home/neviim/go/pkg/mod \
  /home/neviim/go/bin/go run \
    -mod=mod \
    /tmp/reset_pwd.go \
    "$ROOT/backend/data/homeestoque.db" \
    "$EMAIL" \
    "$SENHA"
