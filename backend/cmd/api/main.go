package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/neviim/homeestoque/backend/internal/backup"
	"github.com/neviim/homeestoque/backend/internal/config"
	"github.com/neviim/homeestoque/backend/internal/database"
	"github.com/neviim/homeestoque/backend/internal/server"
	internalversion "github.com/neviim/homeestoque/backend/internal/version"
)

// version é injetado pelo ldflags em build-time: -X main.version=X.Y.Z
var version = "dev"

// findRepoRoot sobe a partir do executável até encontrar o arquivo VERSION.
func findRepoRoot() string {
	start := "."
	if exe, err := os.Executable(); err == nil {
		start = filepath.Dir(exe)
	}
	dir := start
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "VERSION")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return start
}

func main() {
	internalversion.Running = version

	// repoRoot: sobe a partir do executável procurando o arquivo VERSION.
	// Funciona tanto em dev (Air: backend/tmp/main) quanto em prod (bin/api).
	repoRoot := findRepoRoot()

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

	handler := server.BuildRouter(db, cfg, server.Options{
		BackupManager: bkMgr,
		RepoRoot:      repoRoot,
		RestartFunc:   restart,
	})

	addr := ":" + cfg.Port
	log.Printf("HomeEstoque API rodando em http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
