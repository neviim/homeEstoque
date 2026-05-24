package handlers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func locPath(id int64) string {
	return "/api/locations/" + strconv.FormatInt(id, 10)
}

func TestLocations_List_IncludesFullPath(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	garagem := testutil.CreateLocation(t, db, "TestGaragem", "comodo", nil)
	bancada := testutil.CreateLocation(t, db, "TestBancada", "movel", &garagem)

	status, body := testutil.Request(t, srv, "GET", "/api/locations", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out []map[string]interface{}
	testutil.DecodeJSON(t, body, &out)

	var found bool
	for _, l := range out {
		if l["id"] == float64(bancada) {
			assert.Equal(t, "TestGaragem > TestBancada", l["full_path"])
			found = true
		}
	}
	assert.True(t, found, "bancada deveria estar na lista com full_path resolvido")
}

func TestLocations_Create_DefaultsTypeToOutro(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/locations", admin.Token, map[string]interface{}{
		"name": "SemTipo",
		// type omitido
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "outro", out["type"])
}

func TestLocations_Create_EmptyName_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/locations", admin.Token, map[string]interface{}{
		"name": "",
		"type": "comodo",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestLocations_Create_WithParent(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	parentID := testutil.CreateLocation(t, db, "Sala", "comodo", nil)

	status, body := testutil.Request(t, srv, "POST", "/api/locations", admin.Token, map[string]interface{}{
		"name":      "Estante",
		"type":      "movel",
		"parent_id": parentID,
	})
	require.Equal(t, http.StatusCreated, status)

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "Sala > Estante", out["full_path"], "full_path deve ser resolvido na criação")
}

func TestLocations_Delete_SetsNullOnItems(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	locID := testutil.CreateLocation(t, db, "ToDelete", "comodo", nil)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{LocationID: &locID})

	status, _ := testutil.Request(t, srv, "DELETE", locPath(locID), admin.Token, nil)
	require.Equal(t, http.StatusNoContent, status)

	var locRef *int64
	require.NoError(t, db.QueryRow(`SELECT location_id FROM items WHERE id = ?`, itemID).Scan(&locRef))
	assert.Nil(t, locRef, "location_id deveria virar NULL após delete (FK ON DELETE SET NULL)")
}

func TestLocations_Delete_ChildrenLoseParent(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	parent := testutil.CreateLocation(t, db, "ParentLoc", "comodo", nil)
	child := testutil.CreateLocation(t, db, "ChildLoc", "movel", &parent)

	status, _ := testutil.Request(t, srv, "DELETE", locPath(parent), admin.Token, nil)
	require.Equal(t, http.StatusNoContent, status)

	var parentRef *int64
	require.NoError(t, db.QueryRow(`SELECT parent_id FROM locations WHERE id = ?`, child).Scan(&parentRef))
	assert.Nil(t, parentRef, "filha deveria perder parent_id após delete da pai (FK SET NULL)")
}
