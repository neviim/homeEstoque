package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/neviim/homeestoque/backend/internal/backup"
)

// BackupHandler agrupa todos os endpoints do módulo de backup.
type BackupHandler struct {
	Manager *backup.Manager
}

// List GET /api/backups
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	backups, err := h.Manager.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": backups})
}

// Create POST /api/backups
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	b, err := h.Manager.Create(r.Context(), "manual")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// Verify POST /api/backups/{id}/verify
func (h *BackupHandler) Verify(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	b, err := h.Manager.Verify(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// Download GET /api/backups/{id}/download
func (h *BackupHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	f, b, err := h.Manager.OpenForDownload(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+b.Filename+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(b.SizeBytes, 10))
	_, _ = io.Copy(w, f)
}

// PrepareRestore POST /api/backups/{id}/restore/prepare
func (h *BackupHandler) PrepareRestore(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	tok, exp, err := h.Manager.PrepareRestore(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"confirm_token": tok,
		"expires_at":    exp.UTC().Format(time.RFC3339),
	})
}

// Restore POST /api/backups/{id}/restore
func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body struct {
		ConfirmToken string `json:"confirm_token"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	if body.ConfirmToken == "" {
		writeError(w, http.StatusBadRequest, "confirm_token obrigatório")
		return
	}
	if err := h.Manager.Restore(r.Context(), id, body.ConfirmToken); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "restored"})
}

// Delete DELETE /api/backups/{id}
func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.Manager.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
}

// GetSchedule GET /api/backup/schedule
func (h *BackupHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	s, err := h.Manager.GetSchedule(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// UpdateSchedule PUT /api/backup/schedule
func (h *BackupHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var in backup.Schedule
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	out, err := h.Manager.UpdateSchedule(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("id inválido: %v", err))
		return 0, false
	}
	return id, true
}
