package middleware

import (
	"net/http"
	"strings"
)

// MaintenanceGate consulta `isMaintenance` em cada request; quando ativo,
// devolve 503 para todas as rotas exceto:
//   - /health
//   - rotas em allowPrefixes (ex.: /api/backups/.../restore — pra permitir o
//     próprio fluxo de restore enquanto o sistema está em manutenção)
func MaintenanceGate(isMaintenance func() bool, allowPrefixes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMaintenance() {
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			for _, p := range allowPrefixes {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSONError(w, http.StatusServiceUnavailable, "sistema em manutenção (restore em andamento)")
		})
	}
}
