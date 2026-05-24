package mcptools_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neviim/homeestoque/backend/internal/mcptools"
	"github.com/neviim/homeestoque/backend/internal/testutil"
)

func TestMCP_ListCategories_ReturnsObjectNotArray(t *testing.T) {
	// Regressão importante: bug anterior retornava array bruto. O MCP SDK exige
	// objeto JSON no structured content.
	tools := newTools(t)
	_, raw, err := tools.ListCategories(context.Background(), nil, mcptools.ListCategoriesArgs{})
	require.NoError(t, err)

	out, ok := raw.(mcptools.ListCategoriesResult)
	require.True(t, ok, "deve ser ListCategoriesResult (objeto), não slice")
	assert.NotNil(t, out.Categories)
	assert.NotEmpty(t, out.Categories, "seed default já popula categorias")
}

func TestMCP_ListCategories_IncludesItemCount(t *testing.T) {
	tools := newTools(t)
	catID := testutil.CreateCategory(t, tools.DB, "PopulatedTest")
	testutil.CreateItem(t, tools.DB, "A", testutil.ItemOpts{CategoryID: &catID})
	testutil.CreateItem(t, tools.DB, "B", testutil.ItemOpts{CategoryID: &catID})

	_, raw, _ := tools.ListCategories(context.Background(), nil, mcptools.ListCategoriesArgs{})
	out := raw.(mcptools.ListCategoriesResult)

	var found bool
	for _, c := range out.Categories {
		if c.Name == "PopulatedTest" {
			assert.Equal(t, 2, c.ItemCount)
			found = true
		}
	}
	assert.True(t, found)
}

func TestMCP_CreateCategory_EmptyName_ReturnsError(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.CreateCategory(context.Background(), nil, mcptools.CreateCategoryArgs{Name: ""})
	require.Error(t, err)
}

func TestMCP_CreateCategory_WithOptionalFields(t *testing.T) {
	tools := newTools(t)
	_, _, err := tools.CreateCategory(context.Background(), nil, mcptools.CreateCategoryArgs{
		Name:  "Nova Cat",
		Icon:  "cpu",
		Color: "#ff0000",
	})
	require.NoError(t, err)

	// Verifica via DB que o INSERT teve os campos corretos
	var name, icon, color string
	require.NoError(t, tools.DB.QueryRow(
		`SELECT name, icon, color FROM categories WHERE name = ?`, "Nova Cat",
	).Scan(&name, &icon, &color))
	assert.Equal(t, "Nova Cat", name)
	assert.Equal(t, "cpu", icon)
	assert.Equal(t, "#ff0000", color)
}
