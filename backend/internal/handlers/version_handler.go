package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/neviim/homeestoque/backend/internal/version"
)

type VersionHandler struct {
	RepoRoot   string
	RestartFn  func()
}

type versionResponse struct {
	Running         string `json:"running"`
	Available       string `json:"available"`
	UpdateAvailable bool   `json:"update_available"`
}

func (h *VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	avail := version.Available(h.RepoRoot)
	resp := versionResponse{
		Running:         version.Running,
		Available:       avail,
		UpdateAvailable: version.IsUpdateAvailable(h.RepoRoot),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *VersionHandler) Apply(w http.ResponseWriter, r *http.Request) {
	if h.RestartFn == nil {
		http.Error(w, "restart não configurado", http.StatusNotImplemented)
		return
	}
	h.RestartFn()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"restarting":true}`))
}
