package backup_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestCreate_ProducesValidArchiveWithDBAndUploads(t *testing.T) {
	env := testutil.NewBackupEnv(t)

	// Adiciona um arquivo em UPLOAD_DIR pra garantir que entra no tar.
	uploadFile := filepath.Join(env.UploadDir, "test-photo.jpg")
	require.NoError(t, os.WriteFile(uploadFile, []byte("FAKE-IMG-BYTES"), 0o644))

	b, err := env.Manager.Create(context.Background(), "manual")
	require.NoError(t, err)
	require.NotNil(t, b)

	assert.Equal(t, "manual", b.Type)
	assert.Equal(t, "ok", b.Status)
	assert.NotEmpty(t, b.SHA256)
	assert.Greater(t, b.SizeBytes, int64(0))

	// Arquivo existe em disco
	path := filepath.Join(env.BackupDir, b.Filename)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, b.SizeBytes, info.Size())

	// sha256 confere com o arquivo em disco
	got := hashFile(t, path)
	assert.Equal(t, b.SHA256, got, "sha256 armazenado deve bater com o arquivo")

	// Conteúdo do tar contém db + uploads
	entries := listTarEntries(t, path)
	assert.Contains(t, entries, "db/homeestoque.db")
	assert.Contains(t, entries, "uploads/test-photo.jpg")
}

func TestList_ReturnsBackupsMostRecentFirst(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b1, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)
	b2, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	list, err := env.Manager.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// Ordem desc por id (mais recente primeiro)
	assert.Equal(t, b2.ID, list[0].ID)
	assert.Equal(t, b1.ID, list[1].ID)
}

func TestVerify_DetectsCorruption(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	// Corromper o arquivo: trunca para alguns bytes
	path := filepath.Join(env.BackupDir, b.Filename)
	require.NoError(t, os.Truncate(path, 10))

	verified, err := env.Manager.Verify(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, "corrupted", verified.Status)
	assert.NotEmpty(t, verified.Notes)
}

func TestVerify_OkWhenIntact(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	verified, err := env.Manager.Verify(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, "ok", verified.Status)
	assert.NotNil(t, verified.VerifiedAt)
}

func TestDelete_RemovesFileAndRow(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)
	path := filepath.Join(env.BackupDir, b.Filename)
	require.FileExists(t, path)

	require.NoError(t, env.Manager.Delete(ctx, b.ID))
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "arquivo deveria ter sido removido")

	list, err := env.Manager.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestPrepareRestore_GeneratesUniqueToken(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	tok1, _, err := env.Manager.PrepareRestore(ctx, b.ID)
	require.NoError(t, err)
	tok2, _, err := env.Manager.PrepareRestore(ctx, b.ID)
	require.NoError(t, err)
	assert.NotEqual(t, tok1, tok2, "cada prepare deve gerar token novo")
	assert.Len(t, tok1, 48) // 24 bytes em hex
}

func TestRestore_RequiresValidToken(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	ctx := context.Background()

	b, err := env.Manager.Create(ctx, "manual")
	require.NoError(t, err)

	err = env.Manager.Restore(ctx, b.ID, "token-errado")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token")
}

// hashFile recomputa sha256 dum arquivo em disco — duplicação propositada,
// não usa código do pacote sob teste.
func hashFile(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	require.NoError(t, err)
	return hex.EncodeToString(h.Sum(nil))
}

// listTarEntries devolve a lista de nomes dentro do .tar.gz.
func listTarEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()
	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
	}
	return names
}
