package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Schedule é a representação serializável da row singleton em backup_schedule.
type Schedule struct {
	Enabled        bool       `json:"enabled"`
	Frequency      string     `json:"frequency"`   // "daily" | "weekly"
	Weekday        *int       `json:"weekday"`     // 0=Domingo..6=Sábado (apenas se weekly)
	TimeOfDay      string     `json:"time_of_day"` // "HH:MM"
	RetentionCount int        `json:"retention_count"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
}

// Scheduler dispara backups automáticos conforme a Schedule no banco.
type Scheduler struct {
	manager *Manager

	mu      sync.Mutex
	cron    *cron.Cron
	entryID cron.EntryID
	reload  chan struct{}
	stop    chan struct{}
	started bool
}

// StartScheduler instancia o scheduler e o associa ao manager.
func (m *Manager) StartScheduler(ctx context.Context) {
	s := &Scheduler{
		manager: m,
		cron:    cron.New(),
		reload:  make(chan struct{}, 1),
		stop:    make(chan struct{}),
	}
	m.scheduler = s
	s.cron.Start()
	s.applySchedule(ctx)
	s.started = true

	go func() {
		for {
			select {
			case <-s.stop:
				return
			case <-s.reload:
				s.applySchedule(ctx)
			}
		}
	}()
}

// StopScheduler interrompe o ciclo de cron (chamado por shutdown).
func (m *Manager) StopScheduler() {
	if m.scheduler == nil || !m.scheduler.started {
		return
	}
	close(m.scheduler.stop)
	ctx := m.scheduler.cron.Stop()
	<-ctx.Done()
	m.scheduler.started = false
}

// Reload sinaliza pro scheduler reler a configuração do banco.
func (s *Scheduler) Reload() {
	select {
	case s.reload <- struct{}{}:
	default:
	}
}

func (s *Scheduler) applySchedule(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove entry antiga
	if s.entryID != 0 {
		s.cron.Remove(s.entryID)
		s.entryID = 0
	}

	sched, err := loadSchedule(ctx, s.manager.DB())
	if err != nil {
		log.Printf("backup scheduler: load schedule: %v", err)
		return
	}
	if !sched.Enabled {
		_ = updateNextRun(ctx, s.manager.DB(), nil)
		return
	}
	spec, err := cronSpec(sched)
	if err != nil {
		log.Printf("backup scheduler: cronSpec: %v", err)
		return
	}
	id, err := s.cron.AddFunc(spec, func() { s.runJob() })
	if err != nil {
		log.Printf("backup scheduler: AddFunc: %v", err)
		return
	}
	s.entryID = id
	entry := s.cron.Entry(id)
	next := entry.Next
	_ = updateNextRun(ctx, s.manager.DB(), &next)
}

// runJob executa o backup automático + retention prune.
func (s *Scheduler) runJob() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if s.manager.IsInMaintenance() {
		log.Printf("backup scheduler: sistema em manutenção, pulando run")
		return
	}

	b, err := s.manager.Create(ctx, "auto")
	if err != nil {
		log.Printf("backup scheduler: create: %v", err)
		return
	}
	log.Printf("backup scheduler: criou backup #%d (%s)", b.ID, b.Filename)

	if err := s.prune(ctx); err != nil {
		log.Printf("backup scheduler: prune: %v", err)
	}

	now := time.Now()
	_, _ = s.manager.DB().ExecContext(ctx,
		`UPDATE backup_schedule SET last_run_at=? WHERE id=1`, now)

	// Atualiza next_run_at
	if s.entryID != 0 {
		next := s.cron.Entry(s.entryID).Next
		_ = updateNextRun(ctx, s.manager.DB(), &next)
	}
}

// prune apaga arquivos auto excedentes além da retenção (mantém os mais recentes).
func (s *Scheduler) prune(ctx context.Context) error {
	sched, err := loadSchedule(ctx, s.manager.DB())
	if err != nil {
		return err
	}
	if sched.RetentionCount <= 0 {
		return nil
	}
	rows, err := s.manager.DB().QueryContext(ctx,
		`SELECT id FROM backups WHERE type='auto' AND status='ok'
		 ORDER BY datetime(created_at) DESC, id DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) <= sched.RetentionCount {
		return nil
	}
	excess := ids[sched.RetentionCount:]
	for _, id := range excess {
		if err := s.manager.Delete(ctx, id); err != nil {
			log.Printf("backup scheduler: prune delete #%d: %v", id, err)
		}
	}
	return nil
}

// loadSchedule lê a row singleton de backup_schedule.
func loadSchedule(ctx context.Context, db *sql.DB) (Schedule, error) {
	var s Schedule
	var weekday sql.NullInt64
	var lastRun, nextRun sql.NullTime
	err := db.QueryRowContext(ctx,
		`SELECT enabled, frequency, weekday, time_of_day, retention_count, last_run_at, next_run_at
		 FROM backup_schedule WHERE id = 1`,
	).Scan(&s.Enabled, &s.Frequency, &weekday, &s.TimeOfDay, &s.RetentionCount, &lastRun, &nextRun)
	if err != nil {
		return s, err
	}
	if weekday.Valid {
		w := int(weekday.Int64)
		s.Weekday = &w
	}
	if lastRun.Valid {
		t := lastRun.Time
		s.LastRunAt = &t
	}
	if nextRun.Valid {
		t := nextRun.Time
		s.NextRunAt = &t
	}
	return s, nil
}

func updateNextRun(ctx context.Context, db *sql.DB, next *time.Time) error {
	if next == nil {
		_, err := db.ExecContext(ctx, `UPDATE backup_schedule SET next_run_at=NULL WHERE id=1`)
		return err
	}
	_, err := db.ExecContext(ctx, `UPDATE backup_schedule SET next_run_at=? WHERE id=1`, *next)
	return err
}

// cronSpec converte uma Schedule num cron expression de 5 campos.
//   "daily 03:00"           → "0 3 * * *"
//   "weekly 0 (dom) 03:00" → "0 3 * * 0"
func cronSpec(s Schedule) (string, error) {
	h, m, err := parseHHMM(s.TimeOfDay)
	if err != nil {
		return "", err
	}
	switch s.Frequency {
	case "daily":
		return fmt.Sprintf("%d %d * * *", m, h), nil
	case "weekly":
		if s.Weekday == nil {
			return "", errors.New("weekly requer weekday")
		}
		if *s.Weekday < 0 || *s.Weekday > 6 {
			return "", errors.New("weekday fora do intervalo 0-6")
		}
		return fmt.Sprintf("%d %d * * %d", m, h, *s.Weekday), nil
	default:
		return "", fmt.Errorf("frequency inválida: %q", s.Frequency)
	}
}

func parseHHMM(s string) (int, int, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("time_of_day inválido: %q", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("hora inválida em %q", s)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("minuto inválido em %q", s)
	}
	return h, m, nil
}

// GetSchedule devolve a Schedule atual (para o handler GET).
func (m *Manager) GetSchedule(ctx context.Context) (Schedule, error) {
	return loadSchedule(ctx, m.DB())
}

// UpdateSchedule persiste mudanças e aciona reload do cron.
func (m *Manager) UpdateSchedule(ctx context.Context, s Schedule) (Schedule, error) {
	// Valida cronSpec antes de gravar
	if s.Enabled {
		if _, err := cronSpec(s); err != nil {
			return Schedule{}, err
		}
	}
	if s.RetentionCount < 1 {
		s.RetentionCount = 1
	}
	if s.RetentionCount > 100 {
		s.RetentionCount = 100
	}
	var weekday any
	if s.Weekday != nil {
		weekday = *s.Weekday
	}
	_, err := m.DB().ExecContext(ctx,
		`UPDATE backup_schedule
		 SET enabled=?, frequency=?, weekday=?, time_of_day=?, retention_count=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=1`,
		boolToInt(s.Enabled), s.Frequency, weekday, s.TimeOfDay, s.RetentionCount,
	)
	if err != nil {
		return Schedule{}, err
	}
	if m.scheduler != nil {
		m.scheduler.Reload()
	}
	return m.GetSchedule(ctx)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
