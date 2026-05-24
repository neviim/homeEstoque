package permissions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/permissions"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestRolePermissions_AdminHasAllAfterSeed(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	perms, err := permissions.RolePermissions(db, "admin")
	require.NoError(t, err)
	assert.Len(t, perms, len(permissions.Catalog), "admin deveria ter todas as %d permissões do catálogo", len(permissions.Catalog))
}

func TestRolePermissions_ViewerHasExactlyTwo(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	perms, err := permissions.RolePermissions(db, "viewer")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dashboard.view", "items.view"}, perms)
}

func TestRolePermissions_NonexistentRoleReturnsEmptySlice(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	perms, err := permissions.RolePermissions(db, "nao-existe")
	require.NoError(t, err)
	assert.NotNil(t, perms, "deve retornar slice vazio, não nil")
	assert.Empty(t, perms)
}

func TestUserPermissions_ReturnsRolePermissions(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	user := testutil.CreateUser(t, db, "Alice", "alice@x.com", "senha123", "viewer")

	perms, err := permissions.UserPermissions(db, user.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dashboard.view", "items.view"}, perms)
}

func TestUserPermissions_UserWithInvalidRoleReturnsEmpty(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	// Insere user com role que não existe na tabela roles — simulando role
	// que foi removido depois (o sistema previne, mas e se acontecer?)
	testutil.MustExec(t, db,
		`INSERT INTO users (name, email, password_hash, role, status) VALUES (?, ?, ?, ?, 'active')`,
		"Orfão", "orfao@x.com", "hash", "ghost-role",
	)
	var uid int64
	testutil.MustQueryRow(t, db, `SELECT id FROM users WHERE email = ?`, []interface{}{"orfao@x.com"}, &uid)

	perms, err := permissions.UserPermissions(db, uid)
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestHasPermission_TrueWhenRoleHas(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")

	ok, err := permissions.HasPermission(db, admin.ID, "users.manage")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestHasPermission_FalseWhenRoleDoesNot(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	viewer := testutil.CreateUser(t, db, "Viewer", "viewer@x.com", "senha123", "viewer")

	ok, err := permissions.HasPermission(db, viewer.ID, "users.manage")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHasPermission_FalseForNonexistentUser(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	ok, err := permissions.HasPermission(db, 99999, "items.view")
	require.NoError(t, err)
	assert.False(t, ok)
}
