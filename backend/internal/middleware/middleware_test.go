package middleware_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/auth"
	"github.com/neviim/homeestoque/backend/internal/middleware"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

// fakeHandler captura o user_id do contexto e devolve no body — usado para
// confirmar que o Auth middleware propaga o id corretamente.
func fakeHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		uid := middleware.GetUserID(r)
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": uid})
	}
}

func TestAuth_NoHeader_Returns401(t *testing.T) {
	h := middleware.Auth(testutil.TestJWTSecret)(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_MalformedHeader_Returns401(t *testing.T) {
	cases := []string{
		"NotBearer xxx",
		"Bearer",         // sem token
		"xxx.yyy.zzz",    // sem Bearer
		"basic dXNlcjpw", // outro scheme
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			h := middleware.Auth(testutil.TestJWTSecret)(fakeHandler(t))
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", raw)
			h.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	h := middleware.Auth(testutil.TestJWTSecret)(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalido.nao.eh.jwt")
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_ExpiredToken_Returns401(t *testing.T) {
	// Token expirado manualmente
	claims := &auth.Claims{
		UserID: 1, Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(testutil.TestJWTSecret))
	require.NoError(t, err)

	h := middleware.Auth(testutil.TestJWTSecret)(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_ValidToken_PutsUserIDInContext(t *testing.T) {
	tok := testutil.TokenFor(t, 42, "x@y.com")

	h := middleware.Auth(testutil.TestJWTSecret)(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(42), body["uid"])
}

func TestRequirePermission_NoAuth_Returns401(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	h := middleware.RequirePermission(db, "items.view")(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequirePermission_AdminHasAccess(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	admin := testutil.CreateUser(t, db, "Admin", "admin@x.com", "senha123", "admin")

	h := middleware.RequirePermission(db, "users.manage")(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := withUser(httptest.NewRequest("GET", "/", nil), admin.ID, admin.Email)
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequirePermission_ViewerBlocked(t *testing.T) {
	db := testutil.NewSeededTestDB(t)
	viewer := testutil.CreateUser(t, db, "Viewer", "viewer@x.com", "senha123", "viewer")

	h := middleware.RequirePermission(db, "users.manage")(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := withUser(httptest.NewRequest("GET", "/", nil), viewer.ID, viewer.Email)
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequirePermission_ReturnsJSONErrorOn403(t *testing.T) {
	// O frontend depende do body ser JSON com campo "error" para o toast.
	db := testutil.NewSeededTestDB(t)
	viewer := testutil.CreateUser(t, db, "Viewer", "v@x.com", "senha123", "viewer")

	h := middleware.RequirePermission(db, "users.manage")(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := withUser(httptest.NewRequest("GET", "/", nil), viewer.ID, viewer.Email)
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
	body := strings.TrimSpace(rr.Body.String())
	assert.Contains(t, body, `"error"`)
	assert.Contains(t, body, "acesso negado")
}

func TestRequirePermission_DBError_Returns500(t *testing.T) {
	// Fecha o DB para simular erro — HasPermission devolve err, middleware
	// deve responder 500 (não 200 nem 401).
	db := testutil.NewSeededTestDB(t)
	viewer := testutil.CreateUser(t, db, "V", "v@x.com", "senha123", "viewer")
	_ = db.Close() // força erro nas próximas queries

	h := middleware.RequirePermission(db, "items.view")(fakeHandler(t))
	rr := httptest.NewRecorder()
	req := withUser(httptest.NewRequest("GET", "/", nil), viewer.ID, viewer.Email)
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// withUser injeta o user_id no contexto da request — equivalente ao que o
// middleware.Auth faria, mas sem precisar fazer JWT round-trip.
func withUser(r *http.Request, uid int64, email string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, uid)
	ctx = context.WithValue(ctx, middleware.UserEmailKey, email)
	return r.WithContext(ctx)
}

// Garante que a importação do sql não fica órfã se alguém remexer nos testes
// (necessário só para o compilador, sem comportamento em runtime).
var _ = (*sql.DB)(nil)
var _ = fmt.Sprintf
