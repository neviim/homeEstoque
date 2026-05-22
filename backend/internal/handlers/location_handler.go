package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/neviim/homeestoque/backend/internal/locpath"
	"github.com/neviim/homeestoque/backend/internal/models"
)

type LocationHandler struct {
	DB *sql.DB
}

func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT l.id, l.name, l.type, l.parent_id, COALESCE(l.description,''), l.created_at,
			   (SELECT COUNT(*) FROM items i WHERE i.location_id = l.id) AS item_count
		FROM locations l ORDER BY l.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []models.Location{}
	for rows.Next() {
		var l models.Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Type, &l.ParentID, &l.Description, &l.CreatedAt, &l.ItemCount); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, l)
	}
	rows.Close()
	m := locpath.LoadLocationMap(h.DB)
	for i := range out {
		out[i].FullPath = locpath.BuildFullPathFromMap(m, out[i].ID)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var l models.Location
	if err := decodeJSON(r, &l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if l.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	if l.Type == "" {
		l.Type = "outro"
	}
	res, err := h.DB.Exec(`INSERT INTO locations (name, type, parent_id, description) VALUES (?, ?, ?, ?)`,
		l.Name, l.Type, l.ParentID, l.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	l.ID = id
	l.FullPath = locpath.BuildFullPath(h.DB, id)
	writeJSON(w, http.StatusCreated, l)
}

func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var l models.Location
	if err := decodeJSON(r, &l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err := h.DB.Exec(`UPDATE locations SET name=?, type=?, parent_id=?, description=? WHERE id=?`,
		l.Name, l.Type, l.ParentID, l.Description, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	l.ID = id
	l.FullPath = locpath.BuildFullPath(h.DB, id)
	writeJSON(w, http.StatusOK, l)
}

func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, err := h.DB.Exec(`DELETE FROM locations WHERE id=?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
