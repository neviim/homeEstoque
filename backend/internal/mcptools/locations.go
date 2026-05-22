package mcptools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neviim/homeestoque/backend/internal/locpath"
	"github.com/neviim/homeestoque/backend/internal/models"
)

// ---- list_locations ----

type ListLocationsArgs struct{}

type LocationSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	ParentID    *int64 `json:"parent_id,omitempty"`
	Description string `json:"description,omitempty"`
	FullPath    string `json:"full_path"`
	ItemCount   int    `json:"item_count"`
}

type ListLocationsResult struct {
	Locations []LocationSummary `json:"locations"`
}

func (t *Tools) ListLocations(ctx context.Context, req *mcp.CallToolRequest, _ ListLocationsArgs) (*mcp.CallToolResult, any, error) {
	rows, err := t.DB.Query(`
		SELECT l.id, l.name, l.type, l.parent_id, COALESCE(l.description,''),
		       (SELECT COUNT(*) FROM items i WHERE i.location_id = l.id) AS item_count
		FROM locations l ORDER BY l.name`)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	locs := []LocationSummary{}
	for rows.Next() {
		var l LocationSummary
		if err := rows.Scan(&l.ID, &l.Name, &l.Type, &l.ParentID, &l.Description, &l.ItemCount); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		locs = append(locs, l)
	}

	// Resolve o caminho completo de cada localização (ex: "Garagem > Caixa").
	m := locpath.LoadLocationMap(t.DB)
	for i := range locs {
		locs[i].FullPath = locpath.BuildFullPathFromMap(m, locs[i].ID)
	}

	out := ListLocationsResult{Locations: locs}
	return jsonResult(out), out, nil
}

// ---- create_location ----

type CreateLocationArgs struct {
	Name        string `json:"name" jsonschema:"nome da localização (obrigatório)"`
	Type        string `json:"type,omitempty" jsonschema:"tipo: comodo, movel, caixa, armario ou outro (default: outro)"`
	ParentID    *int64 `json:"parent_id,omitempty" jsonschema:"id da localização pai para hierarquia"`
	Description string `json:"description,omitempty" jsonschema:"descrição livre"`
}

func (t *Tools) CreateLocation(ctx context.Context, req *mcp.CallToolRequest, args CreateLocationArgs) (*mcp.CallToolResult, any, error) {
	args.Name = strings.TrimSpace(args.Name)
	if args.Name == "" {
		return nil, nil, fmt.Errorf("nome é obrigatório")
	}
	if args.Type == "" {
		args.Type = "outro"
	}
	if !validLocationTypes[args.Type] {
		return nil, nil, fmt.Errorf("type inválido: %q (use: comodo, movel, caixa, armario, outro)", args.Type)
	}
	if args.ParentID != nil {
		var exists int
		_ = t.DB.QueryRow("SELECT COUNT(*) FROM locations WHERE id = ?", *args.ParentID).Scan(&exists)
		if exists == 0 {
			return nil, nil, fmt.Errorf("parent_id %d não existe", *args.ParentID)
		}
	}

	res, err := t.DB.Exec(
		`INSERT INTO locations (name, type, parent_id, description) VALUES (?, ?, ?, ?)`,
		args.Name, args.Type, args.ParentID, args.Description,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert: %w", err)
	}
	id, _ := res.LastInsertId()
	fullPath := locpath.BuildFullPath(t.DB, id)
	logOp("create_location", "id=%d name=%q path=%q", id, args.Name, fullPath)

	out := models.Location{
		ID:          id,
		Name:        args.Name,
		Type:        args.Type,
		ParentID:    args.ParentID,
		Description: args.Description,
		FullPath:    fullPath,
	}
	return jsonResult(out), out, nil
}
