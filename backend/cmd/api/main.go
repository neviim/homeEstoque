package main

import (
	"log"
	"net/http"
	"os"

	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/server"
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

	handler := server.BuildRouter(db, cfg, server.Options{})

	addr := ":" + cfg.Port
	log.Printf("HomeEstoque API rodando em http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
