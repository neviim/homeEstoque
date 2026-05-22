# API — Itens

Base URL: `http://localhost:8080/api`  
Todos os endpoints (exceto `/items/{id}/qrcode`) requerem `Authorization: Bearer <token>`.

---

## GET /items

Lista itens com filtros opcionais e paginação server-side.

**Query params**

| Parâmetro | Tipo | Default | Descrição |
|-----------|------|---------|-----------|
| `search` | string | — | Busca em nome, código, marca, modelo, descrição |
| `category_id` | int | — | Filtra por categoria |
| `location_id` | int | — | Filtra por localização |
| `page` | int | 1 | Página (1-indexed) |
| `limit` | int | 12 | Itens por página |

**Exemplo — listar todos**
```bash
curl -s "http://localhost:8080/api/items" \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Exemplo — buscar "furadeira"**
```bash
curl -s "http://localhost:8080/api/items?search=furadeira" \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Exemplo — listar eletrônicos, página 2**
```bash
curl -s "http://localhost:8080/api/items?category_id=1&page=2&limit=12" \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "items": [
    {
      "id": 42,
      "code": "EST-A3F7B219",
      "name": "Furadeira de Impacto",
      "description": "Furadeira Bosch 650W com maleta",
      "brand": "Bosch",
      "model": "GSB 650 RE",
      "serial_number": "BSH2024001",
      "quantity": 1,
      "unit": "un",
      "condition": "bom",
      "purchase_date": "2023-03-15",
      "purchase_price": 289.90,
      "notes": "",
      "category_id": 2,
      "category_name": "Ferramentas",
      "location_id": 3,
      "location_path": "Garagem > Armário de Ferramentas > Caixa Vermelha",
      "created_at": "2026-05-01T10:00:00Z",
      "updated_at": "2026-05-20T15:30:00Z"
    }
  ],
  "total": 47,
  "page": 1,
  "limit": 12,
  "total_pages": 4
}
```

---

## POST /items

Cria um novo item. O código `EST-XXXXXXXX` é gerado automaticamente.

**Body completo**
```json
{
  "name": "Notebook Dell Inspiron",
  "description": "Notebook para trabalho remoto",
  "brand": "Dell",
  "model": "Inspiron 15 3511",
  "serial_number": "DELL2024-XK91",
  "quantity": 1,
  "unit": "un",
  "purchase_date": "2024-01-10",
  "purchase_price": 3299.00,
  "condition": "bom",
  "notes": "Carregador na gaveta do escritório",
  "category_id": 1,
  "location_id": 5
}
```

Campos obrigatórios: `name`  
Defaults: `quantity=1`, `unit="un"`, `condition="novo"`

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Notebook Dell Inspiron",
    "brand": "Dell",
    "model": "Inspiron 15 3511",
    "serial_number": "DELL2024-XK91",
    "quantity": 1,
    "purchase_price": 3299.00,
    "condition": "bom",
    "category_id": 1,
    "location_id": 5
  }' | jq
```

**Resposta 201**
```json
{
  "id": 55,
  "code": "EST-C8D2E1F4",
  "name": "Notebook Dell Inspiron",
  "brand": "Dell",
  "model": "Inspiron 15 3511",
  "serial_number": "DELL2024-XK91",
  "quantity": 1,
  "unit": "un",
  "condition": "bom",
  "purchase_price": 3299.00,
  "category_id": 1,
  "category_name": "Eletrônicos",
  "location_id": 5,
  "location_path": "Escritório > Mesa",
  "photos": [],
  "created_at": "2026-05-22T14:45:00Z",
  "updated_at": "2026-05-22T14:45:00Z"
}
```

---

## GET /items/{id}

Retorna um item específico com todos os campos, fotos e localização completa.

**Exemplo**
```bash
curl -s http://localhost:8080/api/items/42 \
  -H "Authorization: Bearer $TOKEN" | jq
```

---

## PUT /items/{id}

Atualiza um item (atualização completa — envie todos os campos).  
Se `location_id` mudar, um registro de movimentação é criado automaticamente.

**Exemplo — mover item para outra localização**
```bash
curl -s -X PUT http://localhost:8080/api/items/42 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Furadeira de Impacto",
    "brand": "Bosch",
    "model": "GSB 650 RE",
    "quantity": 1,
    "unit": "un",
    "condition": "bom",
    "category_id": 2,
    "location_id": 7
  }' | jq
```

---

## DELETE /items/{id}

Remove o item e todas as suas fotos e movimentações (CASCADE).

```bash
curl -s -X DELETE http://localhost:8080/api/items/55 \
  -H "Authorization: Bearer $TOKEN"
# 204 No Content
```

---

## GET /items/{id}/movements

Lista o histórico de movimentações de um item.

**Exemplo**
```bash
curl -s http://localhost:8080/api/items/42/movements \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
[
  {
    "id": 15,
    "item_id": 42,
    "from_location_id": null,
    "to_location_id": 3,
    "from_location_name": "",
    "to_location_name": "Caixa Vermelha",
    "quantity": 1,
    "reason": "Cadastro inicial",
    "user_id": 2,
    "user_name": "Maria Silva",
    "created_at": "2026-05-01T10:00:00Z"
  },
  {
    "id": 28,
    "item_id": 42,
    "from_location_id": 3,
    "to_location_id": 7,
    "from_location_name": "Caixa Vermelha",
    "to_location_name": "Bancada",
    "quantity": 1,
    "reason": "Usando para reforma",
    "user_id": 3,
    "user_name": "MCP Assistant",
    "created_at": "2026-05-22T11:00:00Z"
  }
]
```

---

## POST /items/{id}/photos

Faz upload de uma foto para o item. Aceita `multipart/form-data`.

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/items/42/photos \
  -H "Authorization: Bearer $TOKEN" \
  -F "photo=@/caminho/para/foto.jpg" | jq
```

**Resposta 201**
```json
{
  "id": 3,
  "item_id": 42,
  "filename": "1716393600_foto.jpg",
  "original_name": "foto.jpg",
  "size": 245678,
  "url": "/uploads/1716393600_foto.jpg",
  "created_at": "2026-05-22T15:00:00Z"
}
```

---

## DELETE /items/{id}/photos/{photoId}

Remove uma foto do item.

```bash
curl -s -X DELETE http://localhost:8080/api/items/42/photos/3 \
  -H "Authorization: Bearer $TOKEN"
# 204 No Content
```

---

## GET /items/{id}/qrcode

Gera e retorna a imagem QR Code do item. **Endpoint público** — não requer autenticação.

O QR Code codifica a URL `http://localhost:8080/api/items/{id}`.

```bash
# Salvar QR Code como PNG
curl -s http://localhost:8080/api/items/42/qrcode -o qrcode_item42.png

# Abrir no navegador
xdg-open http://localhost:8080/api/items/42/qrcode
```

Na UI, o ícone QR Code na lista de itens abre um modal com a imagem e botão para download.
