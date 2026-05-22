package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neviim/homeestoque/backend/internal/locpath"
	"github.com/neviim/homeestoque/backend/internal/middleware"
	"github.com/neviim/homeestoque/backend/internal/models"
	qrcode "github.com/skip2/go-qrcode"
)

type ItemHandler struct {
	DB        *sql.DB
	UploadDir string
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := strings.TrimSpace(q.Get("search"))
	categoryID := q.Get("category_id")
	locationID := q.Get("location_id")

	limit := 12
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 {
		limit = l
	}
	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}
	offset := (page - 1) * limit

	where := ` WHERE 1=1`
	args := []interface{}{}

	if search != "" {
		where += ` AND (i.name LIKE ? OR i.description LIKE ? OR i.code LIKE ? OR i.brand LIKE ? OR i.model LIKE ?)`
		s := "%" + search + "%"
		args = append(args, s, s, s, s, s)
	}
	if categoryID != "" {
		where += ` AND i.category_id = ?`
		args = append(args, categoryID)
	}
	if locationID != "" {
		where += ` AND i.location_id = ?`
		args = append(args, locationID)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM items i LEFT JOIN categories c ON c.id = i.category_id LEFT JOIN locations l ON l.id = i.location_id` + where
	if err := h.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dataQuery := `
		SELECT i.id, i.code, i.name, COALESCE(i.description,''), COALESCE(i.brand,''),
			   COALESCE(i.model,''), COALESCE(i.serial_number,''), i.quantity, i.unit,
			   i.purchase_date, i.purchase_price, i.condition, COALESCE(i.notes,''),
			   i.category_id, i.location_id, i.created_at, i.updated_at,
			   COALESCE(c.name, '') AS category_name,
			   COALESCE(l.name, '') AS location_name
		FROM items i
		LEFT JOIN categories c ON c.id = i.category_id
		LEFT JOIN locations l ON l.id = i.location_id` + where + ` ORDER BY i.updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.DB.Query(dataQuery, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []models.Item{}
	locNames := []string{}
	for rows.Next() {
		var i models.Item
		var locName string
		if err := rows.Scan(&i.ID, &i.Code, &i.Name, &i.Description, &i.Brand, &i.Model,
			&i.SerialNumber, &i.Quantity, &i.Unit, &i.PurchaseDate, &i.PurchasePrice,
			&i.Condition, &i.Notes, &i.CategoryID, &i.LocationID,
			&i.CreatedAt, &i.UpdatedAt, &i.CategoryName, &locName); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, i)
		locNames = append(locNames, locName)
	}
	rows.Close()
	locMap := locpath.LoadLocationMap(h.DB)
	for idx := range out {
		if out[idx].LocationID != nil {
			out[idx].LocationPath = locpath.BuildFullPathFromMap(locMap, *out[idx].LocationID)
		} else {
			out[idx].LocationPath = locNames[idx]
		}
	}

	totalPages := (total + limit - 1) / limit
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":       out,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var i models.Item
	err := h.DB.QueryRow(`
		SELECT id, code, name, COALESCE(description,''), COALESCE(brand,''), COALESCE(model,''),
			   COALESCE(serial_number,''), quantity, unit, purchase_date, purchase_price,
			   condition, COALESCE(notes,''), category_id, location_id, created_at, updated_at
		FROM items WHERE id=?`, id).Scan(
		&i.ID, &i.Code, &i.Name, &i.Description, &i.Brand, &i.Model,
		&i.SerialNumber, &i.Quantity, &i.Unit, &i.PurchaseDate, &i.PurchasePrice,
		&i.Condition, &i.Notes, &i.CategoryID, &i.LocationID, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "item não encontrado")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if i.CategoryID != nil {
		_ = h.DB.QueryRow("SELECT name FROM categories WHERE id=?", *i.CategoryID).Scan(&i.CategoryName)
	}
	if i.LocationID != nil {
		i.LocationPath = locpath.BuildFullPath(h.DB, *i.LocationID)
	}
	i.Photos = h.loadPhotos(i.ID, r)
	writeJSON(w, http.StatusOK, i)
}

func (h *ItemHandler) loadPhotos(itemID int64, r *http.Request) []models.ItemPhoto {
	rows, err := h.DB.Query(`SELECT id, item_id, filename, COALESCE(original_name,''), COALESCE(size,0), created_at FROM item_photos WHERE item_id=? ORDER BY created_at`, itemID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	base := fmt.Sprintf("%s://%s", scheme, r.Host)
	out := []models.ItemPhoto{}
	for rows.Next() {
		var p models.ItemPhoto
		if err := rows.Scan(&p.ID, &p.ItemID, &p.Filename, &p.OriginalName, &p.Size, &p.CreatedAt); err == nil {
			p.URL = fmt.Sprintf("%s/uploads/%s", base, p.Filename)
			out = append(out, p)
		}
	}
	return out
}

func generateCode() string {
	return "EST-" + strings.ToUpper(uuid.New().String()[:8])
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var i models.Item
	if err := decodeJSON(r, &i); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if i.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	if i.Code == "" {
		i.Code = generateCode()
	}
	if i.Quantity == 0 {
		i.Quantity = 1
	}
	if i.Unit == "" {
		i.Unit = "un"
	}
	if i.Condition == "" {
		i.Condition = "novo"
	}
	res, err := h.DB.Exec(`INSERT INTO items
		(code, name, description, brand, model, serial_number, quantity, unit,
		 purchase_date, purchase_price, condition, notes, category_id, location_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		i.Code, i.Name, i.Description, i.Brand, i.Model, i.SerialNumber, i.Quantity, i.Unit,
		i.PurchaseDate, i.PurchasePrice, i.Condition, i.Notes, i.CategoryID, i.LocationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	i.ID = id

	if i.LocationID != nil {
		uid := middleware.GetUserID(r)
		_, _ = h.DB.Exec(`INSERT INTO movements (item_id, from_location_id, to_location_id, quantity, reason, user_id)
			VALUES (?, NULL, ?, ?, ?, ?)`, id, *i.LocationID, i.Quantity, "Cadastro inicial", uid)
	}
	writeJSON(w, http.StatusCreated, i)
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var i models.Item
	if err := decodeJSON(r, &i); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	var oldLoc *int64
	_ = h.DB.QueryRow("SELECT location_id FROM items WHERE id=?", id).Scan(&oldLoc)

	_, err := h.DB.Exec(`UPDATE items SET
		code=?, name=?, description=?, brand=?, model=?, serial_number=?, quantity=?, unit=?,
		purchase_date=?, purchase_price=?, condition=?, notes=?, category_id=?, location_id=?,
		updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		i.Code, i.Name, i.Description, i.Brand, i.Model, i.SerialNumber, i.Quantity, i.Unit,
		i.PurchaseDate, i.PurchasePrice, i.Condition, i.Notes, i.CategoryID, i.LocationID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !sameLocation(oldLoc, i.LocationID) {
		uid := middleware.GetUserID(r)
		_, _ = h.DB.Exec(`INSERT INTO movements (item_id, from_location_id, to_location_id, quantity, reason, user_id)
			VALUES (?, ?, ?, ?, ?, ?)`, id, oldLoc, i.LocationID, i.Quantity, "Movimentação manual", uid)
	}

	i.ID = id
	writeJSON(w, http.StatusOK, i)
}

func sameLocation(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	rows, _ := h.DB.Query("SELECT filename FROM item_photos WHERE item_id=?", id)
	if rows != nil {
		for rows.Next() {
			var fn string
			if err := rows.Scan(&fn); err == nil {
				_ = os.Remove(filepath.Join(h.UploadDir, fn))
			}
		}
		rows.Close()
	}

	_, err := h.DB.Exec(`DELETE FROM items WHERE id=?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *ItemHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "erro no upload (máx 20MB)")
		return
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "arquivo 'photo' não enviado")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
		writeError(w, http.StatusBadRequest, "extensão inválida")
		return
	}
	filename := uuid.New().String() + ext
	dst, err := os.Create(filepath.Join(h.UploadDir, filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer dst.Close()
	n, err := io.Copy(dst, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := h.DB.Exec(`INSERT INTO item_photos (item_id, filename, original_name, size) VALUES (?, ?, ?, ?)`,
		id, filename, header.Filename, n)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	pid, _ := res.LastInsertId()

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	writeJSON(w, http.StatusCreated, models.ItemPhoto{
		ID: pid, ItemID: id, Filename: filename, OriginalName: header.Filename, Size: n,
		URL:       fmt.Sprintf("%s://%s/uploads/%s", scheme, r.Host, filename),
		CreatedAt: time.Now(),
	})
}

func (h *ItemHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseInt(chi.URLParam(r, "photoId"), 10, 64)
	var filename string
	err := h.DB.QueryRow("SELECT filename FROM item_photos WHERE id=?", pid).Scan(&filename)
	if err != nil {
		writeError(w, http.StatusNotFound, "foto não encontrada")
		return
	}
	_ = os.Remove(filepath.Join(h.UploadDir, filename))
	_, _ = h.DB.Exec("DELETE FROM item_photos WHERE id=?", pid)
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *ItemHandler) QRCode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var code string
	err := h.DB.QueryRow("SELECT code FROM items WHERE id=?", id).Scan(&code)
	if err != nil {
		writeError(w, http.StatusNotFound, "item não encontrado")
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	content := fmt.Sprintf("%s://%s/items/%d#%s", scheme, r.Host, id, code)
	png, err := qrcode.Encode(content, qrcode.Medium, 320)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(png)
}

func (h *ItemHandler) Movements(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rows, err := h.DB.Query(`
		SELECT m.id, m.item_id, m.from_location_id, m.to_location_id, m.quantity,
			   COALESCE(m.reason,''), m.user_id, m.created_at,
			   COALESCE(fl.name,'') AS from_name, COALESCE(tl.name,'') AS to_name,
			   COALESCE(u.name,'') AS user_name
		FROM movements m
		LEFT JOIN locations fl ON fl.id = m.from_location_id
		LEFT JOIN locations tl ON tl.id = m.to_location_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE m.item_id=?
		ORDER BY m.created_at DESC`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []models.Movement{}
	for rows.Next() {
		var m models.Movement
		if err := rows.Scan(&m.ID, &m.ItemID, &m.FromLocationID, &m.ToLocationID, &m.Quantity,
			&m.Reason, &m.UserID, &m.CreatedAt, &m.FromLocationName, &m.ToLocationName, &m.UserName); err == nil {
			out = append(out, m)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
