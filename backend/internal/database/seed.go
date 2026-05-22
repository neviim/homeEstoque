package database

import (
	"database/sql"

	"github.com/neviim/homeestoque/backend/internal/permissions"
)

// MCPUserEmail é o email do usuário sintético usado pelo MCP server
// para registrar autoria em movements. Login HTTP nunca será feito por ele.
const MCPUserEmail = "mcp@homeestoque.local"

// seedRoles garante que os 3 perfis semente existam e que o admin sempre
// tenha TODAS as permissões do catálogo atual (idempotente).
func seedRoles(db *sql.DB) error {
	// 1. Cria os 3 perfis se ainda não existirem
	seedRolesData := []struct {
		name, label, description string
		isSystem                 int
	}{
		{"admin", "Administrador", "Acesso total ao sistema. Não pode ser editado nem excluído.", 1},
		{"user", "Usuário", "Pode gerenciar itens, categorias, locais e movimentações.", 0},
		{"viewer", "Visualizador", "Acesso somente leitura ao dashboard e aos itens.", 0},
	}
	for _, r := range seedRolesData {
		_, _ = db.Exec(
			`INSERT OR IGNORE INTO roles (name, label, description, is_system) VALUES (?, ?, ?, ?)`,
			r.name, r.label, r.description, r.isSystem,
		)
	}

	// 2. Admin sempre recebe TODAS as permissões do catálogo atual
	var adminID int64
	if err := db.QueryRow(`SELECT id FROM roles WHERE name = 'admin'`).Scan(&adminID); err != nil {
		return err
	}
	for _, key := range permissions.Keys() {
		_, _ = db.Exec(
			`INSERT OR IGNORE INTO role_permissions (role_id, permission) VALUES (?, ?)`,
			adminID, key,
		)
	}

	// 3. Seed inicial de permissões para user e viewer — apenas se o role
	//    ainda não tem nenhuma permissão (não sobrescreve configuração do admin)
	seedRoleIfEmpty(db, "user", []string{
		"dashboard.view",
		"items.view", "items.create", "items.update", "items.delete", "items.upload_photo",
		"categories.view", "categories.manage",
		"locations.view", "locations.manage",
		"movements.view",
		"export.csv",
	})
	seedRoleIfEmpty(db, "viewer", []string{
		"dashboard.view",
		"items.view",
	})
	return nil
}

func seedRoleIfEmpty(db *sql.DB, roleName string, perms []string) {
	var roleID int64
	var count int
	if err := db.QueryRow(`SELECT id FROM roles WHERE name = ?`, roleName).Scan(&roleID); err != nil {
		return
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM role_permissions WHERE role_id = ?`, roleID).Scan(&count)
	if count > 0 {
		return
	}
	for _, p := range perms {
		_, _ = db.Exec(
			`INSERT OR IGNORE INTO role_permissions (role_id, permission) VALUES (?, ?)`,
			roleID, p,
		)
	}
}

func Seed(db *sql.DB) error {
	// Roles + permissões (ordem importa: precisa existir antes de qualquer user)
	if err := seedRoles(db); err != nil {
		return err
	}

	// Garante usuário MCP com role=user, status=inactive (nunca deve fazer login).
	_, _ = db.Exec(
		`INSERT OR IGNORE INTO users (name, email, password_hash, role, status) VALUES (?, ?, ?, 'user', 'inactive')`,
		"MCP Assistant", MCPUserEmail, "!disabled!",
	)
	// Garante que o primeiro usuário humano é sempre admin+active (idempotente).
	_, _ = db.Exec(`
		UPDATE users SET role = 'admin', status = 'active'
		WHERE id = (SELECT MIN(id) FROM users WHERE email != ?)
	`, MCPUserEmail)

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if count > 0 {
		return nil
	}

	categories := []struct {
		name, icon, color string
	}{
		{"Eletrônicos", "cpu", "#3b82f6"},
		{"Computadores", "monitor", "#6366f1"},
		{"Cabos e Adaptadores", "cable", "#8b5cf6"},
		{"Hardware / Componentes", "memory-stick", "#ec4899"},
		{"Ferramentas", "wrench", "#f59e0b"},
		{"Chaves", "key", "#eab308"},
		{"Louças e Cozinha", "utensils", "#10b981"},
		{"Alimentos", "apple", "#22c55e"},
		{"Livros", "book", "#06b6d4"},
		{"Roupas", "shirt", "#f97316"},
		{"Documentos", "file-text", "#64748b"},
		{"Outros", "package", "#94a3b8"},
	}

	for _, c := range categories {
		_, err := db.Exec("INSERT INTO categories (name, icon, color) VALUES (?, ?, ?)", c.name, c.icon, c.color)
		if err != nil {
			return err
		}
	}

	var locCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM locations").Scan(&locCount)
	if locCount == 0 {
		locations := []struct {
			name, typ string
		}{
			{"Sala", "comodo"},
			{"Quarto", "comodo"},
			{"Cozinha", "comodo"},
			{"Escritório", "comodo"},
			{"Garagem", "comodo"},
			{"Bancada", "movel"},
			{"Armário", "movel"},
			{"Caixa Organizadora", "caixa"},
		}
		for _, l := range locations {
			_, _ = db.Exec("INSERT INTO locations (name, type) VALUES (?, ?)", l.name, l.typ)
		}
	}

	return nil
}
