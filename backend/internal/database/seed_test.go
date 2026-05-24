package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/permissions"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestSeed_CreatesThreeSeedRoles(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	rows, err := db.Query(`SELECT name FROM roles ORDER BY name`)
	require.NoError(t, err)
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		roles = append(roles, n)
	}
	assert.Equal(t, []string{"admin", "user", "viewer"}, roles)
}

func TestSeed_IsIdempotent_DoesNotDuplicate(t *testing.T) {
	db := testutil.NewTestDB(t)
	require.NoError(t, database.Seed(db))
	require.NoError(t, database.Seed(db)) // 2ª chamada

	var rolesCount, permsCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM roles`).Scan(&rolesCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'admin')`).Scan(&permsCount))

	assert.Equal(t, 3, rolesCount, "Seed 2× não deve duplicar roles")
	assert.Equal(t, len(permissions.Catalog), permsCount, "admin deve manter exatamente as perms do catálogo após 2 seeds")
}

func TestSeed_AdminAlwaysHasAllPermissions(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	perms, err := permissions.RolePermissions(db, "admin")
	require.NoError(t, err)
	assert.Len(t, perms, len(permissions.Catalog),
		"admin deve ter exatamente todas as %d permissões do catálogo", len(permissions.Catalog))

	// Cada key do catálogo está presente
	for _, k := range permissions.Keys() {
		assert.Contains(t, perms, k, "admin sem permissão %s", k)
	}
}

func TestSeed_AdminGetsNewPermissionAddedAfterFirstSeed(t *testing.T) {
	// Cenário: DB já existia antes de novas permissões serem adicionadas no catálogo.
	// O seed deve completar as keys faltando — não exigir reset do DB.
	db := testutil.NewTestDB(t)
	require.NoError(t, database.Seed(db))

	// Remove manualmente uma permissão do admin (simula DB pré-existente sem essa perm)
	testutil.MustExec(t, db,
		`DELETE FROM role_permissions WHERE permission = ? AND role_id = (SELECT id FROM roles WHERE name = 'admin')`,
		"roles.manage",
	)

	// Roda Seed de novo — deve reinserir
	require.NoError(t, database.Seed(db))
	perms, err := permissions.RolePermissions(db, "admin")
	require.NoError(t, err)
	assert.Contains(t, perms, "roles.manage", "Seed deve reinserir permissão faltante no admin")
}

func TestSeed_UserAndViewerHaveExpectedDefaults(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	userPerms, err := permissions.RolePermissions(db, "user")
	require.NoError(t, err)
	assert.NotContains(t, userPerms, "users.manage")
	assert.NotContains(t, userPerms, "roles.manage")
	assert.NotContains(t, userPerms, "dashboard.view_value")
	assert.Contains(t, userPerms, "items.create")

	viewerPerms, err := permissions.RolePermissions(db, "viewer")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dashboard.view", "items.view"}, viewerPerms)
}

func TestSeed_SeedRoleIfEmpty_DoesNotOverwriteCustomPerms(t *testing.T) {
	// Se admin reconfigurou as perms do role "user", um re-seed não pode
	// sobrescrever — só popula se estava vazio.
	db := testutil.NewTestDB(t)
	require.NoError(t, database.Seed(db))

	// Admin remove tudo de "user" exceto items.view
	testutil.MustExec(t, db,
		`DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'user') AND permission != 'items.view'`,
	)

	require.NoError(t, database.Seed(db))
	perms, err := permissions.RolePermissions(db, "user")
	require.NoError(t, err)
	assert.Equal(t, []string{"items.view"}, perms,
		"Seed não deve repopular role que já foi customizado")
}

func TestSeed_CreatesMCPUser(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	var name, role, status string
	require.NoError(t, db.QueryRow(
		`SELECT name, role, status FROM users WHERE email = ?`,
		database.MCPUserEmail,
	).Scan(&name, &role, &status))

	assert.Equal(t, "MCP Assistant", name)
	assert.Equal(t, "user", role)
	assert.Equal(t, "inactive", status, "MCP user nunca pode fazer login")
}

func TestSeed_PromotesFirstHumanToAdmin(t *testing.T) {
	db := testutil.NewTestDB(t)
	require.NoError(t, database.Seed(db))

	// Insere um humano com role=user — Seed novamente deve promovê-lo
	testutil.MustExec(t, db,
		`INSERT INTO users (name, email, password_hash, role, status) VALUES ('Humano', 'humano@x.com', 'h', 'user', 'pending')`,
	)
	require.NoError(t, database.Seed(db))

	var role, status string
	require.NoError(t, db.QueryRow(
		`SELECT role, status FROM users WHERE email = 'humano@x.com'`,
	).Scan(&role, &status))
	assert.Equal(t, "admin", role)
	assert.Equal(t, "active", status)
}

func TestSeed_DefaultCategoriesOnFirstRun(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	var c1 int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&c1))
	assert.Greater(t, c1, 0, "primeira execução deveria seedar categorias")
}

func TestSeed_DoesNotDuplicateCategoriesOnReRun(t *testing.T) {
	db := testutil.NewSeededTestDB(t)

	var c1 int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&c1))

	require.NoError(t, database.Seed(db)) // 2ª chamada

	var c2 int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&c2))
	assert.Equal(t, c1, c2, "Seed não deve duplicar categorias")
}
