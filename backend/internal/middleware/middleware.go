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

// writeJSONError envia um body JSON com campo "error" e o status code dado.
// Não usa http.Error pois ela força Content-Type: text/plain — o frontend
// depende de JSON para extrair a mensagem via `err.response.data.error`.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}` + "\n"))
}

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeJSONError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			claims, err := auth.ParseToken(parts[1], secret)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
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
				writeJSONError(w, http.StatusUnauthorized, "sem autenticação")
				return
			}
			ok, err := permissions.HasPermission(db, uid, key)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "erro ao verificar permissão")
				return
			}
			if !ok {
				writeJSONError(w, http.StatusForbidden, "acesso negado")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
