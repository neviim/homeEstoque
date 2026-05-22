package mcptools

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neviim/homeestoque/backend/internal/locpath"
)

// ItemSummary é o formato de item exposto pelas tools MCP. Achata os campos
// principais e inclui o caminho legível da localização — formato pensado
// para consumo direto pelo LLM.
type ItemSummary struct {
	ID            int64    `json:"id"`
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Brand         string   `json:"brand,omitempty"`
	Model         string   `json:"model,omitempty"`
	SerialNumber  string   `json:"serial_number,omitempty"`
	Quantity      int      `json:"quantity"`
	Unit          string   `json:"unit"`
	Condition     string   `json:"condition"`
	Notes         string   `json:"notes,omitempty"`
	PurchaseDate  *string  `json:"purchase_date,omitempty"`
	PurchasePrice *float64 `json:"purchase_price,omitempty"`
	CategoryID    *int64   `json:"category_id,omitempty"`
	CategoryName  string   `json:"category_name,omitempty"`
	LocationID    *int64   `json:"location_id,omitempty"`
	LocationPath  string   `json:"location_path,omitempty"`
}

// ---- list_items ----

type ListItemsArgs struct {
	Search     string `json:"search,omitempty" jsonschema:"termo de busca parcial em nome, descrição, código, marca ou modelo"`
	CategoryID *int64 `json:"category_id,omitempty" jsonschema:"filtrar por categoria"`
	LocationID *int64 `json:"location_id,omitempty" jsonschema:"filtrar por localização"`
	Page       int    `json:"page,omitempty" jsonschema:"página (1-indexed, default 1)"`
	Limit      int    `json:"limit,omitempty" jsonschema:"itens por página (default 20, máximo 50)"`
}

type ListItemsResult struct {
	Items      []ItemSummary `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

func (t *Tools) ListItems(ctx context.Context, req *mcp.CallToolRequest, args ListItemsArgs) (*mcp.CallToolResult, any, error) {
	if args.Page <= 0 {
		args.Page = 1
	}
	if args.Limit <= 0 {
		args.Limit = 20
	}
	if args.Limit > 50 {
		args.Limit = 50 // teto para não estourar contexto do LLM
	}

	where := " WHERE 1=1"
	qargs := []any{}
	if s := strings.TrimSpace(args.Search); s != "" {
		where += " AND (i.name LIKE ? OR i.description LIKE ? OR i.code LIKE ? OR i.brand LIKE ? OR i.model LIKE ?)"
		like := "%" + s + "%"
		qargs = append(qargs, like, like, like, like, like)
	}
	if args.CategoryID != nil {
		where += " AND i.category_id = ?"
		qargs = append(qargs, *args.CategoryID)
	}
	if args.LocationID != nil {
		where += " AND i.location_id = ?"
		qargs = append(qargs, *args.LocationID)
	}

	var total int
	if err := t.DB.QueryRow("SELECT COUNT(*) FROM items i"+where, qargs...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count: %w", err)
	}

	offset := (args.Page - 1) * args.Limit
	rows, err := t.DB.Query(
		`SELECT i.id, i.code, i.name, COALESCE(i.description,''), COALESCE(i.brand,''),
		        COALESCE(i.model,''), COALESCE(i.serial_number,''), i.quantity, i.unit,
		        i.condition, COALESCE(i.notes,''), i.purchase_date, i.purchase_price,
		        i.category_id, i.location_id,
		        COALESCE(c.name,'') AS category_name
		 FROM items i
		 LEFT JOIN categories c ON c.id = i.category_id`+where+
			" ORDER BY i.updated_at DESC LIMIT ? OFFSET ?",
		append(qargs, args.Limit, offset)...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	items := []ItemSummary{}
	for rows.Next() {
		var s ItemSummary
		if err := rows.Scan(
			&s.ID, &s.Code, &s.Name, &s.Description, &s.Brand, &s.Model, &s.SerialNumber,
			&s.Quantity, &s.Unit, &s.Condition, &s.Notes, &s.PurchaseDate, &s.PurchasePrice,
			&s.CategoryID, &s.LocationID, &s.CategoryName,
		); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, s)
	}

	// Resolve caminhos de localização em batch.
	m := locpath.LoadLocationMap(t.DB)
	for i := range items {
		if items[i].LocationID != nil {
			items[i].LocationPath = locpath.BuildFullPathFromMap(m, *items[i].LocationID)
		}
	}

	totalPages := (total + args.Limit - 1) / args.Limit
	out := ListItemsResult{Items: items, Total: total, Page: args.Page, Limit: args.Limit, TotalPages: totalPages}
	return jsonResult(out), out, nil
}

// ---- get_item ----

type GetItemArgs struct {
	ID   *int64 `json:"id,omitempty" jsonschema:"id numérico do item"`
	Code string `json:"code,omitempty" jsonschema:"código alfanumérico do item (ex: EST-AB12CD34)"`
}

func (t *Tools) GetItem(ctx context.Context, req *mcp.CallToolRequest, args GetItemArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == nil && strings.TrimSpace(args.Code) == "" {
		return nil, nil, fmt.Errorf("informe id ou code")
	}

	var (
		row *sql.Row
		q   = `SELECT i.id, i.code, i.name, COALESCE(i.description,''), COALESCE(i.brand,''),
		             COALESCE(i.model,''), COALESCE(i.serial_number,''), i.quantity, i.unit,
		             i.condition, COALESCE(i.notes,''), i.purchase_date, i.purchase_price,
		             i.category_id, i.location_id, COALESCE(c.name,'')
		      FROM items i LEFT JOIN categories c ON c.id = i.category_id`
	)
	if args.ID != nil {
		row = t.DB.QueryRow(q+" WHERE i.id = ?", *args.ID)
	} else {
		row = t.DB.QueryRow(q+" WHERE i.code = ?", strings.TrimSpace(args.Code))
	}

	var s ItemSummary
	err := row.Scan(
		&s.ID, &s.Code, &s.Name, &s.Description, &s.Brand, &s.Model, &s.SerialNumber,
		&s.Quantity, &s.Unit, &s.Condition, &s.Notes, &s.PurchaseDate, &s.PurchasePrice,
		&s.CategoryID, &s.LocationID, &s.CategoryName,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("item não encontrado")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	if s.LocationID != nil {
		s.LocationPath = locpath.BuildFullPath(t.DB, *s.LocationID)
	}
	return jsonResult(s), s, nil
}

// ---- find_item_location ----
// Tool central pedida pelo usuário: dado um termo livre, encontra o(s) item(ns)
// que casam e retorna onde estão.

type FindItemLocationArgs struct {
	Query string `json:"query" jsonschema:"termo de busca livre (ex: 'furadeira', 'EST-AB12', 'samsung'). Faz match parcial em name, code, brand, model e description."`
}

type FoundItem struct {
	Name         string `json:"name"`
	Code         string `json:"code"`
	Quantity     int    `json:"quantity"`
	Unit         string `json:"unit"`
	LocationPath string `json:"location_path"`
	CategoryName string `json:"category_name,omitempty"`
}

type FindItemLocationResult struct {
	Query   string      `json:"query"`
	Matches []FoundItem `json:"matches"`
	Total   int         `json:"total"`
}

func (t *Tools) FindItemLocation(ctx context.Context, req *mcp.CallToolRequest, args FindItemLocationArgs) (*mcp.CallToolResult, any, error) {
	q := strings.TrimSpace(args.Query)
	if q == "" {
		return nil, nil, fmt.Errorf("query não pode ser vazia")
	}
	like := "%" + q + "%"

	rows, err := t.DB.Query(
		`SELECT i.name, i.code, i.quantity, i.unit, i.location_id, COALESCE(c.name,'')
		 FROM items i LEFT JOIN categories c ON c.id = i.category_id
		 WHERE i.name LIKE ? OR i.code LIKE ? OR i.brand LIKE ? OR i.model LIKE ? OR i.description LIKE ?
		 ORDER BY
		   CASE WHEN i.name LIKE ? THEN 0 WHEN i.code LIKE ? THEN 1 ELSE 2 END,
		   i.updated_at DESC
		 LIMIT 20`,
		like, like, like, like, like, like, like,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	results := []FoundItem{}
	locIDs := []int64{}
	for rows.Next() {
		var f FoundItem
		var locID *int64
		if err := rows.Scan(&f.Name, &f.Code, &f.Quantity, &f.Unit, &locID, &f.CategoryName); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, f)
		if locID != nil {
			locIDs = append(locIDs, *locID)
		} else {
			locIDs = append(locIDs, 0)
		}
	}

	m := locpath.LoadLocationMap(t.DB)
	for i, id := range locIDs {
		if id > 0 {
			results[i].LocationPath = locpath.BuildFullPathFromMap(m, id)
		} else {
			results[i].LocationPath = "(sem localização definida)"
		}
	}

	out := FindItemLocationResult{Query: q, Matches: results, Total: len(results)}
	return jsonResult(out), out, nil
}

// ---- create_item ----

type CreateItemArgs struct {
	Name          string   `json:"name" jsonschema:"nome do item (obrigatório)"`
	Description   string   `json:"description,omitempty"`
	Brand         string   `json:"brand,omitempty"`
	Model         string   `json:"model,omitempty"`
	SerialNumber  string   `json:"serial_number,omitempty"`
	Quantity      int      `json:"quantity,omitempty" jsonschema:"quantidade (default 1)"`
	Unit          string   `json:"unit,omitempty" jsonschema:"unidade (default 'un')"`
	PurchaseDate  string   `json:"purchase_date,omitempty" jsonschema:"data de compra YYYY-MM-DD"`
	PurchasePrice *float64 `json:"purchase_price,omitempty"`
	Condition     string   `json:"condition,omitempty" jsonschema:"condição: novo, bom, regular, ruim (default 'novo')"`
	Notes         string   `json:"notes,omitempty"`
	CategoryID    *int64   `json:"category_id,omitempty"`
	LocationID    *int64   `json:"location_id,omitempty"`
}

func (t *Tools) CreateItem(ctx context.Context, req *mcp.CallToolRequest, args CreateItemArgs) (*mcp.CallToolResult, any, error) {
	args.Name = strings.TrimSpace(args.Name)
	if args.Name == "" {
		return nil, nil, fmt.Errorf("name é obrigatório")
	}
	if args.Quantity <= 0 {
		args.Quantity = 1
	}
	if args.Unit == "" {
		args.Unit = "un"
	}
	if args.Condition == "" {
		args.Condition = "novo"
	}
	if err := validateRefs(t.DB, args.CategoryID, args.LocationID); err != nil {
		return nil, nil, err
	}

	code := generateCode()
	var pdate any
	if args.PurchaseDate != "" {
		pdate = args.PurchaseDate
	}

	res, err := t.DB.Exec(
		`INSERT INTO items (code, name, description, brand, model, serial_number, quantity, unit,
		                    purchase_date, purchase_price, condition, notes, category_id, location_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		code, args.Name, args.Description, args.Brand, args.Model, args.SerialNumber,
		args.Quantity, args.Unit, pdate, args.PurchasePrice, args.Condition, args.Notes,
		args.CategoryID, args.LocationID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert: %w", err)
	}
	id, _ := res.LastInsertId()

	if args.LocationID != nil {
		_, _ = t.DB.Exec(
			`INSERT INTO movements (item_id, from_location_id, to_location_id, quantity, reason, user_id)
			 VALUES (?, NULL, ?, ?, ?, ?)`,
			id, *args.LocationID, args.Quantity, "Cadastro via MCP", t.MCPUserID,
		)
	}
	logOp("create_item", "id=%d code=%s name=%q", id, code, args.Name)

	return t.GetItem(ctx, req, GetItemArgs{ID: &id})
}

// ---- update_item ----
// Atualização parcial: só campos presentes no input são alterados.

type UpdateItemArgs struct {
	ID            int64    `json:"id" jsonschema:"id do item a atualizar (obrigatório)"`
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Brand         *string  `json:"brand,omitempty"`
	Model         *string  `json:"model,omitempty"`
	SerialNumber  *string  `json:"serial_number,omitempty"`
	Quantity      *int     `json:"quantity,omitempty"`
	Unit          *string  `json:"unit,omitempty"`
	PurchaseDate  *string  `json:"purchase_date,omitempty"`
	PurchasePrice *float64 `json:"purchase_price,omitempty"`
	Condition     *string  `json:"condition,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	CategoryID    *int64   `json:"category_id,omitempty"`
	LocationID    *int64   `json:"location_id,omitempty"`
}

func (t *Tools) UpdateItem(ctx context.Context, req *mcp.CallToolRequest, args UpdateItemArgs) (*mcp.CallToolResult, any, error) {
	// Carrega estado atual.
	var (
		cur     CreateItemArgs
		oldLoc  *int64
		oldQty  int
		oldCode string
	)
	err := t.DB.QueryRow(
		`SELECT code, name, COALESCE(description,''), COALESCE(brand,''), COALESCE(model,''),
		        COALESCE(serial_number,''), quantity, unit, COALESCE(purchase_date,''), purchase_price,
		        condition, COALESCE(notes,''), category_id, location_id
		 FROM items WHERE id = ?`,
		args.ID,
	).Scan(
		&oldCode, &cur.Name, &cur.Description, &cur.Brand, &cur.Model, &cur.SerialNumber,
		&cur.Quantity, &cur.Unit, &cur.PurchaseDate, &cur.PurchasePrice, &cur.Condition,
		&cur.Notes, &cur.CategoryID, &cur.LocationID,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("item id=%d não encontrado", args.ID)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read: %w", err)
	}
	oldLoc = cur.LocationID
	oldQty = cur.Quantity

	// Aplica deltas.
	if args.Name != nil {
		cur.Name = *args.Name
	}
	if args.Description != nil {
		cur.Description = *args.Description
	}
	if args.Brand != nil {
		cur.Brand = *args.Brand
	}
	if args.Model != nil {
		cur.Model = *args.Model
	}
	if args.SerialNumber != nil {
		cur.SerialNumber = *args.SerialNumber
	}
	if args.Quantity != nil {
		cur.Quantity = *args.Quantity
	}
	if args.Unit != nil {
		cur.Unit = *args.Unit
	}
	if args.PurchaseDate != nil {
		cur.PurchaseDate = *args.PurchaseDate
	}
	if args.PurchasePrice != nil {
		cur.PurchasePrice = args.PurchasePrice
	}
	if args.Condition != nil {
		cur.Condition = *args.Condition
	}
	if args.Notes != nil {
		cur.Notes = *args.Notes
	}
	if args.CategoryID != nil {
		cur.CategoryID = args.CategoryID
	}
	if args.LocationID != nil {
		cur.LocationID = args.LocationID
	}

	if err := validateRefs(t.DB, cur.CategoryID, cur.LocationID); err != nil {
		return nil, nil, err
	}

	var pdate any
	if cur.PurchaseDate != "" {
		pdate = cur.PurchaseDate
	}

	_, err = t.DB.Exec(
		`UPDATE items SET name=?, description=?, brand=?, model=?, serial_number=?, quantity=?, unit=?,
		                  purchase_date=?, purchase_price=?, condition=?, notes=?, category_id=?, location_id=?,
		                  updated_at=CURRENT_TIMESTAMP
		 WHERE id = ?`,
		cur.Name, cur.Description, cur.Brand, cur.Model, cur.SerialNumber, cur.Quantity, cur.Unit,
		pdate, cur.PurchasePrice, cur.Condition, cur.Notes, cur.CategoryID, cur.LocationID,
		args.ID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update: %w", err)
	}

	// Registra movement se mudou de local.
	if !samePtr(oldLoc, cur.LocationID) {
		_, _ = t.DB.Exec(
			`INSERT INTO movements (item_id, from_location_id, to_location_id, quantity, reason, user_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			args.ID, oldLoc, cur.LocationID, cur.Quantity, "Atualização via MCP", t.MCPUserID,
		)
	}
	logOp("update_item", "id=%d code=%s qty=%d->%d", args.ID, oldCode, oldQty, cur.Quantity)

	return t.GetItem(ctx, req, GetItemArgs{ID: &args.ID})
}

// ---- move_item ----

type MoveItemArgs struct {
	ItemID       int64  `json:"item_id" jsonschema:"id do item a mover (obrigatório)"`
	ToLocationID int64  `json:"to_location_id" jsonschema:"id da localização destino (obrigatório)"`
	Quantity     int    `json:"quantity,omitempty" jsonschema:"quantidade movimentada (default: quantidade atual do item)"`
	Reason       string `json:"reason,omitempty" jsonschema:"motivo da movimentação (default: 'Movimentação via MCP')"`
}

func (t *Tools) MoveItem(ctx context.Context, req *mcp.CallToolRequest, args MoveItemArgs) (*mcp.CallToolResult, any, error) {
	if args.ItemID == 0 || args.ToLocationID == 0 {
		return nil, nil, fmt.Errorf("item_id e to_location_id são obrigatórios")
	}
	// Confirma destino existe.
	if err := validateRefs(t.DB, nil, &args.ToLocationID); err != nil {
		return nil, nil, err
	}

	var (
		oldLoc  *int64
		curQty  int
		curName string
	)
	err := t.DB.QueryRow(`SELECT location_id, quantity, name FROM items WHERE id = ?`, args.ItemID).
		Scan(&oldLoc, &curQty, &curName)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("item id=%d não encontrado", args.ItemID)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read: %w", err)
	}

	if oldLoc != nil && *oldLoc == args.ToLocationID {
		return nil, nil, fmt.Errorf("item já está nesse local")
	}

	qty := args.Quantity
	if qty <= 0 {
		qty = curQty
	}
	reason := args.Reason
	if reason == "" {
		reason = "Movimentação via MCP"
	}

	_, err = t.DB.Exec(
		`UPDATE items SET location_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		args.ToLocationID, args.ItemID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update: %w", err)
	}
	_, _ = t.DB.Exec(
		`INSERT INTO movements (item_id, from_location_id, to_location_id, quantity, reason, user_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		args.ItemID, oldLoc, args.ToLocationID, qty, reason, t.MCPUserID,
	)
	logOp("move_item", "id=%d name=%q to=%d", args.ItemID, curName, args.ToLocationID)

	return t.GetItem(ctx, req, GetItemArgs{ID: &args.ItemID})
}

// ---- helpers internos ----

func validateRefs(db *sql.DB, catID, locID *int64) error {
	if catID != nil {
		var n int
		_ = db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = ?", *catID).Scan(&n)
		if n == 0 {
			return fmt.Errorf("category_id %d não existe", *catID)
		}
	}
	if locID != nil {
		var n int
		_ = db.QueryRow("SELECT COUNT(*) FROM locations WHERE id = ?", *locID).Scan(&n)
		if n == 0 {
			return fmt.Errorf("location_id %d não existe", *locID)
		}
	}
	return nil
}

func samePtr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
