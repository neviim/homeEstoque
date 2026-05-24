package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"github.com/neviim/homeestoque/backend/internal/models"
)

type ExtraHandler struct {
	DB *sql.DB
}

func (h *ExtraHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	var s models.DashboardStats
	_ = h.DB.QueryRow("SELECT COUNT(*), COALESCE(SUM(quantity),0), COALESCE(SUM(purchase_price*quantity),0) FROM items").
		Scan(&s.TotalItems, &s.TotalQuantity, &s.TotalValue)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&s.TotalCategories)
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM locations").Scan(&s.TotalLocations)

	rows, err := h.DB.Query(`
		SELECT i.id, i.code, i.name, i.quantity, i.unit, i.condition, i.created_at, i.updated_at,
			   COALESCE(c.name, ''), COALESCE(l.name, '')
		FROM items i
		LEFT JOIN categories c ON c.id = i.category_id
		LEFT JOIN locations l ON l.id = i.location_id
		ORDER BY i.created_at DESC LIMIT 6`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var i models.Item
			var locName string
			if err := rows.Scan(&i.ID, &i.Code, &i.Name, &i.Quantity, &i.Unit, &i.Condition,
				&i.CreatedAt, &i.UpdatedAt, &i.CategoryName, &locName); err == nil {
				i.LocationPath = locName
				s.RecentItems = append(s.RecentItems, i)
			}
		}
	}

	urows, err := h.DB.Query(`
		SELECT i.id, i.code, i.name, i.quantity, i.unit, i.condition, i.created_at, i.updated_at,
			   COALESCE(c.name, ''), COALESCE(l.name, '')
		FROM items i
		LEFT JOIN categories c ON c.id = i.category_id
		LEFT JOIN locations l ON l.id = i.location_id
		ORDER BY i.updated_at DESC LIMIT 5`)
	if err == nil {
		defer urows.Close()
		for urows.Next() {
			var i models.Item
			var locName string
			if err := urows.Scan(&i.ID, &i.Code, &i.Name, &i.Quantity, &i.Unit, &i.Condition,
				&i.CreatedAt, &i.UpdatedAt, &i.CategoryName, &locName); err == nil {
				i.LocationPath = locName
				s.UpdatedItems = append(s.UpdatedItems, i)
			}
		}
	}

	crows, err := h.DB.Query(`
		SELECT c.id, c.name, COALESCE(c.icon,''), COALESCE(c.color,''),
			   (SELECT COUNT(*) FROM items i WHERE i.category_id = c.id) AS cnt
		FROM categories c ORDER BY cnt DESC LIMIT 6`)
	if err == nil {
		defer crows.Close()
		for crows.Next() {
			var c models.Category
			if err := crows.Scan(&c.ID, &c.Name, &c.Icon, &c.Color, &c.ItemCount); err == nil {
				s.TopCategories = append(s.TopCategories, c)
			}
		}
	}

	if s.RecentItems == nil {
		s.RecentItems = []models.Item{}
	}
	if s.UpdatedItems == nil {
		s.UpdatedItems = []models.Item{}
	}
	if s.TopCategories == nil {
		s.TopCategories = []models.Category{}
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *ExtraHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT i.code, i.name, COALESCE(i.description,''), COALESCE(i.brand,''), COALESCE(i.model,''),
			   COALESCE(i.serial_number,''), i.quantity, i.unit, COALESCE(i.condition,''),
			   COALESCE(c.name,''), COALESCE(l.name,''),
			   COALESCE(i.purchase_date,''), COALESCE(i.purchase_price,0),
			   i.created_at
		FROM items i
		LEFT JOIN categories c ON c.id = i.category_id
		LEFT JOIN locations l ON l.id = i.location_id
		ORDER BY i.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="estoque.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.Write([]string{
		"Código", "Nome", "Descrição", "Marca", "Modelo", "Nº Série",
		"Quantidade", "Unidade", "Condição", "Categoria", "Local",
		"Data Compra", "Preço Compra", "Criado em",
	})
	for rows.Next() {
		var code, name, desc, brand, model, serial, unit, cond, cat, loc, pdate, created string
		var qty int
		var price float64
		if err := rows.Scan(&code, &name, &desc, &brand, &model, &serial, &qty, &unit, &cond, &cat, &loc, &pdate, &price, &created); err == nil {
			_ = cw.Write([]string{
				code, name, desc, brand, model, serial,
				strconv.Itoa(qty), unit, cond, cat, loc,
				pdate, fmt.Sprintf("%.2f", price), created,
			})
		}
	}
	cw.Flush()
}

func (h *ExtraHandler) AllMovements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 15
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 {
		limit = l
	}
	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}
	offset := (page - 1) * limit

	where := ""
	filterArgs := []any{}
	if uid := q.Get("user_id"); uid != "" {
		where = " WHERE m.user_id = ?"
		filterArgs = append(filterArgs, uid)
	}

	var total int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM movements m"+where, filterArgs...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := h.DB.Query(`
		SELECT m.id, m.item_id, m.from_location_id, m.to_location_id, m.quantity,
			   COALESCE(m.reason,''), m.user_id, m.created_at,
			   COALESCE(fl.name,''), COALESCE(tl.name,''), COALESCE(u.name,''), COALESCE(i.name,'')
		FROM movements m
		LEFT JOIN locations fl ON fl.id = m.from_location_id
		LEFT JOIN locations tl ON tl.id = m.to_location_id
		LEFT JOIN users u ON u.id = m.user_id
		LEFT JOIN items i ON i.id = m.item_id`+
		where+
		" ORDER BY m.created_at DESC LIMIT ? OFFSET ?",
		append(filterArgs, limit, offset)...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []models.Movement{}
	for rows.Next() {
		var m models.Movement
		if err := rows.Scan(&m.ID, &m.ItemID, &m.FromLocationID, &m.ToLocationID, &m.Quantity,
			&m.Reason, &m.UserID, &m.CreatedAt,
			&m.FromLocationName, &m.ToLocationName, &m.UserName, &m.ItemName); err == nil {
			out = append(out, m)
		}
	}

	totalPages := (total + limit - 1) / limit
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"movements":   out,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func (h *ExtraHandler) MovementUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT DISTINCT u.id, u.name
		FROM movements m
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id IS NOT NULL
		ORDER BY u.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type userRef struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	users := []userRef{}
	for rows.Next() {
		var u userRef
		if err := rows.Scan(&u.ID, &u.Name); err == nil {
			users = append(users, u)
		}
	}
	writeJSON(w, http.StatusOK, users)
}
