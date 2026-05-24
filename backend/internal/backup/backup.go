// Package backup encapsula a criação, verificação e restauração de backups
// completos (banco SQLite + UPLOAD_DIR) em um único arquivo .tar.gz.
//
// O Manager mantém o ponteiro atômico para o *sql.DB ativo (necessário para que
// o restore consiga trocar a conexão sem reiniciar o processo) e também segura
// um flag de modo manutenção que é consultado pelo middleware maintenance.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/neviim/homeestoque/backend/internal/config"
)

// RestartFunc é chamado pelo restore após extrair os arquivos. A implementação
// padrão (em cmd/api) faz os.Exit(0) numa goroutine; o supervisor (Air em dev,
// systemd em prod) reinicia o processo. Testes injetam um stub que apenas
// sinaliza.
type RestartFunc func()

// Backup é a representação serializável de uma linha da tabela backups.
type Backup struct {
	ID         int64     `json:"id"`
	Filename   string    `json:"filename"`
	SizeBytes  int64     `json:"size_bytes"`
	SHA256     string    `json:"sha256"`
	CreatedAt  time.Time `json:"created_at"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	Notes      string    `json:"notes,omitempty"`
}

// restoreToken é um token de confirmação efêmero gerado por PrepareRestore.
type restoreToken struct {
	token     string
	expiresAt time.Time
}

// Manager orquestra o ciclo de vida dos backups.
type Manager struct {
	db          *sql.DB
	cfg         *config.Config
	restart     RestartFunc
	maintenance atomic.Bool

	tokensMu sync.Mutex
	tokens   map[int64]restoreToken

	scheduler *Scheduler
}

// NewManager cria um manager pronto para uso. Garante que BackupDir existe.
// O scheduler ainda precisa ser iniciado via StartScheduler. restart é chamado
// após restore extrair os arquivos; em produção deve fazer os.Exit(0) para que
// o supervisor reinicie o processo com o novo DB.
func NewManager(db *sql.DB, cfg *config.Config, restart RestartFunc) (*Manager, error) {
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	m := &Manager{
		db:      db,
		cfg:     cfg,
		restart: restart,
		tokens:  make(map[int64]restoreToken),
	}
	if err := m.reconcileWithDisk(context.Background()); err != nil {
		return nil, fmt.Errorf("reconcile: %w", err)
	}
	return m, nil
}

// DB devolve a conexão ativa.
func (m *Manager) DB() *sql.DB { return m.db }

// IsInMaintenance é consultado pelo maintenance middleware.
func (m *Manager) IsInMaintenance() bool { return m.maintenance.Load() }

// BackupDir é o diretório onde os arquivos vivem.
func (m *Manager) BackupDir() string { return m.cfg.BackupDir }

// reconcileWithDisk sincroniza linhas da tabela com arquivos físicos:
// arquivos sem row viram orphan; rows sem arquivo viram missing.
func (m *Manager) reconcileWithDisk(ctx context.Context) error {
	db := m.DB()
	if db == nil {
		return nil
	}

	// Marcar como missing rows cujo arquivo sumiu
	rows, err := db.QueryContext(ctx, `SELECT id, filename FROM backups WHERE status != 'missing'`)
	if err != nil {
		return err
	}
	type idfile struct {
		id   int64
		name string
	}
	var existing []idfile
	for rows.Next() {
		var f idfile
		if err := rows.Scan(&f.id, &f.name); err == nil {
			existing = append(existing, f)
		}
	}
	rows.Close()

	known := map[string]bool{}
	for _, e := range existing {
		known[e.name] = true
		if _, err := os.Stat(filepath.Join(m.cfg.BackupDir, e.name)); errors.Is(err, os.ErrNotExist) {
			_, _ = db.ExecContext(ctx, `UPDATE backups SET status='missing' WHERE id=?`, e.id)
		}
	}

	// Detectar arquivos órfãos em disco
	entries, err := os.ReadDir(m.cfg.BackupDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		if known[e.Name()] {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		_, _ = db.ExecContext(ctx,
			`INSERT OR IGNORE INTO backups (filename, size_bytes, sha256, type, status, notes)
			 VALUES (?, ?, '', 'manual', 'orphan', 'detectado no startup')`,
			e.Name(), info.Size(),
		)
	}
	return nil
}

// Create gera um novo backup completo do tipo dado ("manual" ou "auto").
func (m *Manager) Create(ctx context.Context, kind string) (*Backup, error) {
	if kind != "manual" && kind != "auto" {
		return nil, fmt.Errorf("backup type inválido: %q", kind)
	}
	db := m.DB()
	if db == nil {
		return nil, errors.New("db indisponível")
	}

	if err := os.MkdirAll(m.cfg.BackupDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir BackupDir: %w", err)
	}

	// 1. VACUUM INTO num arquivo temporário (snapshot consistente do SQLite)
	tmpDB := filepath.Join(os.TempDir(), "homeestoque-backup-"+uuid.NewString()+".db")
	defer os.Remove(tmpDB)
	// SQLite não suporta placeholder em VACUUM INTO; precisamos escapar e interpolar.
	if _, err := db.ExecContext(ctx, "VACUUM INTO '"+strings.ReplaceAll(tmpDB, "'", "''")+"'"); err != nil {
		return nil, fmt.Errorf("vacuum into: %w", err)
	}

	// 2. Montar .tar.gz com (db/homeestoque.db + uploads/...)
	// Sufixo de nanosegundo evita colisão de UNIQUE(filename) em chamadas
	// rápidas (testes ou agendamento + manual no mesmo segundo).
	now := time.Now().UTC()
	ts := now.Format("20060102-150405") + fmt.Sprintf("-%06d", now.Nanosecond()/1000)
	finalName := fmt.Sprintf("backup-%s-%s.tar.gz", ts, kind)
	tmpFinal := filepath.Join(m.cfg.BackupDir, ".tmp-"+uuid.NewString()+".tar.gz")
	finalPath := filepath.Join(m.cfg.BackupDir, finalName)

	hasher := sha256.New()
	if err := writeTarGz(tmpFinal, tmpDB, m.cfg.UploadDir, hasher); err != nil {
		_ = os.Remove(tmpFinal)
		return nil, err
	}

	info, err := os.Stat(tmpFinal)
	if err != nil {
		_ = os.Remove(tmpFinal)
		return nil, err
	}

	if err := os.Rename(tmpFinal, finalPath); err != nil {
		_ = os.Remove(tmpFinal)
		return nil, fmt.Errorf("rename: %w", err)
	}

	digest := hex.EncodeToString(hasher.Sum(nil))

	res, err := db.ExecContext(ctx,
		`INSERT INTO backups (filename, size_bytes, sha256, type, status) VALUES (?, ?, ?, ?, 'ok')`,
		finalName, info.Size(), digest, kind)
	if err != nil {
		// Mantém o arquivo no disco — reconcile vai pegar como orphan na próxima.
		return nil, fmt.Errorf("insert backup row: %w", err)
	}
	id, _ := res.LastInsertId()

	return m.GetByID(ctx, id)
}

// writeTarGz escreve um arquivo .tar.gz contendo:
//   - db/homeestoque.db (cópia do snapshot)
//   - uploads/<rel>     (todos os arquivos sob uploadDir)
//
// O hasher recebe os bytes via io.MultiWriter para que o sha256 do .tar.gz seja
// calculado no mesmo passe da escrita.
func writeTarGz(outPath, dbSnapshotPath, uploadDir string, hasher io.Writer) (err error) {
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create tar: %w", err)
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	mw := io.MultiWriter(out, hasher)
	gw := gzip.NewWriter(mw)
	tw := tar.NewWriter(gw)

	// (a) snapshot do DB
	if err := addFileToTar(tw, dbSnapshotPath, "db/homeestoque.db"); err != nil {
		return err
	}

	// (b) walk uploadDir (se existir)
	if _, serr := os.Stat(uploadDir); serr == nil {
		err := filepath.Walk(uploadDir, func(p string, info os.FileInfo, werr error) error {
			if werr != nil {
				return werr
			}
			if info.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(uploadDir, p)
			if rerr != nil {
				return rerr
			}
			return addFileToTar(tw, p, filepath.ToSlash(filepath.Join("uploads", rel)))
		})
		if err != nil {
			return fmt.Errorf("walk uploads: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}

func addFileToTar(tw *tar.Writer, srcPath, arcName string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    arcName,
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

// GetByID busca um backup pela primary key.
func (m *Manager) GetByID(ctx context.Context, id int64) (*Backup, error) {
	b := &Backup{}
	var verifiedAt sql.NullTime
	var notes sql.NullString
	err := m.DB().QueryRowContext(ctx,
		`SELECT id, filename, size_bytes, sha256, created_at, type, status, verified_at, notes
		 FROM backups WHERE id = ?`, id,
	).Scan(&b.ID, &b.Filename, &b.SizeBytes, &b.SHA256, &b.CreatedAt, &b.Type, &b.Status, &verifiedAt, &notes)
	if err != nil {
		return nil, err
	}
	if verifiedAt.Valid {
		t := verifiedAt.Time
		b.VerifiedAt = &t
	}
	if notes.Valid {
		b.Notes = notes.String
	}
	return b, nil
}

// List devolve todos os backups conhecidos, mais recentes primeiro.
func (m *Manager) List(ctx context.Context) ([]Backup, error) {
	rows, err := m.DB().QueryContext(ctx,
		`SELECT id, filename, size_bytes, sha256, created_at, type, status, verified_at, notes
		 FROM backups ORDER BY datetime(created_at) DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Backup{}
	for rows.Next() {
		var b Backup
		var verifiedAt sql.NullTime
		var notes sql.NullString
		if err := rows.Scan(&b.ID, &b.Filename, &b.SizeBytes, &b.SHA256, &b.CreatedAt,
			&b.Type, &b.Status, &verifiedAt, &notes); err != nil {
			return nil, err
		}
		if verifiedAt.Valid {
			t := verifiedAt.Time
			b.VerifiedAt = &t
		}
		if notes.Valid {
			b.Notes = notes.String
		}
		out = append(out, b)
	}
	return out, nil
}

// Verify recomputa o sha256, confere com o armazenado e valida o conteúdo do
// .tar.gz (lista entradas + checa PRAGMA integrity_check do banco embutido).
func (m *Manager) Verify(ctx context.Context, id int64) (*Backup, error) {
	b, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(m.cfg.BackupDir, b.Filename)

	status, notes := verifyArchive(path, b.SHA256)
	_, _ = m.DB().ExecContext(ctx,
		`UPDATE backups SET status=?, notes=?, verified_at=CURRENT_TIMESTAMP WHERE id=?`,
		status, notes, id,
	)
	return m.GetByID(ctx, id)
}

// verifyArchive faz as 3 checagens; retorna (status, notes) para persistir.
func verifyArchive(path, expectedSHA string) (string, string) {
	f, err := os.Open(path)
	if err != nil {
		return "missing", "arquivo não encontrado em disco"
	}
	defer f.Close()

	// 1. sha256
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "corrupted", "falha ao ler arquivo: " + err.Error()
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if expectedSHA != "" && got != expectedSHA {
		return "corrupted", "sha256 não confere"
	}

	// 2. validar tar.gz + extrair DB para temp
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "corrupted", "seek: " + err.Error()
	}
	gr, err := gzip.NewReader(f)
	if err != nil {
		return "corrupted", "gzip inválido: " + err.Error()
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	tmpDir, err := os.MkdirTemp("", "homeestoque-verify-")
	if err != nil {
		return "corrupted", "tmpdir: " + err.Error()
	}
	defer os.RemoveAll(tmpDir)
	var dbPath string

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "corrupted", "tar entry: " + err.Error()
		}
		if hdr.Name == "db/homeestoque.db" {
			dbPath = filepath.Join(tmpDir, "homeestoque.db")
			out, err := os.Create(dbPath)
			if err != nil {
				return "corrupted", "extract db: " + err.Error()
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "corrupted", "copy db: " + err.Error()
			}
			out.Close()
		}
	}
	if dbPath == "" {
		return "corrupted", "arquivo sem db/homeestoque.db"
	}

	// 3. PRAGMA integrity_check num conexão throwaway
	tdb, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(DELETE)")
	if err != nil {
		return "corrupted", "open extracted db: " + err.Error()
	}
	defer tdb.Close()
	var ok string
	if err := tdb.QueryRow("PRAGMA integrity_check").Scan(&ok); err != nil {
		return "corrupted", "integrity_check falhou: " + err.Error()
	}
	if ok != "ok" {
		return "corrupted", "integrity_check: " + ok
	}
	return "ok", ""
}

// Delete remove o arquivo do disco e a linha da tabela.
func (m *Manager) Delete(ctx context.Context, id int64) error {
	b, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(m.cfg.BackupDir, b.Filename))
	_, err = m.DB().ExecContext(ctx, `DELETE FROM backups WHERE id=?`, id)
	return err
}

// OpenForDownload retorna o file handle do .tar.gz pronto pra stream.
func (m *Manager) OpenForDownload(ctx context.Context, id int64) (*os.File, *Backup, error) {
	b, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if b.Status == "missing" {
		return nil, nil, errors.New("arquivo indisponível em disco")
	}
	path := filepath.Join(m.cfg.BackupDir, b.Filename)
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, b, nil
}

// PrepareRestore gera um token de confirmação de uso único válido por 5 minutos.
func (m *Manager) PrepareRestore(ctx context.Context, id int64) (string, time.Time, error) {
	b, err := m.GetByID(ctx, id)
	if err != nil {
		return "", time.Time{}, err
	}
	if b.Status == "missing" || b.Status == "corrupted" {
		return "", time.Time{}, fmt.Errorf("backup com status %q não pode ser restaurado", b.Status)
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	tok := hex.EncodeToString(raw)
	exp := time.Now().Add(5 * time.Minute)

	m.tokensMu.Lock()
	m.tokens[id] = restoreToken{token: tok, expiresAt: exp}
	m.tokensMu.Unlock()

	return tok, exp, nil
}

func (m *Manager) consumeToken(id int64, presented string) error {
	m.tokensMu.Lock()
	defer m.tokensMu.Unlock()
	t, ok := m.tokens[id]
	if !ok {
		return errors.New("nenhum token de restore preparado para este backup")
	}
	delete(m.tokens, id)
	if time.Now().After(t.expiresAt) {
		return errors.New("token de restore expirou — solicite um novo")
	}
	if t.token != presented {
		return errors.New("token inválido")
	}
	return nil
}
