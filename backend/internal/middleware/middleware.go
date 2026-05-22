package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/permissions"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"
const UserEmailKey ctxKey = "user_email"

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}
			claims, err := auth.ParseToken(parts[1], secret)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) int64 {
	if v, ok := r.Context().Value(UserIDKey).(int64); ok {
		return v
	}
	return 0
}

// RequirePermission verifica se o usuário autenticado tem a permission `key`.
// Retorna 401 se não autenticado, 403 se autenticado mas sem permissão.
// A consulta vai sempre ao banco — mudanças no role passam a valer no próximo request.
func RequirePermission(db *sql.DB, key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid := GetUserID(r)
			if uid == 0 {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"sem autenticação"}`, http.StatusUnauthorized)
				return
			}
			ok, err := permissions.HasPermission(db, uid, key)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"erro ao verificar permissão"}`, http.StatusInternalServerError)
				return
			}
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
