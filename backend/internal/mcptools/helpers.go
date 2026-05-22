// Package mcptools implementa as ferramentas MCP que expõem o inventário
// HomeEstoque ao Claude e a outros clientes MCP. Compartilha banco e models
// com o backend HTTP via o mesmo módulo Go.
package mcptools

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/neviim/homeestoque/backend/internal/database"
)

// Tools encapsula as dependências necessárias para todas as ferramentas MCP.
// Uma instância é criada no main e seus métodos são registrados como handlers.
type Tools struct {
	DB        *sql.DB
	MCPUserID int64 // user_id atribuído a movements criados via MCP
}

// New constrói Tools resolvendo o user_id do "MCP Assistant" criado pelo seed.
func New(db *sql.DB) (*Tools, error) {
	var uid int64
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", database.MCPUserEmail).Scan(&uid)
	if err != nil {
		return nil, fmt.Errorf("resolver usuário MCP (%s): %w — rode o seed primeiro", database.MCPUserEmail, err)
	}
	return &Tools{DB: db, MCPUserID: uid}, nil
}

// generateCode gera um SKU "EST-XXXXXXXX" para novos itens.
// Mantém o mesmo formato usado pelo handler HTTP (item_handler.go).
func generateCode() string {
	return "EST-" + strings.ToUpper(uuid.New().String()[:8])
}

// logOp loga operações de escrita em stderr. Não interfere no stdio JSON-RPC
// porque o SDK só escreve respostas em stdout.
func logOp(tool string, format string, args ...any) {
	log.Printf("[mcp] tool=%s %s", tool, fmt.Sprintf(format, args...))
}

// validLocationTypes é o conjunto fechado de tipos de localização aceitos.
// Espelha as constantes definidas no frontend (Locations.tsx).
var validLocationTypes = map[string]bool{
	"comodo":  true,
	"movel":   true,
	"caixa":   true,
	"armario": true,
	"outro":   true,
}
