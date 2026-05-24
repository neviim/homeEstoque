package backup_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"

	"github.com/neviim/homeestoque/backend/internal/testutil"
)

// TestRestore_RestoresOriginalState valida o fluxo end-to-end:
//  1. Cria item, depois backup
//  2. Apaga item
//  3. Restore — Manager extrai arquivos e chama restart stub
//  4. Reabre o DB do disco e verifica que o item ressuscitou
func TestRestore_RestoresOriginalState(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	// Categoria + item original
	catID := testutil.CreateCategory(t, env.DB, "Eletrônicos teste")
	itemID := testutil.CreateItem(t, env.DB, "Item original", testutil.ItemOpts{
		CategoryID: &catID,
		Quantity:   3,
	})

	// Backup capturando o estado atual
	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	// Apaga item: depois do restore deveria reaparecer
	_, err = env.DB.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, itemID)
	require.NoError(t, err)
	var countAfterDelete int
	require.NoError(t, env.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM items WHERE id = ?`, itemID,
	).Scan(&countAfterDelete))
	assert.Equal(t, 0, countAfterDelete)

	// Prepare + Restore
	tok, _, err := env.Manager.PrepareRestore(ctx, b.ID)
	require.NoError(t, err)
	require.NoError(t, env.Manager.Restore(ctx, b.ID, tok))

	// Restart stub foi chamado
	assert.True(t, *env.RestartCalled, "restart deveria ter sido chamado")

	// O DB original foi fechado pelo Restore. Reabrimos pra inspecionar.
	db, err := sql.Open("sqlite", env.DBPath+"?_pragma=journal_mode(DELETE)")
	require.NoError(t, err)
	defer db.Close()

	var found int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM items WHERE id = ?`, itemID,
	).Scan(&found))
	assert.Equal(t, 1, found, "item deveria ter sido restaurado")
}

func TestRestore_RestoresUploads(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	// Cria um arquivo em uploads/, faz backup, apaga, restore.
	origPath := filepath.Join(env.UploadDir, "foto.jpg")
	require.NoError(t, os.WriteFile(origPath, []byte("BYTES-ORIGINAIS"), 0o644))

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	require.NoError(t, os.Remove(origPath))
	_, statErr := os.Stat(origPath)
	require.True(t, os.IsNotExist(statErr))

	tok, _, err := env.Manager.PrepareRestore(ctx, b.ID)
	require.NoError(t, err)
	require.NoError(t, env.Manager.Restore(ctx, b.ID, tok))

	// Arquivo voltou
	data, err := os.ReadFile(origPath)
	require.NoError(t, err)
	assert.Equal(t, "BYTES-ORIGINAIS", string(data))
}

func TestRestore_TokenIsSingleUse(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	tok, _, err := env.Manager.PrepareRestore(ctx, b.ID)
	require.NoError(t, err)
	require.NoError(t, env.Manager.Restore(ctx, b.ID, tok))

	// Segunda tentativa com o mesmo token falha
	err = env.Manager.Restore(ctx, b.ID, tok)
	require.Error(t, err)
}
