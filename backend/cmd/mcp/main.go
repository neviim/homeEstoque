// Command homeestoque-mcp expõe o inventário HomeEstoque via Model Context Protocol.
//
// Roda como subprocess de um client MCP (Claude Desktop, Claude Code) via stdio.
// Lê DB_PATH do .env / env vars para localizar o mesmo SQLite usado pelo backend HTTP.
package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/mcptools"
)

const serverName = "homeestoque"
const serverVersion = "0.1.0"

func main() {
	// Logs vão para stderr (stdout é reservado para o protocolo JSON-RPC).
	log.SetOutput(os.Stderr)
	log.SetPrefix("[mcp] ")

	cfg := config.Load()
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	// Garante que o usuário "MCP Assistant" exista (idempotente).
	if err := database.Seed(db); err != nil {
		log.Printf("seed warning: %v", err)
	}

	tools, err := mcptools.New(db)
	if err != nil {
		log.Fatalf("tools init: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	// === Consulta ===
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_items",
		Description: "Lista itens do inventário com filtros opcionais (busca textual, categoria, local) e paginação. Retorna até 50 itens por página.",
	}, tools.ListItems)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_item",
		Description: "Busca um item específico pelo id ou pelo code (ex: EST-AB12CD34). Retorna todos os campos + caminho completo da localização.",
	}, tools.GetItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_item_location",
		Description: "Responde 'onde está X?'. Faz busca parcial em nome, código, marca, modelo e descrição. Retorna até 20 matches com o caminho completo de onde cada um está armazenado.",
	}, tools.FindItemLocation)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_categories",
		Description: "Lista todas as categorias com a contagem de itens em cada uma.",
	}, tools.ListCategories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_locations",
		Description: "Lista todas as localizações com seus caminhos completos (ex: 'Garagem > Caixa Ferramentas') e contagem de itens.",
	}, tools.ListLocations)

	// === Criação ===
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_item",
		Description: "Cria um novo item no inventário. Gera código SKU automaticamente. Se location_id for fornecido, registra movement inicial.",
	}, tools.CreateItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_category",
		Description: "Cria uma nova categoria. Pode ter parent_id para hierarquia.",
	}, tools.CreateCategory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_location",
		Description: "Cria uma nova localização. Tipos válidos: comodo, movel, caixa, armario, outro. Pode ter parent_id para hierarquia (ex: 'Caixa' dentro de 'Garagem').",
	}, tools.CreateLocation)

	// === Atualização / movimentação ===
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_item",
		Description: "Atualiza campos de um item existente (parcial — só os campos enviados são alterados). Se location_id mudar, registra movement automaticamente.",
	}, tools.UpdateItem)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_item",
		Description: "Move um item para outra localização. Atalho dedicado para movimentação que registra entry em movements com motivo configurável.",
	}, tools.MoveItem)

	log.Printf("homeestoque-mcp v%s pronto (db=%s, mcp_user_id=%d)",
		serverVersion, cfg.DBPath, tools.MCPUserID)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server.Run: %v", err)
	}
}
