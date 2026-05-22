# API — Movimentações

Base URL: `http://localhost:8080/api`  
Todos os endpoints requerem `Authorization: Bearer <token>`.

Movimentações são criadas automaticamente quando um item é criado com `location_id` ou quando `location_id` muda em um update. O endpoint abaixo lista **todas** as movimentações do sistema.

---

## GET /movements

Lista todas as movimentações com paginação.

**Query params**

| Parâmetro | Tipo | Default | Descrição |
|-----------|------|---------|-----------|
| `page` | int | 1 | Página (1-indexed) |
| `limit` | int | 15 | Registros por página |

**Exemplo**
```bash
curl -s "http://localhost:8080/api/movements?page=1&limit=15" \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "movements": [
    {
      "id": 52,
      "item_id": 47,
      "item_name": "Aspirador de Pó Philips",
      "from_location_id": 2,
      "to_location_id": 6,
      "from_location_name": "Armário de Ferramentas",
      "to_location_name": "Despensa",
      "quantity": 1,
      "reason": "Movimentação via MCP",
      "user_id": 3,
      "user_name": "MCP Assistant",
      "created_at": "2026-05-22T11:30:00Z"
    },
    {
      "id": 51,
      "item_id": 42,
      "item_name": "Furadeira de Impacto",
      "from_location_id": null,
      "to_location_id": 3,
      "from_location_name": "",
      "to_location_name": "Caixa Vermelha",
      "quantity": 1,
      "reason": "Cadastro inicial",
      "user_id": 2,
      "user_name": "Maria Silva",
      "created_at": "2026-05-01T10:00:00Z"
    }
  ],
  "total": 52,
  "page": 1,
  "limit": 15,
  "total_pages": 4
}
```

**Movimentações por item**

Para ver o histórico de um item específico use `GET /items/{id}/movements` (documentado em [itens.md](itens.md)).

---

## Como movimentações são criadas

| Ação | Quem cria | Reason padrão |
|------|-----------|---------------|
| Criar item com `location_id` (API HTTP) | Handler HTTP | `"Cadastro inicial"` |
| Atualizar `location_id` (API HTTP) | Handler HTTP | `"Atualização"` |
| `create_item` com `location_id` (MCP) | MCP | `"Cadastro via MCP"` |
| `update_item` com novo `location_id` (MCP) | MCP | `"Atualização via MCP"` |
| `move_item` (MCP) | MCP | `"Movimentação via MCP"` ou reason customizado |

**Identificar movimentos do MCP**: na coluna "Usuário" aparece "MCP Assistant" para todos os movimentos originados pelo servidor MCP.
