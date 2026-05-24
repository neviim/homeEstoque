package permissions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalog_HasExpectedSize(t *testing.T) {
	// Hoje são 20 capacidades (15 originais + 4 do módulo de backup + 1 system.update).
	// Quando adicionar nova, atualizar este número é parte do contrato —
	// sinaliza que precisa documentar/seedar.
	assert.Len(t, Catalog, 20)
}

func TestCatalog_AllKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Catalog {
		assert.False(t, seen[p.Key], "key duplicada no catálogo: %s", p.Key)
		seen[p.Key] = true
	}
}

func TestCatalog_AllEntriesHaveRequiredFields(t *testing.T) {
	for _, p := range Catalog {
		assert.NotEmpty(t, p.Key, "key vazia")
		assert.NotEmpty(t, p.Label, "label vazio em %s", p.Key)
		assert.NotEmpty(t, p.Description, "description vazio em %s", p.Key)
		assert.NotEmpty(t, p.Category, "category vazio em %s", p.Key)
	}
}

func TestKeys_ReturnsAllCatalogKeys(t *testing.T) {
	keys := Keys()
	assert.Len(t, keys, len(Catalog))
	for i, p := range Catalog {
		assert.Equal(t, p.Key, keys[i])
	}
}

func TestExists(t *testing.T) {
	assert.True(t, Exists("items.view"))
	assert.True(t, Exists("roles.manage"))
	assert.False(t, Exists("inexistente.key"))
	assert.False(t, Exists(""))
	assert.False(t, Exists("ITEMS.VIEW"), "comparação deve ser case-sensitive")
}
