package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/neviim/homeestoque/backend/internal/backup"
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

	// Após restore, o módulo de backup pede um restart do processo. Em dev o
	// Air relança; em prod o supervisor (systemd, docker restart=always) faz
	// o mesmo. Sai com código 0 (saída intencional, não erro).
	restart := func() {
		go func() {
			time.Sleep(800 * time.Millisecond)
			log.Printf("backup: restore concluído, reiniciando processo")
			os.Exit(0)
		}()
	}

	bkMgr, err := backup.NewManager(db, cfg, restart)
	if err != nil {
		log.Fatalf("backup manager: %v", err)
	}
	bkMgr.StartScheduler(context.Background())
	defer bkMgr.StopScheduler()

	handler := server.BuildRouter(db, cfg, server.Options{BackupManager: bkMgr})

	addr := ":" + cfg.Port
	log.Printf("HomeEstoque API rodando em http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
