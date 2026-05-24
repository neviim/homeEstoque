package handlers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func uidPath(id int64, suffix ...string) string {
	p := "/api/users/" + strconv.FormatInt(id, 10)
	for _, s := range suffix {
		p += s
	}
	return p
}

func TestUsers_List_ExcludesMCPUser(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "GET", "/api/users", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct {
		Users []struct {
			Email string `json:"email"`
		} `json:"users"`
	}
	testutil.DecodeJSON(t, body, &out)
	for _, u := range out.Users {
		assert.NotEqual(t, "mcp@homeestoque.local", u.Email, "MCP user nunca deve aparecer no /users")
	}
}

func TestUsers_Create_WithValidRole_Returns201Active(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/users", admin.Token, map[string]string{
		"name": "Bob", "email": "bob@x.com", "password": "senha123", "role": "user",
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "user", out["role"])
	assert.Equal(t, "active", out["status"], "admin cria user já como active (sem pending)")
}

func TestUsers_Create_NonexistentRole_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/users", admin.Token, map[string]string{
		"name": "X", "email": "x@x.com", "password": "senha123", "role": "hacker",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestUsers_Create_DuplicateEmail_Returns409(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	testutil.CreateUser(t, db, "Existente", "dup@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/users", admin.Token, map[string]string{
		"name": "Outro", "email": "dup@x.com", "password": "senha123", "role": "user",
	})
	assert.Equal(t, http.StatusConflict, status)
}

func TestUsers_Create_ShortPassword_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/users", admin.Token, map[string]string{
		"name": "X", "email": "x@x.com", "password": "12", "role": "user",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestUsers_Update_ChangesNameAndRole_NotEmail(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	target := testutil.CreateUser(t, db, "Bob", "bob@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	// Payload válido — só campos aceitos
	status, body := testutil.Request(t, srv, "PUT", uidPath(target.ID), admin.Token, map[string]string{
		"name": "Robert",
		"role": "viewer",
	})
	require.Equal(t, http.StatusOK, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "Robert", out["name"])
	assert.Equal(t, "bob@x.com", out["email"], "email permanece intacto no banco")
	assert.Equal(t, "viewer", out["role"])
}

func TestUsers_Update_RejectsUnknownFields(t *testing.T) {
	// Defesa em profundidade: handler usa DisallowUnknownFields, então tentar
	// enviar email (que é imutável) retorna 400 em vez de mudar silenciosamente.
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	target := testutil.CreateUser(t, db, "Bob", "bob@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", uidPath(target.ID), admin.Token, map[string]string{
		"name":  "Robert",
		"email": "tentativa-de-mudar@x.com",
		"role":  "viewer",
	})
	assert.Equal(t, http.StatusBadRequest, status,
		"campo desconhecido deve ser rejeitado em vez de ignorado")
}

func TestUsers_UpdateStatus_ApprovePending(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	pending := testutil.CreateUserWithStatus(t, db, "Pen", "p@x.com", "senha123", "user", "pending")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", uidPath(pending.ID, "/status"), admin.Token, map[string]string{
		"status": "active",
	})
	require.Equal(t, http.StatusOK, status)

	var st string
	require.NoError(t, db.QueryRow(`SELECT status FROM users WHERE id = ?`, pending.ID).Scan(&st))
	assert.Equal(t, "active", st)
}

func TestUsers_UpdateStatus_CannotChangeOwn_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", uidPath(admin.ID, "/status"), admin.Token, map[string]string{
		"status": "inactive",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestUsers_Update_LastAdmin_CannotDemoteSelf_Returns409(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "PUT", uidPath(admin.ID), admin.Token, map[string]string{
		"name": "Admin",
		"role": "user", // auto-rebaixamento sendo o último admin
	})
	assert.Equal(t, http.StatusConflict, status, "body=%s", string(body))
}

func TestUsers_UpdateStatus_LastAdmin_CannotInactivate_Returns409(t *testing.T) {
	// Cenário: dois admins (callerAdmin + targetAdmin). callerAdmin inativa
	// targetAdmin → fica com 1 admin ativo. Inativar callerAdmin é auto-status (400).
	// Para forçar o caminho 409 do handler, manipulamos o DB diretamente:
	// deixamos targetAdmin como admin ativo, callerAdmin como o ÚLTIMO admin ativo,
	// e callerAdmin tenta inativar targetAdmin (que é admin, mas inactive). Ainda é
	// permitido — não disparou. O caminho 409 só dispara se houver tentativa de
	// inativar/excluir um admin que reduza o COUNT(active admin)=1 a 0.
	//
	// Solução: cria 2 admins ativos. Caller inativa target (OK, ainda há 1 ativo: caller).
	// Reativa target. Cria fixture: callerAdmin é o caller, há também
	// uniqueAdmin (active, admin). Caller tenta inativar uniqueAdmin enquanto caller
	// está com status='inactive' via SQL → cenário "uniqueAdmin é o último ativo".
	db := testutil.NewSeededTestDB(t)
	caller := testutil.CreateUser(t, db, "Caller", "caller@x.com", "senha123", "admin")
	target := testutil.CreateUser(t, db, "Target", "target@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Inativa o caller via SQL (sem passar pelo handler) — agora target é o único admin ativo
	testutil.MustExec(t, db, `UPDATE users SET status = 'inactive' WHERE id = ?`, caller.ID)
	// Reativa via SQL pra caller ainda conseguir fazer requests (middleware não checa status,
	// só JWT). Agora há 2 admins (caller+target) com status: caller=active, target=active.
	testutil.MustExec(t, db, `UPDATE users SET status = 'active' WHERE id = ?`, caller.ID)
	// Inativa target via SQL — agora apenas caller é admin ativo. Reativa só pra ele aparecer
	// como admin no banco mas continuar válido pra deletar.
	// Fato: o handler só conta WHERE role='admin' AND status='active'. Se inativarmos target via
	// SQL, ainda há caller ativo. Caller tenta inativar target → handler conta admins ativos: 1
	// (apenas caller) → bloqueia 409 porque target é admin e adminCount ≤ 1.
	testutil.MustExec(t, db, `UPDATE users SET status = 'inactive' WHERE id = ?`, target.ID)

	status, body := testutil.Request(t, srv, "PUT", uidPath(target.ID, "/status"), caller.Token, map[string]string{
		"status": "inactive",
	})
	assert.Equal(t, http.StatusConflict, status, "body=%s", string(body))
}

func TestUsers_Delete_LastAdmin_Returns409(t *testing.T) {
	// Mesmo padrão: cria 2 admins, inativa um, tenta deletar o outro admin.
	db := testutil.NewSeededTestDB(t)
	caller := testutil.CreateUser(t, db, "Caller", "caller@x.com", "senha123", "admin")
	target := testutil.CreateUser(t, db, "Target", "target@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Inativa target — agora caller é o único admin ativo.
	testutil.MustExec(t, db, `UPDATE users SET status = 'inactive' WHERE id = ?`, target.ID)

	// Caller deleta target. target ainda é role='admin' (apenas inactive). adminCount=1.
	// Handler deve recusar com 409.
	status, body := testutil.Request(t, srv, "DELETE", uidPath(target.ID), caller.Token, nil)
	assert.Equal(t, http.StatusConflict, status, "body=%s", string(body))
}

func TestUsers_Delete_Self_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "DELETE", uidPath(admin.ID), admin.Token, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestUsers_ResetPassword_NoCurrentRequired(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	target := testutil.CreateUser(t, db, "X", "x@x.com", "senha-antiga", "user")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", uidPath(target.ID, "/password"), admin.Token, map[string]string{
		"password": "senha-nova-via-admin",
	})
	require.Equal(t, http.StatusOK, status)

	var hash string
	require.NoError(t, db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, target.ID).Scan(&hash))
	assert.True(t, auth.CheckPassword("senha-nova-via-admin", hash))
}

func TestUsers_ResetPassword_ShortPassword_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	target := testutil.CreateUser(t, db, "X", "x@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "PUT", uidPath(target.ID, "/password"), admin.Token, map[string]string{
		"password": "12",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}
