# API — Categorias

Base URL: `http://localhost:8080/api`  
Todos os endpoints requerem `Authorization: Bearer <token>`.

---

## GET /categories

Lista todas as categorias com contagem de itens.

**Exemplo**
```bash
curl -s http://localhost:8080/api/categories \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
[
  {
    "id": 1,
    "name": "Eletrônicos",
    "icon": "cpu",
    "color": "#3b82f6",
    "item_count": 12,
    "created_at": "2026-05-01T08:00:00Z"
  },
  {
    "id": 2,
    "name": "Ferramentas",
    "icon": "wrench",
    "color": "#f59e0b",
    "item_count": 8,
    "created_at": "2026-05-01T08:00:00Z"
  },
  {
    "id": 3,
    "name": "Componentes",
    "parent_id": 1,
    "icon": "cpu",
    "color": "#6366f1",
    "item_count": 5,
    "created_at": "2026-05-10T10:00:00Z"
  }
]
```

---

## POST /categories

Cria uma nova categoria.

**Body**
```json
{
  "name": "Cabos e Adaptadores",
  "icon": "plug",
  "color": "#10b981",
  "parent_id": 1
}
```

Campos obrigatórios: `name`  
Campos opcionais: `icon` (nome de ícone lucide-react), `color` (hex), `parent_id`

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Cabos e Adaptadores",
    "icon": "plug",
    "color": "#10b981",
    "parent_id": 1
  }' | jq
```

**Resposta 201**
```json
{
  "id": 9,
  "name": "Cabos e Adaptadores",
  "parent_id": 1,
  "icon": "plug",
  "color": "#10b981",
  "created_at": "2026-05-22T14:00:00Z"
}
```

---

## PUT /categories/{id}

Atualiza uma categoria existente (atualização completa — envie todos os campos).

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/categories/9 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Cabos e Adaptadores HDMI",
    "icon": "monitor",
    "color": "#10b981"
  }' | jq
```

---

## DELETE /categories/{id}

Remove uma categoria. Itens vinculados têm `category_id` definido como NULL.

**Exemplo**
```bash
curl -s -X DELETE http://localhost:8080/api/categories/9 \
  -H "Authorization: Bearer $TOKEN"
# 204 No Content
```
