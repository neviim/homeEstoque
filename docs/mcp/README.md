# MCP — Servidor HomeEstoque

O servidor MCP expõe o inventário HomeEstoque para clientes como Claude Code e Claude Desktop através do Model Context Protocol (stdio transport).

## Documentos

| Documento | Conteúdo |
|-----------|----------|
| [servidor.md](servidor.md) | Compilação, configuração e smoke test |
| [ferramentas.md](ferramentas.md) | Referência das 10 tools com exemplos |
| [exemplos-claude-code.md](exemplos-claude-code.md) | 12 exemplos reais de uso com Claude Code |

## Ferramentas disponíveis

| Tool | Tipo | Descrição |
|------|------|-----------|
| `find_item_location` | Consulta | "onde está X?" — busca fuzzy, retorna location_path |
| `list_items` | Consulta | Lista com filtros e paginação |
| `get_item` | Consulta | Detalhe por id ou code |
| `list_categories` | Consulta | Todas as categorias |
| `list_locations` | Consulta | Todas as localizações com full_path |
| `create_item` | Criação | Cria item com SKU automático |
| `create_category` | Criação | Nova categoria (suporta hierarquia) |
| `create_location` | Criação | Nova localização (suporta hierarquia) |
| `update_item` | Atualização | Atualização parcial; registra movement se mudar local |
| `move_item` | Movimentação | Atalho para mover com reason customizável |

> **Sem delete** — o servidor MCP não expõe operação de remoção de itens. Essa decisão reduz o risco de o Claude apagar dados acidentalmente.

## Início rápido

```bash
# 1. Compilar
./tools/build-mcp.sh

# 2. Registrar no Claude Code
claude mcp add homeestoque \
  --scope local \
  -e DB_PATH=/home/neviim/developer/homeEstoque/backend/data/homeestoque.db \
  -- /home/neviim/developer/homeEstoque/bin/homeestoque-mcp

# 3. Verificar (dentro de uma sessão Claude Code)
# /mcp  →  deve listar "homeestoque" com 10 tools
```
