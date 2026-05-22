package permissions

// Permission descreve uma capacidade discreta do sistema. As keys são strings
// estáveis usadas em middleware e no frontend; label/description aparecem na UI.
type Permission struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Catalog é a fonte da verdade para todas as permissões do sistema.
// Adicionar uma entrada aqui torna automaticamente disponível tanto para o
// backend (RequirePermission) quanto para o frontend (UI de roles).
var Catalog = []Permission{
	{"dashboard.view", "Ver Dashboard", "Acessar a página inicial do sistema", "Visualização"},
	{"dashboard.view_value", "Ver valor patrimonial", "Exibir o card 'Valor patrimonial estimado'", "Visualização"},

	{"items.view", "Ver Itens", "Listar e consultar itens do estoque", "Itens"},
	{"items.create", "Criar Itens", "Cadastrar novos itens", "Itens"},
	{"items.update", "Editar Itens", "Atualizar dados de itens existentes", "Itens"},
	{"items.delete", "Excluir Itens", "Remover itens do estoque", "Itens"},
	{"items.upload_photo", "Anexar Fotos", "Upload e remoção de fotos dos itens", "Itens"},

	{"categories.view", "Ver Categorias", "Listar categorias", "Categorias"},
	{"categories.manage", "Gerenciar Categorias", "Criar, editar e excluir categorias", "Categorias"},

	{"locations.view", "Ver Locais", "Listar locais", "Locais"},
	{"locations.manage", "Gerenciar Locais", "Criar, editar e excluir locais", "Locais"},

	{"movements.view", "Ver Movimentações", "Histórico de movimentações", "Movimentações"},

	{"export.csv", "Exportar CSV", "Baixar o estoque em CSV", "Exportação"},

	{"users.manage", "Gerenciar Usuários", "Criar, editar e excluir usuários", "Sistema"},
	{"roles.manage", "Gerenciar Perfis", "Configurar perfis e suas permissões", "Sistema"},
}

// Keys retorna todas as keys do catálogo (útil para o seed do admin).
func Keys() []string {
	out := make([]string, len(Catalog))
	for i, p := range Catalog {
		out[i] = p.Key
	}
	return out
}

// Exists indica se uma key pertence ao catálogo.
func Exists(key string) bool {
	for _, p := range Catalog {
		if p.Key == key {
			return true
		}
	}
	return false
}
