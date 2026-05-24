package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/backup"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestBackup_HTTP_CreateAndList_AdminOK(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	status, body := testutil.Request(t, srv, "POST", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, status, "POST /api/backups: %s", string(body))

	var created backup.Backup
	testutil.DecodeJSON(t, body, &created)
	assert.Equal(t, "manual", created.Type)
	assert.Equal(t, "ok", created.Status)
	assert.NotEmpty(t, created.SHA256)

	status, body = testutil.Request(t, srv, "GET", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusOK, status, string(body))

	var listResp struct {
		Backups []backup.Backup `json:"backups"`
	}
	testutil.DecodeJSON(t, body, &listResp)
	require.Len(t, listResp.Backups, 1)
	assert.Equal(t, created.ID, listResp.Backups[0].ID)
}

func TestBackup_HTTP_NonAdminGets403(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	user := testutil.CreateUser(t, env.DB, "User", "u@x.com", "pw", "user")

	status, _ := testutil.Request(t, srv, "POST", "/api/backups", user.Token, nil)
	assert.Equal(t, http.StatusForbidden, status)

	status, _ = testutil.Request(t, srv, "GET", "/api/backups", user.Token, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestBackup_HTTP_VerifyEndpoint(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	status, body := testutil.Request(t, srv, "POST", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	var created backup.Backup
	testutil.DecodeJSON(t, body, &created)

	status, body = testutil.Request(t, srv, "POST",
		fmt.Sprintf("/api/backups/%d/verify", created.ID), admin.Token, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var verified backup.Backup
	testutil.DecodeJSON(t, body, &verified)
	assert.Equal(t, "ok", verified.Status)
	assert.NotNil(t, verified.VerifiedAt)
}

func TestBackup_HTTP_DownloadEndpoint(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	status, body := testutil.Request(t, srv, "POST", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	var created backup.Backup
	testutil.DecodeJSON(t, body, &created)

	status, body = testutil.Request(t, srv, "GET",
		fmt.Sprintf("/api/backups/%d/download", created.ID), admin.Token, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, created.SizeBytes, int64(len(body)))
}

func TestBackup_HTTP_RestoreRequiresPrepare(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	// Cria backup
	status, body := testutil.Request(t, srv, "POST", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	var created backup.Backup
	testutil.DecodeJSON(t, body, &created)

	// Tentar restore sem prepare → 400
	status, _ = testutil.Request(t, srv, "POST",
		fmt.Sprintf("/api/backups/%d/restore", created.ID), admin.Token,
		map[string]string{"confirm_token": "qualquer"})
	assert.Equal(t, http.StatusBadRequest, status)

	// Prepare → 200 com token
	status, body = testutil.Request(t, srv, "POST",
		fmt.Sprintf("/api/backups/%d/restore/prepare", created.ID), admin.Token, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var prep struct {
		ConfirmToken string `json:"confirm_token"`
	}
	testutil.DecodeJSON(t, body, &prep)
	assert.NotEmpty(t, prep.ConfirmToken)

	// Restore com token correto → 200
	status, body = testutil.Request(t, srv, "POST",
		fmt.Sprintf("/api/backups/%d/restore", created.ID), admin.Token,
		map[string]string{"confirm_token": prep.ConfirmToken})
	require.Equal(t, http.StatusOK, status, string(body))
	assert.True(t, *env.RestartCalled, "restart deveria ter sido acionado")
}

func TestBackup_HTTP_DeleteEndpoint(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	status, body := testutil.Request(t, srv, "POST", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	var created backup.Backup
	testutil.DecodeJSON(t, body, &created)

	status, _ = testutil.Request(t, srv, "DELETE",
		fmt.Sprintf("/api/backups/%d", created.ID), admin.Token, nil)
	assert.Equal(t, http.StatusOK, status)

	// Lista vazia agora
	status, body = testutil.Request(t, srv, "GET", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusOK, status)
	var list struct {
		Backups []backup.Backup `json:"backups"`
	}
	testutil.DecodeJSON(t, body, &list)
	assert.Empty(t, list.Backups)
}

func TestBackup_HTTP_GetAndUpdateSchedule(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	// GET inicial
	status, body := testutil.Request(t, srv, "GET", "/api/backup/schedule", admin.Token, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var initial backup.Schedule
	testutil.DecodeJSON(t, body, &initial)
	assert.False(t, initial.Enabled, "default deve ser disabled")

	// PUT habilita diário 02:00 ret=10
	update := map[string]any{
		"enabled":         true,
		"frequency":       "daily",
		"time_of_day":     "02:00",
		"retention_count": 10,
	}
	status, body = testutil.Request(t, srv, "PUT", "/api/backup/schedule", admin.Token, update)
	require.Equal(t, http.StatusOK, status, string(body))
	var got backup.Schedule
	testutil.DecodeJSON(t, body, &got)
	assert.True(t, got.Enabled)
	assert.Equal(t, "daily", got.Frequency)
	assert.Equal(t, "02:00", got.TimeOfDay)
	assert.Equal(t, 10, got.RetentionCount)
}

func TestBackup_HTTP_MaintenanceModeBlocksOtherEndpoints(t *testing.T) {
	env := testutil.NewBackupEnv(t)
	srv := env.NewServer(t)
	admin := testutil.CreateUser(t, env.DB, "Admin", "admin@x.com", "pw", "admin")

	// Manualmente cria backup e dispara restore — restart só seta flag.
	status, body := testutil.Request(t, srv, "POST", "/api/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	var created backup.Backup
	testutil.DecodeJSON(t, body, &created)

	status, body = testutil.Request(t, srv, "POST",
		fmt.Sprintf("/api/backups/%d/restore/prepare", created.ID), admin.Token, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var prep struct {
		ConfirmToken string `json:"confirm_token"`
	}
	testutil.DecodeJSON(t, body, &prep)

	status, _ = testutil.Request(t, srv, "POST",
		fmt.Sprintf("/api/backups/%d/restore", created.ID), admin.Token,
		map[string]string{"confirm_token": prep.ConfirmToken})
	require.Equal(t, http.StatusOK, status)

	// Após restore, manager está em maintenance — outras rotas devem dar 503
	status, body = testutil.Request(t, srv, "GET", "/api/items", admin.Token, nil)
	assert.Equal(t, http.StatusServiceUnavailable, status, string(body))
}

// json import-keepalive (algumas refactorings deixam acidentalmente um import sem uso)
var _ = json.Valid
