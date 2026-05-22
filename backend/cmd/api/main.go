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

			// Acessível a todos os perfis (inclusive viewer)
			r.Get("/auth/me", authH.Me)
			r.Put("/auth/profile", authH.UpdateProfile)
			r.Put("/auth/password", authH.ChangePassword)

			r.Get("/categories", catH.List)
			r.Get("/locations", locH.List)
			r.Get("/items", itemH.List)
			r.Get("/items/{id}", itemH.Get)
			r.Get("/items/{id}/movements", itemH.Movements)
			r.Get("/movements", exH.AllMovements)
			r.Get("/movements/users", exH.MovementUsers)
			r.Get("/dashboard", exH.Dashboard)
			r.Get("/export/csv", exH.ExportCSV)

			// Operações de escrita — viewer bloqueado
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireWriter(db))
				r.Post("/categories", catH.Create)
				r.Put("/categories/{id}", catH.Update)
				r.Delete("/categories/{id}", catH.Delete)
				r.Post("/locations", locH.Create)
				r.Put("/locations/{id}", locH.Update)
				r.Delete("/locations/{id}", locH.Delete)
				r.Post("/items", itemH.Create)
				r.Put("/items/{id}", itemH.Update)
				r.Delete("/items/{id}", itemH.Delete)
				r.Post("/items/{id}/photos", itemH.UploadPhoto)
				r.Delete("/items/{id}/photos/{photoId}", itemH.DeletePhoto)
			})

			// Somente admin
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin(db))
				r.Get("/users", userH.List)
				r.Post("/users", userH.Create)
				r.Put("/users/{id}", userH.Update)
				r.Put("/users/{id}/status", userH.UpdateStatus)
				r.Put("/users/{id}/password", userH.ResetPassword)
				r.Delete("/users/{id}", userH.Delete)
			})
		})
	})

	addr := ":" + cfg.Port
	log.Printf("HomeEstoque API rodando em http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
