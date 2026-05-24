package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret"

func TestHashPassword_ReturnsBcryptHash(t *testing.T) {
	hash, err := HashPassword("senha123")
	require.NoError(t, err)
	// bcrypt em Go gera prefixo $2a$ (versão padrão do pacote x/crypto/bcrypt)
	assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"),
		"hash deveria começar com $2a$ ou $2b$, got %q", hash[:4])
}

func TestHashPassword_DifferentSaltsForSamePassword(t *testing.T) {
	h1, err := HashPassword("mesma-senha")
	require.NoError(t, err)
	h2, err := HashPassword("mesma-senha")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "salt aleatório deveria produzir hashes diferentes")
}

func TestCheckPassword_TrueForCorrectPassword(t *testing.T) {
	hash, err := HashPassword("senha-correta")
	require.NoError(t, err)
	assert.True(t, CheckPassword("senha-correta", hash))
}

func TestCheckPassword_FalseForWrongPassword(t *testing.T) {
	hash, err := HashPassword("senha-correta")
	require.NoError(t, err)
	assert.False(t, CheckPassword("senha-errada", hash))
}

func TestCheckPassword_FalseForInvalidOrEmptyHash(t *testing.T) {
	assert.False(t, CheckPassword("qualquer", ""))
	assert.False(t, CheckPassword("qualquer", "nao-eh-bcrypt"))
}

func TestGenerateToken_ReturnsValidJWTFormat(t *testing.T) {
	token, err := GenerateToken(42, "x@y.com", testSecret)
	require.NoError(t, err)
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3, "JWT deve ter 3 partes separadas por ponto")
}

func TestParseToken_RecoversClaimsFromValidToken(t *testing.T) {
	token, err := GenerateToken(42, "alice@x.com", testSecret)
	require.NoError(t, err)

	claims, err := ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "alice@x.com", claims.Email)
	assert.Equal(t, "homeestoque", claims.Issuer)
}

func TestParseToken_FailsWithDifferentSecret(t *testing.T) {
	token, err := GenerateToken(1, "x@y.com", testSecret)
	require.NoError(t, err)

	_, err = ParseToken(token, "outro-secret")
	assert.Error(t, err)
}

func TestParseToken_FailsForEmptyOrMalformed(t *testing.T) {
	cases := []string{"", "abc", "nao.eh.jwt", "header.payload"} // < 3 partes
	for _, tc := range cases {
		_, err := ParseToken(tc, testSecret)
		assert.Error(t, err, "token %q deveria falhar", tc)
	}
}

func TestParseToken_FailsForExpiredToken(t *testing.T) {
	// Constrói manualmente um token com exp no passado para testar a expiração
	claims := &Claims{
		UserID: 1,
		Email:  "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "homeestoque",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = ParseToken(signed, testSecret)
	assert.Error(t, err, "token expirado deveria ser rejeitado")
}
