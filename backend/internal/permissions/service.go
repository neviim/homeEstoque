package permissions

import (
	"database/sql"
)

// RolePermissions lista as permissions de um role pelo nome.
// Retorna slice vazio (nunca nil) se o role não existir ou não tiver permissions.
func RolePermissions(db *sql.DB, roleName string) ([]string, error) {
	rows, err := db.Query(
		`SELECT rp.permission
		 FROM role_permissions rp
		 JOIN roles r ON r.id = rp.role_id
		 WHERE r.name = ?
		 ORDER BY rp.permission`,
		roleName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// UserPermissions retorna as permissions efetivas de um usuário (via seu role).
func UserPermissions(db *sql.DB, userID int64) ([]string, error) {
	rows, err := db.Query(
		`SELECT rp.permission
		 FROM users u
		 JOIN roles r ON r.name = u.role
		 JOIN role_permissions rp ON rp.role_id = r.id
		 WHERE u.id = ?
		 ORDER BY rp.permission`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// HasPermission é o check usado pelo middleware: true se o usuário detém a key.
func HasPermission(db *sql.DB, userID int64, key string) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*)
		 FROM users u
		 JOIN roles r ON r.name = u.role
		 JOIN role_permissions rp ON rp.role_id = r.id
		 WHERE u.id = ? AND rp.permission = ?`,
		userID, key,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
