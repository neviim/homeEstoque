package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/handlers"
	"github.com/neviim/homeestoque/backend/internal/middleware"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	if err := database.Seed(db); err != nil {
		log.Printf("seed warning: %v", err)
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

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
		w.Write([]byte(`{"status":"ok","service":"homeestoque-api"}`))
	})

	fs := http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))
	r.Handle("/uploads/*", fs)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Get("/items/{id}/qrcode", itemH.QRCode)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))

			// Rotas de perfil próprio + catálogos para a UI — acessíveis a qualquer logado
			r.Get("/auth/me", authH.Me)
			r.Put("/auth/profile", authH.UpdateProfile)
			r.Put("/auth/password", authH.ChangePassword)
			r.Get("/permissions", roleH.ListCatalog)
			r.Get("/roles", roleH.List)

			// Dashboard
			r.With(middleware.RequirePermission(db, "dashboard.view")).Get("/dashboard", exH.Dashboard)

			// Itens
			r.With(middleware.RequirePermission(db, "items.view")).Get("/items", itemH.List)
			r.With(middleware.RequirePermission(db, "items.view")).Get("/items/{id}", itemH.Get)
			r.With(middleware.RequirePermission(db, "items.view")).Get("/items/{id}/movements", itemH.Movements)
			r.With(middleware.RequirePermission(db, "items.create")).Post("/items", itemH.Create)
			r.With(middleware.RequirePermission(db, "items.update")).Put("/items/{id}", itemH.Update)
			r.With(middleware.RequirePermission(db, "items.delete")).Delete("/items/{id}", itemH.Delete)
			r.With(middleware.RequirePermission(db, "items.upload_photo")).Post("/items/{id}/photos", itemH.UploadPhoto)
			r.With(middleware.RequirePermission(db, "items.upload_photo")).Delete("/items/{id}/photos/{photoId}", itemH.DeletePhoto)

			// Categorias
			r.With(middleware.RequirePermission(db, "categories.view")).Get("/categories", catH.List)
			r.With(middleware.RequirePermission(db, "categories.manage")).Post("/categories", catH.Create)
			r.With(middleware.RequirePermission(db, "categories.manage")).Put("/categories/{id}", catH.Update)
			r.With(middleware.RequirePermission(db, "categories.manage")).Delete("/categories/{id}", catH.Delete)

			// Locais
			r.With(middleware.RequirePermission(db, "locations.view")).Get("/locations", locH.List)
			r.With(middleware.RequirePermission(db, "locations.manage")).Post("/locations", locH.Create)
			r.With(middleware.RequirePermission(db, "locations.manage")).Put("/locations/{id}", locH.Update)
			r.With(middleware.RequirePermission(db, "locations.manage")).Delete("/locations/{id}", locH.Delete)

			// Movimentações
			r.With(middleware.RequirePermission(db, "movements.view")).Get("/movements", exH.AllMovements)
			r.With(middleware.RequirePermission(db, "movements.view")).Get("/movements/users", exH.MovementUsers)

			// Exportação
			r.With(middleware.RequirePermission(db, "export.csv")).Get("/export/csv", exH.ExportCSV)

			// Gestão de usuários
			r.With(middleware.RequirePermission(db, "users.manage")).Get("/users", userH.List)
			r.With(middleware.RequirePermission(db, "users.manage")).Post("/users", userH.Create)
			r.With(middleware.RequirePermission(db, "users.manage")).Put("/users/{id}", userH.Update)
			r.With(middleware.RequirePermission(db, "users.manage")).Put("/users/{id}/status", userH.UpdateStatus)
			r.With(middleware.RequirePermission(db, "users.manage")).Put("/users/{id}/password", userH.ResetPassword)
			r.With(middleware.RequirePermission(db, "users.manage")).Delete("/users/{id}", userH.Delete)

			// Gestão de perfis (roles & permissions)
			r.With(middleware.RequirePermission(db, "roles.manage")).Post("/roles", roleH.Create)
			r.With(middleware.RequirePermission(db, "roles.manage")).Put("/roles/{id}", roleH.Update)
			r.With(middleware.RequirePermission(db, "roles.manage")).Delete("/roles/{id}", roleH.Delete)
			r.With(middleware.RequirePermission(db, "roles.manage")).Put("/roles/{id}/permissions", roleH.UpdatePermissions)
		})
	})

	addr := ":" + cfg.Port
	log.Printf("HomeEstoque API rodando em http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
