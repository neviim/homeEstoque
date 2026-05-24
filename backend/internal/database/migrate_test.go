package database_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestMigrate_CreatesAllExpectedTables(t *testing.T) {
	db := testutil.NewTestDB(t)

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		tables = append(tables, n)
	}

	expected := []string{
		"categories", "item_photos", "items", "locations",
		"movements", "role_permissions", "roles", "users",
	}
	sort.Strings(expected)
	for _, want := range expected {
		assert.Contains(t, tables, want, "tabela %s não foi criada", want)
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	// Abrir o mesmo DB 2× consecutivas. A 2ª chamada vai rodar migrate de
	// novo (com schemas já criados) — não pode dar erro.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db1, err := database.Open(path)
	require.NoError(t, err)
	require.NoError(t, db1.Close())

	db2, err := database.Open(path)
	require.NoError(t, err, "migrate idempotente deveria suportar re-execução")
	require.NoError(t, db2.Close())
}

func TestOpen_CreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "novo", "subdir", "test.db")

	db, err := database.Open(nested)
	require.NoError(t, err)
	defer db.Close()

	// Diretório deve ter sido criado automaticamente
	assert.FileExists(t, nested)
}
