package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/testutil"
)

// itemPath para endpoints /api/items/...
func itemPath(id int64, suffix ...string) string {
	p := "/api/items/" + strconv.FormatInt(id, 10)
	for _, s := range suffix {
		p += s
	}
	return p
}

// =================== List ===================

func TestItems_List_DefaultPagination(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Cria 15 items para testar a paginação default (limit=12)
	for i := 0; i < 15; i++ {
		testutil.CreateItem(t, db, "Item"+strconv.Itoa(i), testutil.ItemOpts{})
	}

	status, body := testutil.Request(t, srv, "GET", "/api/items", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct {
		Items      []map[string]interface{} `json:"items"`
		Total      int                      `json:"total"`
		Page       int                      `json:"page"`
		Limit      int                      `json:"limit"`
		TotalPages int                      `json:"total_pages"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 15, out.Total)
	assert.Equal(t, 1, out.Page)
	assert.Equal(t, 12, out.Limit, "default limit é 12")
	assert.Len(t, out.Items, 12)
	assert.Equal(t, 2, out.TotalPages)
}

func TestItems_List_SearchFiltersByMultipleFields(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	testutil.CreateItem(t, db, "Furadeira", testutil.ItemOpts{Brand: "Bosch"})
	testutil.CreateItem(t, db, "Martelo", testutil.ItemOpts{Brand: "Tramontina"})

	// Busca pelo brand
	status, body := testutil.Request(t, srv, "GET", "/api/items?search=Bosch", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, "Furadeira", out.Items[0]["name"])
}

func TestItems_List_FilterByCategory(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	cat1 := testutil.CreateCategory(t, db, "TestCat1")
	cat2 := testutil.CreateCategory(t, db, "TestCat2")
	testutil.CreateItem(t, db, "A", testutil.ItemOpts{CategoryID: &cat1})
	testutil.CreateItem(t, db, "B", testutil.ItemOpts{CategoryID: &cat1})
	testutil.CreateItem(t, db, "C", testutil.ItemOpts{CategoryID: &cat2})

	url := "/api/items?category_id=" + strconv.FormatInt(cat1, 10)
	status, body := testutil.Request(t, srv, "GET", url, admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct{ Total int `json:"total"` }
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 2, out.Total)
}

func TestItems_List_NonexistentFilter_ReturnsEmpty(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "GET", "/api/items?category_id=99999", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)
	var out struct{ Total int `json:"total"` }
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, 0, out.Total)
}

// =================== Create ===================

func TestItems_Create_GeneratesCode(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/items", admin.Token, map[string]interface{}{
		"name": "Nova Furadeira",
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	code := out["code"].(string)
	assert.True(t, strings.HasPrefix(code, "EST-"), "code deve começar com EST-, got %q", code)
	assert.Len(t, code, 12, "EST-XXXXXXXX = 12 chars")
}

func TestItems_Create_AppliesDefaults(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	_, body := testutil.Request(t, srv, "POST", "/api/items", admin.Token, map[string]interface{}{
		"name": "X",
	})
	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, float64(1), out["quantity"], "default quantity=1")
	assert.Equal(t, "un", out["unit"])
	assert.Equal(t, "novo", out["condition"])
}

func TestItems_Create_RegistersInitialMovement(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	locID := testutil.CreateLocation(t, db, "Garagem", "comodo", nil)

	status, body := testutil.Request(t, srv, "POST", "/api/items", admin.Token, map[string]interface{}{
		"name":        "Item com local",
		"location_id": locID,
	})
	require.Equal(t, http.StatusCreated, status)

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	itemID := int64(out["id"].(float64))

	// Deve haver 1 movement: from=NULL, to=locID
	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM movements WHERE item_id = ? AND from_location_id IS NULL AND to_location_id = ?`,
		itemID, locID,
	).Scan(&count))
	assert.Equal(t, 1, count, "create com location deve registrar movement inicial")
}

func TestItems_Create_EmptyName_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/items", admin.Token, map[string]interface{}{
		"name": "",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

// =================== Get ===================

func TestItems_Get_ReturnsLocationPath(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	garagem := testutil.CreateLocation(t, db, "Garagem", "comodo", nil)
	bancada := testutil.CreateLocation(t, db, "Bancada", "movel", &garagem)
	caixa := testutil.CreateLocation(t, db, "Caixa Ferramentas", "caixa", &bancada)

	itemID := testutil.CreateItem(t, db, "Furadeira", testutil.ItemOpts{LocationID: &caixa})

	status, body := testutil.Request(t, srv, "GET", itemPath(itemID), admin.Token, nil)
	require.Equal(t, http.StatusOK, status)
	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "Garagem > Bancada > Caixa Ferramentas", out["location_path"])
}

func TestItems_Get_Nonexistent_Returns404(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "GET", "/api/items/99999", admin.Token, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

// =================== Update ===================

func TestItems_Update_LocationChange_RegistersMovement(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	loc1 := testutil.CreateLocation(t, db, "Local1", "comodo", nil)
	loc2 := testutil.CreateLocation(t, db, "Local2", "comodo", nil)
	itemID := testutil.CreateItem(t, db, "Móvel", testutil.ItemOpts{LocationID: &loc1})

	status, _ := testutil.Request(t, srv, "PUT", itemPath(itemID), admin.Token, map[string]interface{}{
		"name":        "Móvel",
		"code":        "EST-TEST",
		"quantity":    1,
		"unit":        "un",
		"condition":   "novo",
		"location_id": loc2,
	})
	require.Equal(t, http.StatusOK, status)

	// Deve haver movement com from=loc1, to=loc2
	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM movements WHERE item_id = ? AND from_location_id = ? AND to_location_id = ?`,
		itemID, loc1, loc2,
	).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestItems_Update_SameLocation_NoMovement(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	loc := testutil.CreateLocation(t, db, "Local", "comodo", nil)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{LocationID: &loc})

	// Conta movements antes da update (Create não dispara aqui — usamos testutil.CreateItem)
	var before int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM movements WHERE item_id = ?`, itemID).Scan(&before))

	status, _ := testutil.Request(t, srv, "PUT", itemPath(itemID), admin.Token, map[string]interface{}{
		"name":        "X renamed",
		"code":        "EST-X",
		"quantity":    2,
		"unit":        "un",
		"condition":   "bom",
		"location_id": loc, // mesma location
	})
	require.Equal(t, http.StatusOK, status)

	var after int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM movements WHERE item_id = ?`, itemID).Scan(&after))
	assert.Equal(t, before, after, "update sem mudar location não cria movement")
}

// =================== Delete ===================

func TestItems_Delete_CascadesPhotosAndMovements(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	loc := testutil.CreateLocation(t, db, "Local", "comodo", nil)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{LocationID: &loc})

	// Cria fotos e movements
	testutil.MustExec(t, db, `INSERT INTO item_photos (item_id, filename) VALUES (?, ?)`, itemID, "foto.jpg")
	testutil.MustExec(t, db, `INSERT INTO movements (item_id, to_location_id, quantity) VALUES (?, ?, 1)`, itemID, loc)

	status, _ := testutil.Request(t, srv, "DELETE", itemPath(itemID), admin.Token, nil)
	require.Equal(t, http.StatusNoContent, status)

	var photos, moves int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM item_photos WHERE item_id = ?`, itemID).Scan(&photos))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM movements WHERE item_id = ?`, itemID).Scan(&moves))
	assert.Equal(t, 0, photos)
	assert.Equal(t, 0, moves)
}

// =================== QRCode ===================

func TestItems_QRCode_ReturnsValidPNG(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{})

	// QRCode é público (sem auth)
	resp, err := http.Get(srv.URL + itemPath(itemID, "/qrcode"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	// PNG magic bytes
	require.GreaterOrEqual(t, len(body), 8)
	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, body[:8],
		"primeiros 8 bytes devem ser o magic number do PNG")
}

func TestItems_QRCode_NoAuthNeeded(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{})

	// Sem Authorization header — deve passar (rota pública)
	status, _ := testutil.Request(t, srv, "GET", itemPath(itemID, "/qrcode"), "", nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestItems_QRCode_NonexistentItem_Returns404(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "GET", "/api/items/99999/qrcode", "", nil)
	assert.Equal(t, http.StatusNotFound, status)
}

// =================== Photos ===================

func TestItems_UploadPhoto_StoresFileAndDBRow(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{})

	// Cria multipart com 1x1 PNG fake
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fileWriter, err := w.CreateFormFile("photo", "test.png")
	require.NoError(t, err)
	// Dados mínimos (não precisa ser PNG válido, só ter a extensão certa)
	_, _ = fileWriter.Write([]byte("\x89PNG\x0D\x0A\x1A\x0Aconteudo-falso-mas-suficiente"))
	w.Close()

	req, _ := http.NewRequest("POST", srv.URL+itemPath(itemID, "/photos"), body)
	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verifica linha no DB
	var photoCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM item_photos WHERE item_id = ?`, itemID).Scan(&photoCount))
	assert.Equal(t, 1, photoCount)
}

func TestItems_UploadPhoto_InvalidExtension_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{})

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fileWriter, _ := w.CreateFormFile("photo", "malicious.exe")
	_, _ = fileWriter.Write([]byte("MZ\x90\x00"))
	w.Close()

	req, _ := http.NewRequest("POST", srv.URL+itemPath(itemID, "/photos"), body)
	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, _ := srv.Client().Do(req)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestItems_DeletePhoto_RemovesDBRowAndFile(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv, uploadDir := testutil.NewTestServerWithUploadDir(t, db)
	itemID := testutil.CreateItem(t, db, "X", testutil.ItemOpts{})

	// Upload uma foto primeiro
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, _ := w.CreateFormFile("photo", "test.png")
	_, _ = fw.Write([]byte("fake"))
	w.Close()

	req, _ := http.NewRequest("POST", srv.URL+itemPath(itemID, "/photos"), body)
	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var photo struct {
		ID       int64  `json:"id"`
		Filename string `json:"filename"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	require.NoError(t, json.Unmarshal(respBody, &photo))

	// Confirma que o arquivo existe no disco
	filePath := filepath.Join(uploadDir, photo.Filename)
	require.FileExists(t, filePath)

	// DELETE a foto
	status, _ := testutil.Request(t, srv, "DELETE",
		itemPath(itemID, "/photos/"+strconv.FormatInt(photo.ID, 10)), admin.Token, nil)
	require.Equal(t, http.StatusNoContent, status)

	// Arquivo removido do disco
	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err), "arquivo deveria ter sido removido do disco")
}