package mcptools_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/mcptools"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

// newTools cria um Tools com DB seedado e MCP user já registrado.
func newTools(t *testing.T) *mcptools.Tools {
	t.Helper()
	db := testutil.NewSeededTestDB(t)
	tools, err := mcptools.New(db)
	if err != nil {
		t.Fatalf("mcptools.New: %v", err)
	}
	return tools
}

// =================== list_items ===================

func TestMCP_ListItems_ReturnsStructuredResult(t *testing.T) {
	tools := newTools(t)
	for i := 0; i < 5; i++ {
		testutil.CreateItem(t, tools.DB, "I", testutil.ItemOpts{})
	}

	_, raw, err := tools.ListItems(context.Background(), nil, mcptools.ListItemsArgs{})
	require.NoError(t, err)

	out, ok := raw.(mcptools.ListItemsResult)
	require.True(t, ok, "deve retornar ListItemsResult (objeto, não array)")
	assert.Equal(t, 5, out.Total)
	assert.Equal(t, 1, out.Page)
	assert.Equal(t, 20, out.Limit, "default = 20")
}

func TestMCP_ListItems_LimitCappedAt50(t *testing.T) {
	tools := newTools(t)
	_, raw, err := tools.ListItems(context.Background(), nil, mcptools.ListItemsArgs{Limit: 999})
	require.NoError(t, err)
	out := raw.(mcptools.ListItemsResult)
	assert.Equal(t, 50, out.Limit, "limit deve ser capado a 50")
}

func TestMCP_ListItems_FiltersBySearch(t *testing.T) {
	tools := newTools(t)
	testutil.CreateItem(t, tools.DB, "Furadeira Bosch", testutil.ItemOpts{Brand: "Bosch"})
	testutil.CreateItem(t, tools.DB, "Martelo", testutil.ItemOpts{Brand: "Tramontina"})

	_, raw, err := tools.ListItems(context.Background(), nil, mcptools.ListItemsArgs{Search: "Bosch"})
	require.NoError(t, err)
	out := raw.(mcptools.ListItemsResult)
	assert.Equal(t, 1, out.Total)
}

// =================== get_item ===================

func TestMCP_GetItem_ByID_ReturnsLocationPath(t *testing.T) {
	tools := newTools(t)
	garagem := testutil.CreateLocation(t, tools.DB, "Garagem", "comodo", nil)
	caixa := testutil.CreateLocation(t, tools.DB, "Caixa", "caixa", &garagem)
	itemID := testutil.CreateItem(t, tools.DB, "Furadeira", testutil.ItemOpts{LocationID: &caixa})

	_, raw, err := tools.GetItem(context.Background(), nil, mcptools.GetItemArgs{ID: &itemID})
	require.NoError(t, err)
	item := raw.(mcptools.ItemSummary)
	assert.Equal(t, "Furadeira", item.Name)
	assert.Equal(t, "Garagem > Caixa", item.LocationPath)
}

func TestMCP_GetItem_ByCode(t *testing.T) {
	tools := newTools(t)
	itemID := testutil.CreateItem(t, tools.DB, "X", testutil.ItemOpts{})
	// Lê o code gerado pelo testutil
	var code string
	require.NoError(t, tools.DB.QueryRow(`SELECT code FROM items WHERE id = ?`, itemID).Scan(&code))

	_, raw, err := tools.GetItem(context.Background(), nil, mcptools.GetItemArgs{Code: code})
	require.NoError(t, err)
	item := raw.(mcptools.ItemSummary)
	assert.Equal(t, itemID, item.ID)
}

func TestMCP_GetItem_Nonexistent_ReturnsError(t *testing.T) {
	tools := newTools(t)
	noID := int64(99999)
	_, _, err := tools.GetItem(context.Background(), nil, mcptools.GetItemArgs{ID: &noID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "não encontrado")
}

func TestMCP_GetItem_NeitherIDNorCode_ReturnsError(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.GetItem(context.Background(), nil, mcptools.GetItemArgs{})
	require.Error(t, err)
}

// =================== find_item_location ===================

func TestMCP_FindItemLocation_FuzzySearchMultipleFields(t *testing.T) {
	tools := newTools(t)
	loc := testutil.CreateLocation(t, tools.DB, "Caixa Ferramentas", "caixa", nil)
	testutil.CreateItem(t, tools.DB, "Furadeira de Impacto", testutil.ItemOpts{LocationID: &loc})

	_, raw, err := tools.FindItemLocation(context.Background(), nil, mcptools.FindItemLocationArgs{Query: "furadeira"})
	require.NoError(t, err)
	out := raw.(mcptools.FindItemLocationResult)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, "Caixa Ferramentas", out.Matches[0].LocationPath)
}

func TestMCP_FindItemLocation_NoLocation_ShowsPlaceholder(t *testing.T) {
	tools := newTools(t)
	testutil.CreateItem(t, tools.DB, "Sem local", testutil.ItemOpts{})

	_, raw, err := tools.FindItemLocation(context.Background(), nil, mcptools.FindItemLocationArgs{Query: "sem local"})
	require.NoError(t, err)
	out := raw.(mcptools.FindItemLocationResult)
	require.Equal(t, 1, out.Total)
	assert.Equal(t, "(sem localização definida)", out.Matches[0].LocationPath)
}

func TestMCP_FindItemLocation_EmptyQuery_ReturnsError(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.FindItemLocation(context.Background(), nil, mcptools.FindItemLocationArgs{Query: "  "})
	require.Error(t, err)
}

// =================== create_item ===================

func TestMCP_CreateItem_GeneratesCodeAndAppliesDefaults(t *testing.T) {
	tools := newTools(t)

	_, raw, err := tools.CreateItem(context.Background(), nil, mcptools.CreateItemArgs{Name: "Novo"})
	require.NoError(t, err)
	item := raw.(mcptools.ItemSummary)
	assert.NotEmpty(t, item.Code)
	assert.Equal(t, "Novo", item.Name)
	assert.Equal(t, 1, item.Quantity)
	assert.Equal(t, "un", item.Unit)
	assert.Equal(t, "novo", item.Condition)
}

func TestMCP_CreateItem_WithLocation_RegistersMCPMovement(t *testing.T) {
	tools := newTools(t)
	loc := testutil.CreateLocation(t, tools.DB, "Local", "comodo", nil)

	_, raw, err := tools.CreateItem(context.Background(), nil, mcptools.CreateItemArgs{
		Name:       "Item",
		LocationID: &loc,
	})
	require.NoError(t, err)
	item := raw.(mcptools.ItemSummary)

	// Verifica que o movement foi criado com user_id == MCP user
	var userID int64
	require.NoError(t, tools.DB.QueryRow(
		`SELECT user_id FROM movements WHERE item_id = ? AND to_location_id = ?`,
		item.ID, loc,
	).Scan(&userID))
	assert.Equal(t, tools.MCPUserID, userID, "movement deve ter user_id do MCP Assistant")
}

func TestMCP_CreateItem_InvalidCategoryRef_ReturnsError(t *testing.T) {
	tools := newTools(t)
	badCat := int64(99999)

	_, _, err := tools.CreateItem(context.Background(), nil, mcptools.CreateItemArgs{
		Name:       "X",
		CategoryID: &badCat,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "category_id")
}

func TestMCP_CreateItem_EmptyName_ReturnsError(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.CreateItem(context.Background(), nil, mcptools.CreateItemArgs{Name: " "})
	require.Error(t, err)
}

// =================== update_item ===================

func TestMCP_UpdateItem_PartialPreservesUnchanged(t *testing.T) {
	tools := newTools(t)
	itemID := testutil.CreateItem(t, tools.DB, "Original", testutil.ItemOpts{Brand: "MarcaX", Quantity: 5})

	newName := "Renomeado"
	_, raw, err := tools.UpdateItem(context.Background(), nil, mcptools.UpdateItemArgs{
		ID:   itemID,
		Name: &newName,
		// Quantity, Brand não passados — devem preservar
	})
	require.NoError(t, err)
	item := raw.(mcptools.ItemSummary)
	assert.Equal(t, "Renomeado", item.Name)
	assert.Equal(t, "MarcaX", item.Brand, "brand não enviado deve ser preservado")
	assert.Equal(t, 5, item.Quantity, "quantity não enviada deve ser preservada")
}

func TestMCP_UpdateItem_LocationChangeRegistersMovement(t *testing.T) {
	tools := newTools(t)
	loc1 := testutil.CreateLocation(t, tools.DB, "L1", "comodo", nil)
	loc2 := testutil.CreateLocation(t, tools.DB, "L2", "comodo", nil)
	itemID := testutil.CreateItem(t, tools.DB, "X", testutil.ItemOpts{LocationID: &loc1})

	_, _, err := tools.UpdateItem(context.Background(), nil, mcptools.UpdateItemArgs{
		ID:         itemID,
		LocationID: &loc2,
	})
	require.NoError(t, err)

	var count int
	require.NoError(t, tools.DB.QueryRow(
		`SELECT COUNT(*) FROM movements WHERE item_id = ? AND from_location_id = ? AND to_location_id = ?`,
		itemID, loc1, loc2,
	).Scan(&count))
	assert.Equal(t, 1, count, "deve registrar movement quando location muda")
}

// =================== move_item ===================

func TestMCP_MoveItem_RegistersMovementWithReason(t *testing.T) {
	tools := newTools(t)
	loc1 := testutil.CreateLocation(t, tools.DB, "L1", "comodo", nil)
	loc2 := testutil.CreateLocation(t, tools.DB, "L2", "comodo", nil)
	itemID := testutil.CreateItem(t, tools.DB, "X", testutil.ItemOpts{LocationID: &loc1})

	_, _, err := tools.MoveItem(context.Background(), nil, mcptools.MoveItemArgs{
		ItemID:       itemID,
		ToLocationID: loc2,
		Reason:       "Reorganizando garagem",
	})
	require.NoError(t, err)

	var reason string
	require.NoError(t, tools.DB.QueryRow(
		`SELECT reason FROM movements WHERE item_id = ? AND to_location_id = ?`,
		itemID, loc2,
	).Scan(&reason))
	assert.Equal(t, "Reorganizando garagem", reason)
}

func TestMCP_MoveItem_DefaultReason(t *testing.T) {
	tools := newTools(t)
	loc1 := testutil.CreateLocation(t, tools.DB, "L1", "comodo", nil)
	loc2 := testutil.CreateLocation(t, tools.DB, "L2", "comodo", nil)
	itemID := testutil.CreateItem(t, tools.DB, "X", testutil.ItemOpts{LocationID: &loc1})

	_, _, err := tools.MoveItem(context.Background(), nil, mcptools.MoveItemArgs{
		ItemID:       itemID,
		ToLocationID: loc2,
	})
	require.NoError(t, err)

	var reason string
	require.NoError(t, tools.DB.QueryRow(
		`SELECT reason FROM movements WHERE item_id = ? AND to_location_id = ?`,
		itemID, loc2,
	).Scan(&reason))
	assert.Equal(t, "Movimentação via MCP", reason)
}

func TestMCP_MoveItem_SameLocation_ReturnsError(t *testing.T) {
	tools := newTools(t)
	loc := testutil.CreateLocation(t, tools.DB, "L", "comodo", nil)
	itemID := testutil.CreateItem(t, tools.DB, "X", testutil.ItemOpts{LocationID: &loc})

	_, _, err := tools.MoveItem(context.Background(), nil, mcptools.MoveItemArgs{
		ItemID:       itemID,
		ToLocationID: loc,
	})
	require.Error(t, err)
}
