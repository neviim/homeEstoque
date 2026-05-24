package mcptools_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/mcptools"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestMCP_ListLocations_ReturnsObjectNotArray(t *testing.T) {
	// Mesma regressão: bug anterior retornava array bruto. Garantir objeto.
	tools := newTools(t)
	_, raw, err := tools.ListLocations(context.Background(), nil, mcptools.ListLocationsArgs{})
	require.NoError(t, err)
	out, ok := raw.(mcptools.ListLocationsResult)
	require.True(t, ok, "deve ser ListLocationsResult (objeto)")
	assert.NotNil(t, out.Locations)
}

func TestMCP_ListLocations_ResolvesFullPath(t *testing.T) {
	tools := newTools(t)
	garagem := testutil.CreateLocation(t, tools.DB, "TestGaragem", "comodo", nil)
	bancada := testutil.CreateLocation(t, tools.DB, "TestBancada", "movel", &garagem)

	_, raw, err := tools.ListLocations(context.Background(), nil, mcptools.ListLocationsArgs{})
	require.NoError(t, err)
	out := raw.(mcptools.ListLocationsResult)

	var found bool
	for _, l := range out.Locations {
		if l.ID == bancada {
			assert.Equal(t, "TestGaragem > TestBancada", l.FullPath)
			found = true
		}
	}
	assert.True(t, found)
}

func TestMCP_CreateLocation_InvalidType_ReturnsError(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.CreateLocation(context.Background(), nil, mcptools.CreateLocationArgs{
		Name: "X",
		Type: "tipo-invalido",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type inválido")
}

func TestMCP_CreateLocation_DefaultsToOutro(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.CreateLocation(context.Background(), nil, mcptools.CreateLocationArgs{
		Name: "Sem Tipo",
	})
	require.NoError(t, err)

	var typ string
	require.NoError(t, tools.DB.QueryRow(`SELECT type FROM locations WHERE name = ?`, "Sem Tipo").Scan(&typ))
	assert.Equal(t, "outro", typ)
}

func TestMCP_CreateLocation_InvalidParent_ReturnsError(t *testing.T) {
	tools := newTools(t)
	badParent := int64(99999)

	_, _, err := tools.CreateLocation(context.Background(), nil, mcptools.CreateLocationArgs{
		Name:     "X",
		Type:     "comodo",
		ParentID: &badParent,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent_id")
}
