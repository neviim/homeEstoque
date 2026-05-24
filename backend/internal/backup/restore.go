package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Restore aplica o backup `id` no sistema. Fluxo:
//  1. Valida token (uso único, TTL 5min)
//  2. Cria snapshot de segurança pre-restore
//  3. Liga maintenance-mode (middleware passa a retornar 503 pras outras rotas)
//  4. Fecha o *sql.DB atual e extrai os arquivos do .tar.gz por cima
//  5. Chama m.restart() — em produção faz os.Exit(0) numa goroutine após
//     responder o HTTP. Supervisor (systemd, docker, Air em dev) reinicia o
//     processo que vai abrir o novo banco.
//
// O snapshot pre-restore permanece no disco como rede de segurança caso o
// admin precise reverter.
func (m *Manager) Restore(ctx context.Context, id int64, token string) error {
	if err := m.consumeToken(id, token); err != nil {
		return err
	}
	b, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(m.cfg.BackupDir, b.Filename)
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("arquivo não encontrado: %w", err)
	}

	// 1. Snapshot pre-restore (rede de segurança)
	safety, err := m.Create(ctx, "auto")
	if err != nil {
		return fmt.Errorf("snapshot pre-restore: %w", err)
	}
	if safety != nil {
		_, _ = m.DB().ExecContext(ctx,
			`UPDATE backups SET notes='snapshot pre-restore' WHERE id=?`, safety.ID)
	}

	// 2. Entra em maintenance (middleware bloqueia novas requests)
	m.maintenance.Store(true)

	// Pequena janela pra requests em flight terminarem
	time.Sleep(200 * time.Millisecond)

	// 3. Fecha pool atual (após maintenance ON; novas requests dão 503)
	_ = m.db.Close()

	// 4. Extrai arquivos sobre os atuais
	if err := extractArchive(archivePath, m.cfg.DBPath, m.cfg.UploadDir); err != nil {
		// Não temos como reverter aqui sem reiniciar — deixa maintenance ON
		// e propaga o erro. O snapshot pre-restore continua no disco.
		return fmt.Errorf("extract: %w", err)
	}

	// 5. Pede restart ao supervisor (cmd/api faz os.Exit em goroutine).
	if m.restart != nil {
		m.restart()
	}
	return nil
}

// extractArchive lê o .tar.gz e restaura:
//   - entrada `db/homeestoque.db` → dbPath (override do arquivo atual)
//   - entradas `uploads/<rel>`    → uploadDir/<rel> (limpa uploadDir antes)
func extractArchive(archivePath, dbPath, uploadDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	// Limpa uploads/ antes de extrair pra não deixar arquivos órfãos da versão atual.
	if uploadDir != "" {
		if err := os.RemoveAll(uploadDir); err != nil {
			return fmt.Errorf("clear uploads: %w", err)
		}
		if err := os.MkdirAll(uploadDir, 0o755); err != nil {
			return fmt.Errorf("recreate uploads: %w", err)
		}
	}

	// Remove arquivos sidecar do WAL/SHM antes de substituir o DB.
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("db dir: %w", err)
	}

	foundDB := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}
		clean := filepath.ToSlash(filepath.Clean(hdr.Name))

		// Proteção contra tar slip
		if strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
			return fmt.Errorf("entrada de tar inválida: %s", hdr.Name)
		}

		switch {
		case clean == "db/homeestoque.db":
			if err := writeFile(dbPath, tr, hdr.Mode); err != nil {
				return fmt.Errorf("write db: %w", err)
			}
			foundDB = true

		case strings.HasPrefix(clean, "uploads/"):
			if uploadDir == "" {
				continue
			}
			rel := strings.TrimPrefix(clean, "uploads/")
			dest := filepath.Join(uploadDir, rel)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return fmt.Errorf("upload mkdir: %w", err)
			}
			if err := writeFile(dest, tr, hdr.Mode); err != nil {
				return fmt.Errorf("write upload %s: %w", rel, err)
			}
		}
	}
	if !foundDB {
		return errors.New("archive não contém db/homeestoque.db")
	}
	return nil
}

func writeFile(dest string, r io.Reader, mode int64) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, r); err != nil {
		return err
	}
	if mode > 0 {
		_ = os.Chmod(dest, os.FileMode(mode))
	}
	return nil
}
