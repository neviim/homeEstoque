package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/server"
	"github.com/neviim/homeestoque/backend/internal/testutil"
	"github.com/neviim/homeestoque/backend/internal/version"
)

func newVersionServer(t *testing.T, repoRoot string, restartFn func()) *httptest.Server {
	t.Helper()
	db := testutil.NewSeededTestDB(t)
	cfg := &config.Config{
		Port:        "0",
		JWTSecret:   testutil.TestJWTSecret,
		CORSOrigins: []string{"*"},
	}
	handler := server.BuildRouter(db, cfg, server.Options{
		DisableLogger: true,
		RepoRoot:      repoRoot,
		RestartFunc:   restartFn,
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestVersion_GET_ReturnsFields(t *testing.T) {
	version.Running = "0.1.0"
	version.ResetCache()

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.1\n"), 0o644)
	srv := newVersionServer(t, dir, nil)

	status, body := testutil.Request(t, srv, "GET", "/api/version", "", nil)
	require.Equal(t, http.StatusOK, status, string(body))

	var resp struct {
		Running         string `json:"running"`
		Available       string `json:"available"`
		UpdateAvailable bool   `json:"update_available"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "0.1.0", resp.Running)
	assert.Equal(t, "0.1.1", resp.Available)
	assert.True(t, resp.UpdateAvailable)
}

func TestVersion_Apply_WithoutPermission_Returns403(t *testing.T) {
	dir := t.TempDir()
	db := testutil.NewSeededTestDB(t)
	cfg := &config.Config{
		Port:        "0",
		JWTSecret:   testutil.TestJWTSecret,
		CORSOrigins: []string{"*"},
	}
	handler := server.BuildRouter(db, cfg, server.Options{
		DisableLogger: true,
		RepoRoot:      dir,
		RestartFunc:   func() {},
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// user sem permissão system.update
	u := testutil.CreateUser(t, db, "User", "u@x.com", "pw", "user")
	status, _ := testutil.Request(t, srv, "POST", "/api/version/apply", u.Token, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestVersion_Apply_AdminCallsRestartFn(t *testing.T) {
	version.Running = "0.1.0"
	version.ResetCache()

	dir := t.TempDir()
	called := false
	restartFn := func() { called = true }

	db := testutil.NewSeededTestDB(t)
	cfg := &config.Config{
		Port:        "0",
		JWTSecret:   testutil.TestJWTSecret,
		CORSOrigins: []string{"*"},
	}
	handler := server.BuildRouter(db, cfg, server.Options{
		DisableLogger: true,
		RepoRoot:      dir,
		RestartFunc:   restartFn,
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	admin := testutil.CreateUser(t, db, "Admin", "a@x.com", "pw", "admin")
	status, body := testutil.Request(t, srv, "POST", "/api/version/apply", admin.Token, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	assert.True(t, called, "restartFn deve ter sido chamada")

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, true, resp["restarting"])
}
