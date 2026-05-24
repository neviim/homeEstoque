// Package testutil oferece helpers compartilhados pelos testes de integração:
// SQLite em arquivo temporário (t.TempDir), servidor httptest, geração de
// tokens válidos e fixtures básicas. Mantém os arquivos *_test.go enxutos.
package testutil

import (
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/server"
)

// osMkdirAll — pequeno wrapper só pra evitar shadow do `os` pelo nome do arg
// em escopos que reusam a variável `srv`. Mantém o código denso.
var osMkdirAll = os.MkdirAll

// TestJWTSecret é usado em todos os tokens gerados nos testes — fixo e
// previsível, sem ler do ambiente.
const TestJWTSecret = "test-secret-do-not-use-in-prod"

// NewTestDB abre um SQLite em arquivo temporário (t.TempDir) e roda migrate.
// Não chama Seed automaticamente — testes que precisam dos 3 roles devem
// chamar SeedRoles explicitamente. Limpeza via t.Cleanup.
//
// Observação: usamos arquivo em TempDir em vez de `:memory:` porque o driver
// modernc.org/sqlite tem problemas com `:memory:` + múltiplas conexões abertas
// em paralelo (que o connection pool faz por padrão). Arquivo + WAL não tem
// esse problema e é tão rápido quanto.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// NewSeededTestDB é igual a NewTestDB mas roda Seed() (3 roles + permissions
// + categorias/locais default + MCP user). Use quando o teste depende dos
// perfis admin/user/viewer.
func NewSeededTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := NewTestDB(t)
	if err := database.Seed(db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

// NewTestServer monta um httptest.Server com o stack idêntico ao main.go.
// O JWTSecret é forçado para TestJWTSecret. Sem logger pra não poluir output.
func NewTestServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	srv, _ := NewTestServerWithUploadDir(t, db)
	return srv
}

// NewTestServerWithUploadDir é igual a NewTestServer mas devolve também o
// caminho do diretório de uploads criado — útil em testes de fotos que
// precisam verificar o filesystem.
func NewTestServerWithUploadDir(t *testing.T, db *sql.DB) (*httptest.Server, string) {
	t.Helper()
	uploadDir := filepath.Join(t.TempDir(), "uploads")
	if err := osMkdirAll(uploadDir, 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	cfg := &config.Config{
		Port:        "0",
		DBPath:      "ignored-uses-given-db",
		JWTSecret:   TestJWTSecret,
		UploadDir:   uploadDir,
		CORSOrigins: []string{"*"},
	}
	handler := server.BuildRouter(db, cfg, server.Options{DisableLogger: true})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, uploadDir
}

// TokenFor gera um JWT válido para o user_id/email usando o TestJWTSecret.
func TokenFor(t *testing.T, uid int64, email string) string {
	t.Helper()
	tok, err := auth.GenerateToken(uid, email, TestJWTSecret)
	if err != nil {
		t.Fatalf("gen token: %v", err)
	}
	return tok
}

// MustExec roda Exec e falha o teste se houver erro — usado em setup de fixture.
func MustExec(t *testing.T, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// MustQueryRow roda QueryRow.Scan e falha se houver erro.
func MustQueryRow(t *testing.T, db *sql.DB, query string, args []interface{}, dest ...interface{}) {
	t.Helper()
	if err := db.QueryRow(query, args...).Scan(dest...); err != nil {
		t.Fatalf("queryrow %q: %v", query, err)
	}
}
