package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/middleware"
)

type UserHandler struct {
	DB *sql.DB
}

// roleExists indica se o nome de role passado existe na tabela roles.
func (h *UserHandler) roleExists(name string) bool {
	var n int
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM roles WHERE name = ?`, name).Scan(&n)
	return n > 0
}

type userRow struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(
		`SELECT id, name, email, role, status, created_at FROM users
		 WHERE email != 'mcp@homeestoque.local' ORDER BY created_at ASC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar usuários")
		return
	}
	defer rows.Close()
	users := []userRow{}
	for rows.Next() {
		var u userRow
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

type createUserAdminRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserAdminRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Role == "" {
		req.Role = "user"
	}
	if !h.roleExists(req.Role) {
		writeError(w, http.StatusBadRequest, "perfil inexistente")
		return
	}
	if req.Name == "" || req.Email == "" || len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "nome, email e senha (mín. 6) são obrigatórios")
		return
	}
	var exists int
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&exists)
	if exists > 0 {
		writeError(w, http.StatusConflict, "email já cadastrado")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}
	res, err := h.DB.Exec(
		"INSERT INTO users (name, email, password_hash, role, status) VALUES (?, ?, ?, ?, 'active')",
		req.Name, req.Email, hash, req.Role,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	id, _ := res.LastInsertId()
	var createdAt string
	_ = h.DB.QueryRow("SELECT created_at FROM users WHERE id = ?", id).Scan(&createdAt)
	writeJSON(w, http.StatusCreated, userRow{
		ID: id, Name: req.Name, Email: req.Email, Role: req.Role, Status: "active", CreatedAt: createdAt,
	})
}

type updateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	targetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	callerID := middleware.GetUserID(r)
	if targetID == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Role == "" {
		req.Role = "user"
	}
	if !h.roleExists(req.Role) {
		writeError(w, http.StatusBadRequest, "perfil inexistente")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}

	// Impede que o último admin seja rebaixado (caller só pode tirar seu próprio admin
	// se houver outro admin ativo)
	if req.Role != "admin" && targetID == callerID {
		var adminCount int
		_ = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'").Scan(&adminCount)
		if adminCount <= 1 {
			writeError(w, http.StatusConflict, "não é possível remover o último administrador")
			return
		}
	}

	_, err := h.DB.Exec("UPDATE users SET name = ?, role = ? WHERE id = ?", req.Name, req.Role, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar usuário")
		return
	}
	var u userRow
	_ = h.DB.QueryRow("SELECT id, name, email, role, status, created_at FROM users WHERE id = ?", targetID).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt)
	writeJSON(w, http.StatusOK, u)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	targetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	callerID := middleware.GetUserID(r)
	if targetID == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if targetID == callerID {
		writeError(w, http.StatusBadRequest, "não é possível alterar o próprio status")
		return
	}

	var req updateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Status != "active" && req.Status != "inactive" && req.Status != "pending" {
		writeError(w, http.StatusBadRequest, "status inválido")
		return
	}

	// Impede inativar o último admin
	if req.Status != "active" {
		var role string
		_ = h.DB.QueryRow("SELECT role FROM users WHERE id = ?", targetID).Scan(&role)
		if role == "admin" {
			var adminCount int
			_ = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'").Scan(&adminCount)
			if adminCount <= 1 {
				writeError(w, http.StatusConflict, "não é possível inativar o último administrador")
				return
			}
		}
	}

	_, err := h.DB.Exec("UPDATE users SET status = ? WHERE id = ?", req.Status, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar status")
		return
	}
	var u userRow
	_ = h.DB.QueryRow("SELECT id, name, email, role, status, created_at FROM users WHERE id = ?", targetID).
		Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt)
	writeJSON(w, http.StatusOK, u)
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	targetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if targetID == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var req resetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "senha deve ter pelo menos 6 caracteres")
		return
	}
	var exists int
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", targetID).Scan(&exists)
	if exists == 0 {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}
	if _, err := h.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, targetID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao redefinir senha")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "senha redefinida com sucesso"})
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	targetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	callerID := middleware.GetUserID(r)
	if targetID == 0 {
		writeError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if targetID == callerID {
		writeError(w, http.StatusBadRequest, "não é possível excluir a própria conta")
		return
	}

	var role string
	err := h.DB.QueryRow("SELECT role FROM users WHERE id = ?", targetID).Scan(&role)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}
	if role == "admin" {
		var adminCount int
		_ = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' AND status = 'active'").Scan(&adminCount)
		if adminCount <= 1 {
			writeError(w, http.StatusConflict, "não é possível excluir o último administrador")
			return
		}
	}

	if _, err := h.DB.Exec("DELETE FROM users WHERE id = ?", targetID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao excluir usuário")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "usuário excluído"})
}
