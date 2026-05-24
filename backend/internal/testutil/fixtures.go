package testutil

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/neviim/homeestoque/backend/internal/auth"
)

// TestUser representa um usuário criado por CreateUser — facilita acessar
// o id e o token nos testes.
type TestUser struct {
	ID    int64
	Name  string
	Email string
	Role  string
	Token string
}

// CreateUser insere um usuário com senha bcrypt válida e gera um token JWT.
// O DB precisa ter sido seedado (NewSeededTestDB) para que o `role` seja válido.
func CreateUser(t *testing.T, db *sql.DB, name, email, password, role string) *TestUser {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO users (name, email, password_hash, role, status) VALUES (?, ?, ?, ?, 'active')`,
		name, email, hash, role,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, _ := res.LastInsertId()
	return &TestUser{
		ID:    id,
		Name:  name,
		Email: email,
		Role:  role,
		Token: TokenFor(t, id, email),
	}
}

// CreateUserWithStatus permite controlar o status (pending/inactive/active).
func CreateUserWithStatus(t *testing.T, db *sql.DB, name, email, password, role, status string) *TestUser {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO users (name, email, password_hash, role, status) VALUES (?, ?, ?, ?, ?)`,
		name, email, hash, role, status,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, _ := res.LastInsertId()
	return &TestUser{
		ID:    id,
		Name:  name,
		Email: email,
		Role:  role,
		Token: TokenFor(t, id, email),
	}
}

// CreateCategory insere uma categoria de teste e devolve seu id.
func CreateCategory(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO categories (name, color) VALUES (?, ?)`, name, "#000")
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// CreateLocation insere uma localização de teste e devolve seu id.
func CreateLocation(t *testing.T, db *sql.DB, name, typ string, parentID *int64) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO locations (name, type, parent_id) VALUES (?, ?, ?)`, name, typ, parentID)
	if err != nil {
		t.Fatalf("insert location: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// CreateItem insere um item de teste (campos mínimos + opcionais via opts).
type ItemOpts struct {
	Description string
	Brand       string
	Model       string
	Quantity    int
	Unit        string
	Condition   string
	CategoryID  *int64
	LocationID  *int64
	Price       *float64
}

func CreateItem(t *testing.T, db *sql.DB, name string, opts ItemOpts) int64 {
	t.Helper()
	if opts.Quantity == 0 {
		opts.Quantity = 1
	}
	if opts.Unit == "" {
		opts.Unit = "un"
	}
	if opts.Condition == "" {
		opts.Condition = "novo"
	}
	// Gera code único usando UUID (similar ao handler real) para evitar conflito
	// de UNIQUE quando o mesmo `name` é usado várias vezes num teste.
	code := "EST-" + strings.ToUpper(uuid.New().String()[:8])
	res, err := db.Exec(`INSERT INTO items
		(code, name, description, brand, model, quantity, unit, condition,
		 category_id, location_id, purchase_price)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		code, name, opts.Description, opts.Brand, opts.Model, opts.Quantity,
		opts.Unit, opts.Condition, opts.CategoryID, opts.LocationID, opts.Price)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}
