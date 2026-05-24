package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

// =================== Register ===================

func TestRegister_FirstUserBecomesAdminWithToken(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/auth/register", "", map[string]string{
		"name":     "Primeiro",
		"email":    "primeiro@x.com",
		"password": "senha123",
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out struct {
		Token string                 `json:"token"`
		User  map[string]interface{} `json:"user"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.NotEmpty(t, out.Token)
	assert.Equal(t, "admin", out.User["role"])
	assert.Equal(t, "active", out.User["status"])
	// Permissions devem vir como array (mesmo que vazio nunca seria — admin tem todas)
	perms, ok := out.User["permissions"].([]interface{})
	require.True(t, ok, "permissions deve ser array")
	assert.NotEmpty(t, perms, "admin deve ter permissões")
}

func TestRegister_SecondUserBecomesPendingWithoutToken(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	// Já existe um admin
	testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/auth/register", "", map[string]string{
		"name":     "Maria",
		"email":    "maria@x.com",
		"password": "senha123",
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "pending", out["status"])
	assert.NotContains(t, out, "token", "pending não deve receber token")
}

func TestRegister_DuplicateEmail_Returns409(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	testutil.CreateUser(t, db, "Existente", "dup@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/auth/register", "", map[string]string{
		"name":     "Outro",
		"email":    "dup@x.com",
		"password": "senha456",
	})
	assert.Equal(t, http.StatusConflict, status)
}

func TestRegister_ShortPassword_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/auth/register", "", map[string]string{
		"name": "X", "email": "x@x.com", "password": "123",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestRegister_EmptyName_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/auth/register", "", map[string]string{
		"name": "", "email": "x@x.com", "password": "senha123",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestRegister_EmailNormalized(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	_, _ = testutil.Request(t, srv, "POST", "/api/auth/register", "", map[string]string{
		"name": "X", "email": "  ALICE@X.COM  ", "password": "senha123",
	})

	var stored string
	require.NoError(t, db.QueryRow(`SELECT email FROM users WHERE name = 'X'`).Scan(&stored))
	assert.Equal(t, "alice@x.com", stored)
}

func TestRegister_MalformedBody_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/auth/register", "", "{not-json}")
	assert.Equal(t, http.StatusBadRequest, status)
}

// =================== Login ===================

func TestLogin_CorrectCredentials_Returns200WithToken(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	testutil.CreateUser(t, db, "Alice", "alice@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/auth/login", "", map[string]string{
		"email": "alice@x.com", "password": "senha123",
	})
	require.Equal(t, http.StatusOK, status, "body=%s", string(body))

	var out struct {
		Token string `json:"token"`
		User  struct {
			Role        string   `json:"role"`
			Status      string   `json:"status"`
			Permissions []string `json:"permissions"`
		} `json:"user"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.NotEmpty(t, out.Token)
	assert.Equal(t, "admin", out.User.Role)
	assert.Equal(t, "active", out.User.Status)
	assert.Contains(t, out.User.Permissions, "users.manage")
}

func TestLogin_WrongPassword_Returns401(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	testutil.CreateUser(t, db, "X", "x@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/auth/login", "", map[string]string{
		"email": "x@x.com", "password": "errada",
	})
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestLogin_NonexistentEmail_Returns401(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/auth/login", "", map[string]string{
		"email": "ninguem@x.com", "password": "senha123",
	})
	assert.Equal(t, http.StatusUnauthorized, status)
	// Mensagem genérica — não vaza existência do email
	assert.Contains(t, string(body), "credenciais")
}

func TestLogin_PendingStatus_Returns403(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	testutil.CreateUserWithStatus(t, db, "P", "p@x.com", "senha123", "user", "pending")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/auth/login", "", map[string]string{
		"email": "p@x.com", "password": "senha123",
	})
	assert.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "aprova")
}

func TestLogin_InactiveStatus_Returns403(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	testutil.CreateUserWithStatus(t, db, "I", "i@x.com", "senha123", "user", "inactive")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/auth/login", "", map[string]string{
		"email": "i@x.com", "password": "senha123",
	})
	assert.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, string(body), "inativa")
}

func TestLogin_MalformedBody_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/auth/login", "", "{not-json")
	assert.Equal(t, http.StatusBadRequest, status)
}

// =================== Me ===================

func TestMe_NoToken_Returns401(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "GET", "/api/auth/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestMe_ReturnsFreshRoleAndPermissions(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	u := testutil.CreateUser(t, db, "X", "x@x.com", "senha123", "viewer")

	// Admin de outra aba muda o role do user no banco — Me deve refletir já
	testutil.MustExec(t, db, `UPDATE users SET role = 'admin' WHERE id = ?`, u.ID)
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "GET", "/api/auth/me", u.Token, nil)
	require.Equal(t, http.StatusOK, status, "body=%s", string(body))
	var out struct {
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "admin", out.Role)
	assert.Contains(t, out.Permissions, "users.manage")
}

// =================== UpdateProfile ===================

func TestUpdateProfile_UpdatesNameOnly(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	u := testutil.CreateUser(t, db, "Original", "orig@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "PUT", "/api/auth/profile", u.Token, map[string]string{
		"name": "Novo Nome",
	})
	require.Equal(t, http.StatusOK, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "Novo Nome", out["name"])
	assert.Equal(t, "orig@x.com", out["email"], "email não deve mudar")
}

func TestUpdateProfile_EmptyName_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	u := testutil.CreateUser(t, db, "X", "x@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", "/api/auth/profile", u.Token, map[string]string{
		"name": "  ",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

// =================== ChangePassword ===================

func TestChangePassword_CorrectCurrent_Returns200(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	u := testutil.CreateUser(t, db, "X", "x@x.com", "senha-atual", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", "/api/auth/password", u.Token, map[string]string{
		"current_password": "senha-atual",
		"new_password":     "senha-nova-456",
	})
	require.Equal(t, http.StatusOK, status)

	// Verifica que a nova senha funciona e a antiga não
	var hash string
	require.NoError(t, db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, u.ID).Scan(&hash))
	assert.True(t, auth.CheckPassword("senha-nova-456", hash))
	assert.False(t, auth.CheckPassword("senha-atual", hash))
}

func TestChangePassword_WrongCurrent_Returns401(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	u := testutil.CreateUser(t, db, "X", "x@x.com", "senha-atual", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", "/api/auth/password", u.Token, map[string]string{
		"current_password": "ERRADA",
		"new_password":     "qualquer-coisa",
	})
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestChangePassword_ShortNew_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	u := testutil.CreateUser(t, db, "X", "x@x.com", "senha-atual", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", "/api/auth/password", u.Token, map[string]string{
		"current_password": "senha-atual",
		"new_password":     "12345",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}
