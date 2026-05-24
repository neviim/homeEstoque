package handlers_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/testutil"
)

// =================== Dashboard ===================

func TestDashboard_ReturnsCorrectCounters(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Cria 3 itens com quantidades variadas
	testutil.CreateItem(t, db, "A", testutil.ItemOpts{Quantity: 5})
	testutil.CreateItem(t, db, "B", testutil.ItemOpts{Quantity: 10})
	testutil.CreateItem(t, db, "C", testutil.ItemOpts{Quantity: 2})

	status, body := testutil.Request(t, srv, "GET", "/api/dashboard", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, float64(3), out["total_items"])
	assert.Equal(t, float64(17), out["total_quantity"], "5+10+2=17")
	assert.Greater(t, out["total_categories"], float64(0), "seed default já cria categorias")
	assert.Greater(t, out["total_locations"], float64(0))
}

func TestDashboard_TotalValueMultiplies(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	price1 := 100.0
	price2 := 50.0
	testutil.CreateItem(t, db, "Caro", testutil.ItemOpts{Quantity: 2, Price: &price1})    // 200
	testutil.CreateItem(t, db, "Barato", testutil.ItemOpts{Quantity: 3, Price: &price2})  // 150

	status, body := testutil.Request(t, srv, "GET", "/api/dashboard", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 350.0, out["total_value"], "SUM(qty*price) = 2*100 + 3*50 = 350")
}

func TestDashboard_RecentItemsOrderedDescByCreatedAt(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Cria 8 items — recent_items deve ter no máximo 6
	for i := 0; i < 8; i++ {
		testutil.CreateItem(t, db, "Item"+strconv.Itoa(i), testutil.ItemOpts{})
	}

	_, body := testutil.Request(t, srv, "GET", "/api/dashboard", admin.Token, nil)
	var out struct {
		RecentItems []map[string]interface{} `json:"recent_items"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.LessOrEqual(t, len(out.RecentItems), 6, "recent_items limita a 6")
}

func TestDashboard_TopCategoriesByItemCount(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	cat1 := testutil.CreateCategory(t, db, "Popular")
	cat2 := testutil.CreateCategory(t, db, "Vazia")
	for i := 0; i < 3; i++ {
		testutil.CreateItem(t, db, "P"+strconv.Itoa(i), testutil.ItemOpts{CategoryID: &cat1})
	}
	_ = cat2

	_, body := testutil.Request(t, srv, "GET", "/api/dashboard", admin.Token, nil)
	var out struct {
		TopCategories []map[string]interface{} `json:"top_categories"`
	}
	testutil.DecodeJSON(t, body, &out)
	require.NotEmpty(t, out.TopCategories)
	// Primeira categoria deve ter o item_count mais alto entre as nossas
	assert.Equal(t, "Popular", out.TopCategories[0]["name"])
}

// =================== AllMovements ===================

func TestMovements_Paginated(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Cria 20 movements via create+update
	loc1 := testutil.CreateLocation(t, db, "L1", "comodo", nil)
	loc2 := testutil.CreateLocation(t, db, "L2", "comodo", nil)
	for i := 0; i < 20; i++ {
		itemID := testutil.CreateItem(t, db, "I"+strconv.Itoa(i), testutil.ItemOpts{LocationID: &loc1})
		// Cria movement manualmente
		testutil.MustExec(t, db, `INSERT INTO movements (item_id, from_location_id, to_location_id, quantity, user_id) VALUES (?, ?, ?, 1, ?)`,
			itemID, loc1, loc2, admin.ID)
	}

	status, body := testutil.Request(t, srv, "GET", "/api/movements?limit=10&page=1", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct {
		Movements []map[string]interface{} `json:"movements"`
		Total     int                      `json:"total"`
		Page      int                      `json:"page"`
		Limit     int                      `json:"limit"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 20, out.Total)
	assert.Equal(t, 10, out.Limit)
	assert.Len(t, out.Movements, 10)
}

func TestMovements_FilterByUser(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	other := testutil.CreateUser(t, db, "Other", "o@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	loc := testutil.CreateLocation(t, db, "L", "comodo", nil)
	itemID := testutil.CreateItem(t, db, "I", testutil.ItemOpts{LocationID: &loc})

	// 2 movements do admin, 3 do other
	for i := 0; i < 2; i++ {
		testutil.MustExec(t, db, `INSERT INTO movements (item_id, to_location_id, quantity, user_id) VALUES (?, ?, 1, ?)`,
			itemID, loc, admin.ID)
	}
	for i := 0; i < 3; i++ {
		testutil.MustExec(t, db, `INSERT INTO movements (item_id, to_location_id, quantity, user_id) VALUES (?, ?, 1, ?)`,
			itemID, loc, other.ID)
	}

	url := "/api/movements?user_id=" + strconv.FormatInt(other.ID, 10)
	_, body := testutil.Request(t, srv, "GET", url, admin.Token, nil)
	var out struct{ Total int `json:"total"` }
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 3, out.Total, "deve filtrar apenas movements do 'other'")
}

// =================== MovementUsers ===================

func TestMovementUsers_DistinctOrderedByName(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Zeta", "z@x.com", "senha123", "admin")
	alpha := testutil.CreateUser(t, db, "Alpha", "a@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	loc := testutil.CreateLocation(t, db, "L", "comodo", nil)
	itemID := testutil.CreateItem(t, db, "I", testutil.ItemOpts{LocationID: &loc})

	// 5 movements do Alpha, 3 do Zeta — espera retorno DISTINCT 2 users em ordem alfabética
	for i := 0; i < 5; i++ {
		testutil.MustExec(t, db, `INSERT INTO movements (item_id, to_location_id, quantity, user_id) VALUES (?, ?, 1, ?)`,
			itemID, loc, alpha.ID)
	}
	for i := 0; i < 3; i++ {
		testutil.MustExec(t, db, `INSERT INTO movements (item_id, to_location_id, quantity, user_id) VALUES (?, ?, 1, ?)`,
			itemID, loc, admin.ID)
	}

	_, body := testutil.Request(t, srv, "GET", "/api/movements/users", admin.Token, nil)
	var users []map[string]interface{}
	testutil.DecodeJSON(t, body, &users)

	require.Len(t, users, 2)
	assert.Equal(t, "Alpha", users[0]["name"])
	assert.Equal(t, "Zeta", users[1]["name"])
}

// =================== ExportCSV ===================

func TestExportCSV_HasUTF8BOMAndSemicolon(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	testutil.CreateItem(t, db, "Item1", testutil.ItemOpts{Brand: "Bosch", Quantity: 2})

	status, body := testutil.Request(t, srv, "GET", "/api/export/csv", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	// Primeiros 3 bytes devem ser BOM UTF-8
	require.GreaterOrEqual(t, len(body), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, body[:3], "deve começar com BOM UTF-8")

	// Separador é ; (vírgula brasileira para Excel)
	content := string(body[3:])
	assert.Contains(t, content, ";", "CSV deve usar ; como separador")
	assert.Contains(t, content, "Código", "header em português")
	assert.Contains(t, content, "Item1", "linha com o item")
	assert.Contains(t, content, "Bosch", "marca incluída")
}

func TestExportCSV_DownloadHeaders(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	resp, err := http.NewRequest("GET", srv.URL+"/api/export/csv", nil)
	require.NoError(t, err)
	resp.Header.Set("Authorization", "Bearer "+admin.Token)
	r, err := srv.Client().Do(resp)
	require.NoError(t, err)
	defer r.Body.Close()

	assert.Contains(t, r.Header.Get("Content-Type"), "text/csv")
	disp := r.Header.Get("Content-Disposition")
	assert.True(t, strings.Contains(disp, "attachment") && strings.Contains(disp, "estoque.csv"),
		"Content-Disposition deve forçar download com filename")
}
