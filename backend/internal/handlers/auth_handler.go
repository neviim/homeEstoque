package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/middleware"
	"github.com/neviim/homeestoque/backend/internal/permissions"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string                 `json:"token"`
	User  map[string]interface{} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
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

	// Primeiro usuário humano se torna admin+active; demais ficam user+pending.
	var humanCount int
	_ = h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email != 'mcp@homeestoque.local'").Scan(&humanCount)
	role, status := "user", "pending"
	if humanCount == 0 {
		role, status = "admin", "active"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}
	res, err := h.DB.Exec(
		"INSERT INTO users (name, email, password_hash, role, status) VALUES (?, ?, ?, ?, ?)",
		req.Name, req.Email, hash, role, status,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	id, _ := res.LastInsertId()

	if status == "pending" {
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"status":  "pending",
			"message": "Conta criada com sucesso. Aguardando aprovação de um administrador.",
		})
		return
	}

	token, err := auth.GenerateToken(id, req.Email, h.JWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token")
		return
	}
	var createdAt string
	_ = h.DB.QueryRow("SELECT created_at FROM users WHERE id = ?", id).Scan(&createdAt)
	perms, _ := permissions.UserPermissions(h.DB, id)
	writeJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  map[string]interface{}{"id": id, "name": req.Name, "email": req.Email, "created_at": createdAt, "role": role, "status": status, "permissions": perms},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var id int64
	var name, email, hash, createdAt, role, status string
	err := h.DB.QueryRow("SELECT id, name, email, password_hash, created_at, role, status FROM users WHERE email = ?", req.Email).
		Scan(&id, &name, &email, &hash, &createdAt, &role, &status)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro no banco")
		return
	}
	if !auth.CheckPassword(req.Password, hash) {
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}
	if status == "pending" {
		writeError(w, http.StatusForbidden, "conta aguardando aprovação de um administrador")
		return
	}
	if status == "inactive" {
		writeError(w, http.StatusForbidden, "conta inativa")
		return
	}
	token, err := auth.GenerateToken(id, email, h.JWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar token")
		return
	}
	perms, _ := permissions.UserPermissions(h.DB, id)
	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  map[string]interface{}{"id": id, "name": name, "email": email, "created_at": createdAt, "role": role, "status": status, "permissions": perms},
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if uid == 0 {
		writeError(w, http.StatusUnauthorized, "sem autenticação")
		return
	}
	var id int64
	var name, email, createdAt, role, status string
	err := h.DB.QueryRow("SELECT id, name, email, created_at, role, status FROM users WHERE id=?", uid).Scan(&id, &name, &email, &createdAt, &role, &status)
	if err != nil {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}
	perms, _ := permissions.UserPermissions(h.DB, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"id": id, "name": name, "email": email, "created_at": createdAt, "role": role, "status": status, "permissions": perms})
}

type updateProfileRequest struct {
	Name string `json:"name"`
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if uid == 0 {
		writeError(w, http.StatusUnauthorized, "sem autenticação")
		return
	}
	var req updateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "nome é obrigatório")
		return
	}
	_, err := h.DB.Exec("UPDATE users SET name = ? WHERE id = ?", req.Name, uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao atualizar perfil")
		return
	}
	var email, createdAt, role, status string
	_ = h.DB.QueryRow("SELECT email, created_at, role, status FROM users WHERE id = ?", uid).Scan(&email, &createdAt, &role, &status)
	perms, _ := permissions.UserPermissions(h.DB, uid)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": uid, "name": req.Name, "email": email, "created_at": createdAt, "role": role, "status": status, "permissions": perms,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r)
	if uid == 0 {
		writeError(w, http.StatusUnauthorized, "sem autenticação")
		return
	}
	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(req.NewPassword) < 6 {
		writeError(w, http.StatusBadRequest, "nova senha deve ter pelo menos 6 caracteres")
		return
	}
	var hash string
	if err := h.DB.QueryRow("SELECT password_hash FROM users WHERE id = ?", uid).Scan(&hash); err != nil {
		writeError(w, http.StatusNotFound, "usuário não encontrado")
		return
	}
	if !auth.CheckPassword(req.CurrentPassword, hash) {
		writeError(w, http.StatusUnauthorized, "senha atual incorreta")
		return
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}
	if _, err := h.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao alterar senha")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "senha alterada com sucesso"})
}
