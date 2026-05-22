package database

import "database/sql"

// MCPUserEmail é o email do usuário sintético usado pelo MCP server
// para registrar autoria em movements. Login HTTP nunca será feito por ele.
const MCPUserEmail = "mcp@homeestoque.local"

func Seed(db *sql.DB) error {
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
