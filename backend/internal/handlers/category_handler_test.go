package handlers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func catPath(id int64) string {
	return "/api/categories/" + strconv.FormatInt(id, 10)
}

func TestCategories_List_IncludesItemCount(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	catID := testutil.CreateCategory(t, db, "TestCat")
	testutil.CreateItem(t, db, "A", testutil.ItemOpts{CategoryID: &catID})
	testutil.CreateItem(t, db, "B", testutil.ItemOpts{CategoryID: &catID})

	status, body := testutil.Request(t, srv, "GET", "/api/categories", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out []map[string]interface{}
	testutil.DecodeJSON(t, body, &out)

	var found bool
	for _, c := range out {
		if c["name"] == "TestCat" {
			assert.Equal(t, float64(2), c["item_count"])
			found = true
		}
	}
	assert.True(t, found, "categoria TestCat deveria estar na lista")
}

func TestCategories_Create_WithOptionalParent(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	parentID := testutil.CreateCategory(t, db, "Mãe")

	status, body := testutil.Request(t, srv, "POST", "/api/categories", admin.Token, map[string]interface{}{
		"name":      "Filha",
		"parent_id": parentID,
		"color":     "#ff0000",
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "Filha", out["name"])
	assert.Equal(t, float64(parentID), out["parent_id"])
}

func TestCategories_Create_EmptyName_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/categories", admin.Token, map[string]interface{}{
		"name": "",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestCategories_Update_ChangesFields(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	catID := testutil.CreateCategory(t, db, "Original")
	status, _ := testutil.Request(t, srv, "PUT", catPath(catID), admin.Token, map[string]interface{}{
		"name":  "Renomeada",
		"color": "#00ff00",
		"icon":  "icone",
	})
	require.Equal(t, http.StatusOK, status)

	var name, color, icon string
	require.NoError(t, db.QueryRow(`SELECT name, color, icon FROM categories WHERE id = ?`, catID).Scan(&name, &color, &icon))
	assert.Equal(t, "Renomeada", name)
	assert.Equal(t, "#00ff00", color)
	assert.Equal(t, "icone", icon)
}

func TestCategories_Delete_SetsNullOnItems(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	catID := testutil.CreateCategory(t, db, "ToDelete")
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{CategoryID: &catID})

	status, _ := testutil.Request(t, srv, "DELETE", catPath(catID), admin.Token, nil)
	require.Equal(t, http.StatusNoContent, status)

	// Item ainda existe, mas category_id é NULL (FK ON DELETE SET NULL)
	var catRef *int64
	require.NoError(t, db.QueryRow(`SELECT category_id FROM items WHERE id = ?`, itemID).Scan(&catRef))
	assert.Nil(t, catRef, "category_id deveria ter virado NULL após delete da categoria")
}

func TestCategories_Delete_Nonexistent_Still204(t *testing.T) {
	// DELETE de id inexistente retorna 204 (SQLite não distingue 0 rows affected)
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "DELETE", "/api/categories/99999", admin.Token, nil)
	assert.Equal(t, http.StatusNoContent, status)
}
