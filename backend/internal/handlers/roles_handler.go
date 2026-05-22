package handlers

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/neviim/homeestoque/backend/internal/permissions"
)

type RolesHandler struct {
	DB *sql.DB
}

type roleRow struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	UserCount   int      `json:"user_count"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
}

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

// ListCatalog devolve o catálogo de permissões (para a UI renderizar a tela).
func (h *RolesHandler) ListCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"permissions": permissions.Catalog})
}

// List devolve todos os perfis com suas permissões e quantidade de usuários atribuídos.
func (h *RolesHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT r.id, r.name, r.label, COALESCE(r.description, ''), r.is_system, r.created_at,
		       (SELECT COUNT(*) FROM users WHERE users.role = r.name AND users.email != 'mcp@homeestoque.local') AS user_count
		FROM roles r
		ORDER BY r.is_system DESC, r.name ASC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar perfis")
		return
	}
	defer rows.Close()

	roles := []roleRow{}
	for rows.Next() {
		var rr roleRow
		var sys int
		if err := rows.Scan(&rr.ID, &rr.Name, &rr.Label, &rr.Description, &sys, &rr.CreatedAt, &rr.UserCount); err != nil {
			continue
		}
		rr.IsSystem = sys == 1
		perms, _ := permissions.RolePermissions(h.DB, rr.Name)
		rr.Permissions = perms
		roles = append(roles, rr)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"roles": roles})
}

type createRoleRequest struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

func (h *RolesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Name = strings.TrimSpace(strings.ToLower(req.Name))
	req.Label = strings.TrimSpace(req.Label)
	req.Description = strings.TrimSpace(req.Description)

	if !slugRe.MatchString(req.Name) {
		writeError(w, http.StatusBadRequest, "nome inválido: use letras minúsculas, números e _ (2-50 chars, começando por letra)")
		return
	}
	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "label é obrigatório")
		return
	}

	res, err := h.DB.Exec(
		`INSERT INTO roles (name, label, description, is_system) VALUES (?, ?, ?, 0)`,
		req.Name, req.Label, req.Description,
	)
	if err != nil {
		writeError(w, http.StatusConflict, "perfil já existe com esse nome")
		return
	}
	id, _ := res.LastInsertId()
	var createdAt string
	_ = h.DB.QueryRow(`SELECT created_at FROM roles WHERE id = ?`, id).Scan(&createdAt)
	writeJSON(w, http.StatusCreated, roleRow{
		ID: id, Name: req.Name, Label: req.Label, Description: req.Description,
		IsSystem: false, UserCount: 0, Permissions: []string{}, CreatedAt: createdAt,
	})
}

type updateRoleRequest struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

func (h *RolesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if id == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var current struct {
		Name     string
		IsSystem int
	}
	err := h.DB.QueryRow(`SELECT name, is_system FROM roles WHERE id = ?`, id).Scan(&current.Name, &current.IsSystem)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "perfil não encontrado")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar perfil")
		return
	}

	var req updateRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Name = strings.TrimSpace(strings.ToLower(req.Name))
	req.Label = strings.TrimSpace(req.Label)
	req.Description = strings.TrimSpace(req.Description)

	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "label é obrigatório")
		return
	}

	// Perfis de sistema (admin) não podem ter o name alterado
	newName := current.Name
	if !boolFromInt(current.IsSystem) && req.Name != "" && req.Name != current.Name {
		if !slugRe.MatchString(req.Name) {
			writeError(w, http.StatusBadRequest, "nome inválido")
			return
		}
		newName = req.Name
	}

	tx, err := h.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao iniciar transação")
		return
	}
	defer tx.Rollback()

	if newName != current.Name {
		if _, err := tx.Exec(`UPDATE roles SET name = ?, label = ?, description = ? WHERE id = ?`, newName, req.Label, req.Description, id); err != nil {
			writeError(w, http.StatusConflict, "nome de perfil já em uso")
			return
		}
		if _, err := tx.Exec(`UPDATE users SET role = ? WHERE role = ?`, newName, current.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "erro ao migrar usuários")
			return
		}
	} else {
		if _, err := tx.Exec(`UPDATE roles SET label = ?, description = ? WHERE id = ?`, req.Label, req.Description, id); err != nil {
			writeError(w, http.StatusInternalServerError, "erro ao atualizar perfil")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar")
		return
	}

	h.returnRole(w, id)
}

func (h *RolesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if id == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var name string
	var isSystem int
	err := h.DB.QueryRow(`SELECT name, is_system FROM roles WHERE id = ?`, id).Scan(&name, &isSystem)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "perfil não encontrado")
		return
	}
	if isSystem == 1 {
		writeError(w, http.StatusForbidden, "perfil de sistema não pode ser excluído")
		return
	}
	var userCount int
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, name).Scan(&userCount)
	if userCount > 0 {
		writeError(w, http.StatusConflict, "perfil possui usuários atribuídos; reatribua-os antes de excluir")
		return
	}
	if _, err := h.DB.Exec(`DELETE FROM roles WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir perfil")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "perfil excluído"})
}

type updatePermsRequest struct {
	Permissions []string `json:"permissions"`
}

// UpdatePermissions substitui completamente o conjunto de permissões do perfil.
// Para o perfil admin (is_system=1), força a inclusão de TODAS as permissões do
// catálogo — admin nunca pode perder permissão.
func (h *RolesHandler) UpdatePermissions(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if id == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var name string
	var isSystem int
	err := h.DB.QueryRow(`SELECT name, is_system FROM roles WHERE id = ?`, id).Scan(&name, &isSystem)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "perfil não encontrado")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar perfil")
		return
	}

	var req updatePermsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	// Validação: cada key deve existir no catálogo
	seen := map[string]bool{}
	clean := []string{}
	for _, k := range req.Permissions {
		k = strings.TrimSpace(k)
		if k == "" || seen[k] {
			continue
		}
		if !permissions.Exists(k) {
			writeError(w, http.StatusBadRequest, "permissão desconhecida: "+k)
			return
		}
		seen[k] = true
		clean = append(clean, k)
	}

	// Admin sempre tem TODAS — ignora o que veio e força catálogo completo
	if isSystem == 1 && name == "admin" {
		clean = permissions.Keys()
	}

	tx, err := h.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao iniciar transação")
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM role_permissions WHERE role_id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao limpar permissões")
		return
	}
	stmt, err := tx.Prepare(`INSERT INTO role_permissions (role_id, permission) VALUES (?, ?)`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao preparar insert")
		return
	}
	defer stmt.Close()
	for _, k := range clean {
		if _, err := stmt.Exec(id, k); err != nil {
			writeError(w, http.StatusInternalServerError, "erro ao salvar permissão")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao salvar")
		return
	}

	h.returnRole(w, id)
}

func (h *RolesHandler) returnRole(w http.ResponseWriter, id int64) {
	var rr roleRow
	var sys int
	err := h.DB.QueryRow(`
		SELECT r.id, r.name, r.label, COALESCE(r.description, ''), r.is_system, r.created_at,
		       (SELECT COUNT(*) FROM users WHERE users.role = r.name AND users.email != 'mcp@homeestoque.local')
		FROM roles r WHERE r.id = ?
	`, id).Scan(&rr.ID, &rr.Name, &rr.Label, &rr.Description, &sys, &rr.CreatedAt, &rr.UserCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao buscar perfil")
		return
	}
	rr.IsSystem = sys == 1
	perms, _ := permissions.RolePermissions(h.DB, rr.Name)
	rr.Permissions = perms
	writeJSON(w, http.StatusOK, rr)
}

func boolFromInt(i int) bool { return i != 0 }
