package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/neviim/homeestoque/backend/internal/models"
)

type CategoryHandler struct {
	DB *sql.DB
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT c.id, c.name, c.parent_id, COALESCE(c.icon, ''), COALESCE(c.color, ''), c.created_at,
			   (SELECT COUNT(*) FROM items i WHERE i.category_id = c.id) AS item_count
		FROM categories c
		ORDER BY c.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.Icon, &c.Color, &c.CreatedAt, &c.ItemCount); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var c models.Category
	if err := decodeJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if c.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	res, err := h.DB.Exec(`INSERT INTO categories (name, parent_id, icon, color) VALUES (?, ?, ?, ?)`,
		c.Name, c.ParentID, c.Icon, c.Color)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	c.ID = id
	writeJSON(w, http.StatusCreated, c)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var c models.Category
	if err := decodeJSON(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err := h.DB.Exec(`UPDATE categories SET name=?, parent_id=?, icon=?, color=? WHERE id=?`,
		c.Name, c.ParentID, c.Icon, c.Color, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.ID = id
	writeJSON(w, http.StatusOK, c)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, err := h.DB.Exec(`DELETE FROM categories WHERE id=?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
