package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neviim/homeestoque/backend/internal/models"
)

// ---- list_categories ----

type ListCategoriesArgs struct{}

type CategorySummary struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
	ItemCount int    `json:"item_count"`
	ParentID  *int64 `json:"parent_id,omitempty"`
}

type ListCategoriesResult struct {
	Categories []CategorySummary `json:"categories"`
}

func (t *Tools) ListCategories(ctx context.Context, req *mcp.CallToolRequest, _ ListCategoriesArgs) (*mcp.CallToolResult, any, error) {
	rows, err := t.DB.Query(`
		SELECT c.id, c.name, COALESCE(c.icon,''), COALESCE(c.color,''), c.parent_id,
		       (SELECT COUNT(*) FROM items i WHERE i.category_id = c.id) AS item_count
		FROM categories c ORDER BY c.name`)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cats := []CategorySummary{}
	for rows.Next() {
		var c CategorySummary
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.Color, &c.ParentID, &c.ItemCount); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		cats = append(cats, c)
	}

	out := ListCategoriesResult{Categories: cats}
	return jsonResult(out), out, nil
}

// ---- create_category ----

type CreateCategoryArgs struct {
	Name     string `json:"name" jsonschema:"nome da categoria (obrigatório)"`
	Icon     string `json:"icon,omitempty" jsonschema:"ícone lucide-react opcional (ex: cpu, wrench)"`
	Color    string `json:"color,omitempty" jsonschema:"cor hexadecimal opcional (ex: #3b82f6)"`
	ParentID *int64 `json:"parent_id,omitempty" jsonschema:"id da categoria pai para hierarquia, opcional"`
}

func (t *Tools) CreateCategory(ctx context.Context, req *mcp.CallToolRequest, args CreateCategoryArgs) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(args.Name) == "" {
		return nil, nil, fmt.Errorf("nome é obrigatório")
	}

	res, err := t.DB.Exec(
		`INSERT INTO categories (name, icon, color, parent_id) VALUES (?, ?, ?, ?)`,
		args.Name, args.Icon, args.Color, args.ParentID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert: %w", err)
	}
	id, _ := res.LastInsertId()
	logOp("create_category", "id=%d name=%q", id, args.Name)

	out := models.Category{
		ID:       id,
		Name:     args.Name,
		Icon:     args.Icon,
		Color:    args.Color,
		ParentID: args.ParentID,
	}
	return jsonResult(out), out, nil
}

// jsonResult serializa qualquer valor em um CallToolResult de texto JSON
// indentado — útil para que o LLM leia a resposta de forma legível.
func jsonResult(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("erro ao serializar: %v", err)}},
			IsError: true,
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}
