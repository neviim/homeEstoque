package testutil

import (
	"database/sql"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/neviim/homeestoque/backend/internal/backup"
	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/server"
)

// BackupEnv agrupa tudo o que um teste de backup precisa: o DB seedado, o
// caminho real do arquivo SQLite (necessário para verify/restore), o diretório
// de uploads, o manager e uma flag que indica se restart foi chamado.
type BackupEnv struct {
	DB            *sql.DB
	DBPath        string
	UploadDir     string
	BackupDir     string
	Manager       *backup.Manager
	RestartCalled *bool
}

// NewBackupEnv cria um ambiente isolado em t.TempDir() com DB seedado e
// manager pronto. O restart stub apenas seta a flag — não chama os.Exit.
func NewBackupEnv(t *testing.T) *BackupEnv {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Seed(db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	uploadDir := filepath.Join(dir, "uploads")
	if err := osMkdirAll(uploadDir, 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	backupDir := filepath.Join(dir, "backups")

	cfg := &config.Config{DBPath: dbPath, UploadDir: uploadDir, BackupDir: backupDir}
	called := false
	mgr, err := backup.NewManager(db, cfg, func() { called = true })
	if err != nil {
		t.Fatalf("backup.NewManager: %v", err)
	}
	return &BackupEnv{
		DB:            db,
		DBPath:        dbPath,
		UploadDir:     uploadDir,
		BackupDir:     backupDir,
		Manager:       mgr,
		RestartCalled: &called,
	}
}

// NewServer monta um httptest.Server completo com o BackupManager wired.
func (e *BackupEnv) NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := &config.Config{
		Port:        "0",
		DBPath:      e.DBPath,
		JWTSecret:   TestJWTSecret,
		UploadDir:   e.UploadDir,
		BackupDir:   e.BackupDir,
		CORSOrigins: []string{"*"},
	}
	handler := server.BuildRouter(e.DB, cfg, server.Options{
		DisableLogger: true,
		BackupManager: e.Manager,
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}
