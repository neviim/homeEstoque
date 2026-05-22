package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_time_format=sqlite")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// addColumn tenta adicionar uma coluna; ignora erro se já existir (SQLite não tem ADD COLUMN IF NOT EXISTS).
func addColumn(db *sql.DB, table, def string) {
	_, _ = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + def)
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		icon TEXT,
		color TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS locations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		parent_id INTEGER REFERENCES locations(id) ON DELETE SET NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT,
		brand TEXT,
		model TEXT,
		serial_number TEXT,
		quantity INTEGER NOT NULL DEFAULT 1,
		unit TEXT DEFAULT 'un',
		purchase_date DATE,
		purchase_price REAL,
		condition TEXT DEFAULT 'novo',
		notes TEXT,
		category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS item_photos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		filename TEXT NOT NULL,
		original_name TEXT,
		size INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS movements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		from_location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL,
		to_location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL,
		quantity INTEGER NOT NULL DEFAULT 1,
		reason TEXT,
		user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS roles (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL UNIQUE,
		label       TEXT NOT NULL,
		description TEXT,
		is_system   INTEGER NOT NULL DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		role_id    INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
		permission TEXT NOT NULL,
		PRIMARY KEY (role_id, permission)
	);

	CREATE INDEX IF NOT EXISTS idx_items_category ON items(category_id);
	CREATE INDEX IF NOT EXISTS idx_items_location ON items(location_id);
	CREATE INDEX IF NOT EXISTS idx_items_name ON items(name);
	CREATE INDEX IF NOT EXISTS idx_movements_item ON movements(item_id);
	CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// Adiciona colunas em DBs existentes (idempotente: ignora erro de coluna duplicada).
	addColumn(db, "users", "role TEXT NOT NULL DEFAULT 'user'")
	addColumn(db, "users", "status TEXT NOT NULL DEFAULT 'active'")
	return nil
}
