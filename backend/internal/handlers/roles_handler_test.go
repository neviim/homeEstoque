package handlers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/permissions"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func ridPath(id int64, suffix ...string) string {
	p := "/api/roles/" + strconv.FormatInt(id, 10)
	for _, s := range suffix {
		p += s
	}
	return p
}

// ----- TESTES -----

func TestRoles_ListCatalog_Returns15Permissions(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "GET", "/api/permissions", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct {
		Permissions []permissions.Permission `json:"permissions"`
	}
	testutil.DecodeJSON(t, body, &out)
	assert.Len(t, out.Permissions, len(permissions.Catalog))
}

func TestRoles_List_ReturnsThreeSeedRolesWithCounts(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	testutil.CreateUser(t, db, "U1", "u1@x.com", "senha123", "user")
	testutil.CreateUser(t, db, "U2", "u2@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "GET", "/api/roles", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)

	var out struct {
		Roles []struct {
			Name        string   `json:"name"`
			IsSystem    bool     `json:"is_system"`
			UserCount   int      `json:"user_count"`
			Permissions []string `json:"permissions"`
		} `json:"roles"`
	}
	testutil.DecodeJSON(t, body, &out)
	require.GreaterOrEqual(t, len(out.Roles), 3)

	for _, r := range out.Roles {
		switch r.Name {
		case "admin":
			assert.True(t, r.IsSystem)
			assert.Len(t, r.Permissions, len(permissions.Catalog))
			assert.Equal(t, 1, r.UserCount) // o admin que criamos
		case "user":
			assert.False(t, r.IsSystem)
			assert.Equal(t, 2, r.UserCount)
		case "viewer":
			assert.False(t, r.IsSystem)
		}
	}
}

func TestRoles_Create_ValidSlug_Returns201(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, body := testutil.Request(t, srv, "POST", "/api/roles", admin.Token, map[string]string{
		"name": "auditor", "label": "Auditor", "description": "Lê tudo",
	})
	require.Equal(t, http.StatusCreated, status, "body=%s", string(body))

	var out map[string]interface{}
	testutil.DecodeJSON(t, body, &out)
	assert.Equal(t, "auditor", out["name"])
	assert.Equal(t, false, out["is_system"])
	assert.Empty(t, out["permissions"], "novo role começa sem permissões")
}

func TestRoles_Create_InvalidSlug_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Nota: maiúsculas (Auditor) são aceitas porque o handler faz ToLower
	// antes da validação — comportamento intencional, tolerância de UI.
	cases := []string{
		"123role",       // começa com número
		"role-name",     // tem hífen
		"role name",     // tem espaço
		"a",             // muito curto (regex exige mín. 2 chars)
		"",              // vazio
	}
	for _, slug := range cases {
		t.Run(slug, func(t *testing.T) {
			status, _ := testutil.Request(t, srv, "POST", "/api/roles", admin.Token, map[string]string{
				"name": slug, "label": "X",
			})
			assert.Equal(t, http.StatusBadRequest, status, "slug %q deveria ser rejeitado", slug)
		})
	}
}

func TestRoles_Create_DuplicateName_Returns409(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	status, _ := testutil.Request(t, srv, "POST", "/api/roles", admin.Token, map[string]string{
		"name": "admin", "label": "Tentativa duplicada",
	})
	assert.Equal(t, http.StatusConflict, status)
}

func TestRoles_Update_RenamePropagatesToUsers(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	// 2 usuários com o role "user"
	testutil.CreateUser(t, db, "U1", "u1@x.com", "senha123", "user")
	testutil.CreateUser(t, db, "U2", "u2@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	// Descobre o id do role "user"
	var userRoleID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'user'`).Scan(&userRoleID))

	// Renomeia user → membro
	status, _ := testutil.Request(t, srv, "PUT", ridPath(userRoleID), admin.Token, map[string]string{
		"name": "membro", "label": "Membro", "description": "Renomeado",
	})
	require.Equal(t, http.StatusOK, status)

	// Verifica que os 2 humanos tiveram seu role atualizado para "membro".
	// (O usuário sintético MCP — mcp@homeestoque.local — também é afetado pelo
	// UPDATE; é comportamento intencional já que ele só serve de identifier
	// para movements e tem status=inactive, sem login.)
	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = 'membro' AND email != 'mcp@homeestoque.local'`,
	).Scan(&count))
	assert.Equal(t, 2, count)

	// E que ninguém (humano) ficou com role "user"
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = 'user' AND email != 'mcp@homeestoque.local'`,
	).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestRoles_Update_AdminNameIsImmutable(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	var adminID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'admin'`).Scan(&adminID))

	// Tenta renomear admin → super_admin: label e description mudam, name fica
	status, body := testutil.Request(t, srv, "PUT", ridPath(adminID), admin.Token, map[string]string{
		"name": "super_admin", "label": "Super", "description": "Tentativa",
	})
	require.Equal(t, http.StatusOK, status, "body=%s", string(body))

	var name string
	require.NoError(t, db.QueryRow(`SELECT name FROM roles WHERE id = ?`, adminID).Scan(&name))
	assert.Equal(t, "admin", name, "admin.name deve ser imutável (is_system=1)")
}

func TestRoles_Delete_AdminRole_Returns403(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	var adminID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'admin'`).Scan(&adminID))

	status, _ := testutil.Request(t, srv, "DELETE", ridPath(adminID), admin.Token, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestRoles_Delete_RoleWithUsers_Returns409(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	testutil.CreateUser(t, db, "U", "u@x.com", "senha123", "user")
	srv := testutil.NewTestServer(t, db)

	var userRoleID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'user'`).Scan(&userRoleID))

	status, _ := testutil.Request(t, srv, "DELETE", ridPath(userRoleID), admin.Token, nil)
	assert.Equal(t, http.StatusConflict, status)
}

func TestRoles_Delete_EmptyCustomRole_Returns200(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	// Cria role custom
	testutil.Request(t, srv, "POST", "/api/roles", admin.Token, map[string]string{
		"name": "temp_role", "label": "Temp",
	})
	var rid int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'temp_role'`).Scan(&rid))

	status, _ := testutil.Request(t, srv, "DELETE", ridPath(rid), admin.Token, nil)
	assert.Equal(t, http.StatusOK, status)

	// Verifica que sumiu
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM roles WHERE id = ?`, rid).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestRoles_UpdatePermissions_ReplacesSet(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	var viewerID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'viewer'`).Scan(&viewerID))

	// Viewer começa com 2 perms. Substitui por 3 outras.
	status, _ := testutil.Request(t, srv, "PUT", ridPath(viewerID, "/permissions"), admin.Token, map[string]interface{}{
		"permissions": []string{"items.view", "items.create", "categories.view"},
	})
	require.Equal(t, http.StatusOK, status)

	perms, err := permissions.RolePermissions(db, "viewer")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"items.view", "items.create", "categories.view"}, perms,
		"perms anteriores devem ter sido removidas")
}

func TestRoles_UpdatePermissions_InvalidKey_Returns400(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	var viewerID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'viewer'`).Scan(&viewerID))

	status, body := testutil.Request(t, srv, "PUT", ridPath(viewerID, "/permissions"), admin.Token, map[string]interface{}{
		"permissions": []string{"items.view", "permissao.que.nao.existe"},
	})
	assert.Equal(t, http.StatusBadRequest, status, "body=%s", string(body))
}

func TestRoles_UpdatePermissions_AdminForceAll(t *testing.T) {
	// Tentar definir admin com permissões vazias — handler deve forçar todas.
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	srv := testutil.NewTestServer(t, db)

	var adminID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'admin'`).Scan(&adminID))

	status, _ := testutil.Request(t, srv, "PUT", ridPath(adminID, "/permissions"), admin.Token, map[string]interface{}{
		"permissions": []string{}, // tentativa de zerar
	})
	require.Equal(t, http.StatusOK, status)

	perms, err := permissions.RolePermissions(db, "admin")
	require.NoError(t, err)
	assert.Len(t, perms, len(permissions.Catalog), "admin sempre tem todas as permissões")
}

func TestRoles_UpdatePermissions_ChangeReflectsImmediately(t *testing.T) {
	// Sem precisar relogar: chamar /auth/me após mudança vê as novas permissions.
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")
	viewer := testutil.CreateUser(t, db, "V", "v@x.com", "senha123", "viewer")
	srv := testutil.NewTestServer(t, db)

	// Antes da mudança: viewer tem só 2 perms
	_, body := testutil.Request(t, srv, "GET", "/api/auth/me", viewer.Token, nil)
	var meBefore struct {
		Permissions []string `json:"permissions"`
	}
	testutil.DecodeJSON(t, body, &meBefore)
	require.Len(t, meBefore.Permissions, 2)

	// Admin adiciona items.create no viewer
	var viewerID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM roles WHERE name = 'viewer'`).Scan(&viewerID))
	status, _ := testutil.Request(t, srv, "PUT", ridPath(viewerID, "/permissions"), admin.Token, map[string]interface{}{
		"permissions": []string{"dashboard.view", "items.view", "items.create"},
	})
	require.Equal(t, http.StatusOK, status)

	// viewer chama /me com o mesmo token antigo — deve ver as novas perms
	_, body = testutil.Request(t, srv, "GET", "/api/auth/me", viewer.Token, nil)
	var meAfter struct {
		Permissions []string `json:"permissions"`
	}
	testutil.DecodeJSON(t, body, &meAfter)
	assert.Contains(t, meAfter.Permissions, "items.create",
		"mudança de permission do role vale imediatamente (sem relogar)")
}
