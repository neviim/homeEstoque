// Package server monta o chi.Router compartilhado entre o binário de produção
// (cmd/api) e os testes de integração. Centralizar aqui evita drift entre o
// que roda em prod e o que é exercitado pelos httptest.Server.
package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/neviim/homeestoque/backend/internal/backup"
	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/handlers"
	"github.com/neviim/homeestoque/backend/internal/middleware"
)

// Options permite customizar o router em cenários de teste — por exemplo,
// desabilitar o middleware Logger para evitar ruído no `go test -v`.
type Options struct {
	DisableLogger bool
	// BackupManager habilita as rotas /api/backups e /api/backup/schedule.
	// Quando nil, essas rotas não são registradas (útil em testes que não
	// exercitam o módulo de backup).
	BackupManager *backup.Manager
}

// BuildRouter monta o stack completo de middlewares e rotas.
func BuildRouter(db *sql.DB, cfg *config.Config, opts Options) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	if !opts.DisableLogger {
		r.Use(chimw.Logger)
	}
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

	// Maintenance gate: durante restore, bloqueia toda API exceto /health e a
	// rota de restore em si. Só ativa quando BackupManager está presente.
	if opts.BackupManager != nil {
		r.Use(middleware.MaintenanceGate(
			opts.BackupManager.IsInMaintenance,
			[]string{"/api/backups/"},
		))
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authH := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret}
	catH := &handlers.CategoryHandler{DB: db}
	locH := &handlers.LocationHandler{DB: db}
	itemH := &handlers.ItemHandler{DB: db, UploadDir: cfg.UploadDir}
	exH := &handlers.ExtraHandler{DB: db}
	userH := &handlers.UserHandler{DB: db}
	roleH := &handlers.RolesHandler{DB: db}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"homeestoque-api"}`))
	})

	fs := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))
	r.Handle("/uploads/*", fs)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Get("/items/{id}/qrcode", itemH.QRCode)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))

			r.Get("/auth/me", authH.Me)
			r.Put("/auth/profile", authH.UpdateProfile)
			r.Put("/auth/password", authH.ChangePassword)
			r.Get("/permissions", roleH.ListCatalog)
			r.Get("/roles", roleH.List)

			r.With(middleware.RequirePermission(db, "dashboard.view")).Get("/dashboard", exH.Dashboard)

			r.With(middleware.RequirePermission(db, "items.view")).Get("/items", itemH.List)
			r.With(middleware.RequirePermission(db, "items.view")).Get("/items/{id}", itemH.Get)
			r.With(middleware.RequirePermission(db, "items.view")).Get("/items/{id}/movements", itemH.Movements)
			r.With(middleware.RequirePermission(db, "items.create")).Post("/items", itemH.Create)
			r.With(middleware.RequirePermission(db, "items.update")).Put("/items/{id}", itemH.Update)
			r.With(middleware.RequirePermission(db, "items.delete")).Delete("/items/{id}", itemH.Delete)
			r.With(middleware.RequirePermission(db, "items.upload_photo")).Post("/items/{id}/photos", itemH.UploadPhoto)
			r.With(middleware.RequirePermission(db, "items.upload_photo")).Delete("/items/{id}/photos/{photoId}", itemH.DeletePhoto)

			r.With(middleware.RequirePermission(db, "categories.view")).Get("/categories", catH.List)
			r.With(middleware.RequirePermission(db, "categories.manage")).Post("/categories", catH.Create)
			r.With(middleware.RequirePermission(db, "categories.manage")).Put("/categories/{id}", catH.Update)
			r.With(middleware.RequirePermission(db, "categories.manage")).Delete("/categories/{id}", catH.Delete)

			r.With(middleware.RequirePermission(db, "locations.view")).Get("/locations", locH.List)
			r.With(middleware.RequirePermission(db, "locations.manage")).Post("/locations", locH.Create)
			r.With(middleware.RequirePermission(db, "locations.manage")).Put("/locations/{id}", locH.Update)
			r.With(middleware.RequirePermission(db, "locations.manage")).Delete("/locations/{id}", locH.Delete)

			r.With(middleware.RequirePermission(db, "movements.view")).Get("/movements", exH.AllMovements)
			r.With(middleware.RequirePermission(db, "movements.view")).Get("/movements/users", exH.MovementUsers)

			r.With(middleware.RequirePermission(db, "export.csv")).Get("/export/csv", exH.ExportCSV)

			r.With(middleware.RequirePermission(db, "users.manage")).Get("/users", userH.List)
			r.With(middleware.RequirePermission(db, "users.manage")).Post("/users", userH.Create)
			r.With(middleware.RequirePermission(db, "users.manage")).Put("/users/{id}", userH.Update)
			r.With(middleware.RequirePermission(db, "users.manage")).Put("/users/{id}/status", userH.UpdateStatus)
			r.With(middleware.RequirePermission(db, "users.manage")).Put("/users/{id}/password", userH.ResetPassword)
			r.With(middleware.RequirePermission(db, "users.manage")).Delete("/users/{id}", userH.Delete)

			r.With(middleware.RequirePermission(db, "roles.manage")).Post("/roles", roleH.Create)
			r.With(middleware.RequirePermission(db, "roles.manage")).Put("/roles/{id}", roleH.Update)
			r.With(middleware.RequirePermission(db, "roles.manage")).Delete("/roles/{id}", roleH.Delete)
			r.With(middleware.RequirePermission(db, "roles.manage")).Put("/roles/{id}/permissions", roleH.UpdatePermissions)

			if opts.BackupManager != nil {
				bkH := &handlers.BackupHandler{Manager: opts.BackupManager}
				r.With(middleware.RequirePermission(db, "backup.create")).Get("/backups", bkH.List)
				r.With(middleware.RequirePermission(db, "backup.create")).Post("/backups", bkH.Create)
				r.With(middleware.RequirePermission(db, "backup.create")).Post("/backups/{id}/verify", bkH.Verify)
				r.With(middleware.RequirePermission(db, "backup.create")).Delete("/backups/{id}", bkH.Delete)
				r.With(middleware.RequirePermission(db, "backup.download")).Get("/backups/{id}/download", bkH.Download)
				r.With(middleware.RequirePermission(db, "backup.restore")).Post("/backups/{id}/restore/prepare", bkH.PrepareRestore)
				r.With(middleware.RequirePermission(db, "backup.restore")).Post("/backups/{id}/restore", bkH.Restore)
				r.With(middleware.RequirePermission(db, "backup.schedule")).Get("/backup/schedule", bkH.GetSchedule)
				r.With(middleware.RequirePermission(db, "backup.schedule")).Put("/backup/schedule", bkH.UpdateSchedule)
			}
		})
	})

	return r
}
