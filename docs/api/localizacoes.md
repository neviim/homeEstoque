# API — Localizações

Base URL: `http://localhost:8080/api`  
Todos os endpoints requerem `Authorization: Bearer <token>`.

## Hierarquia

Localizações formam uma árvore: um cômodo pode conter um armário, que contém uma caixa. O campo `full_path` retorna o caminho legível:

```
Garagem > Armário de Ferramentas > Caixa Vermelha
```

---

## GET /locations

Lista todas as localizações com caminho completo e contagem de itens.

**Exemplo**
```bash
curl -s http://localhost:8080/api/locations \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
[
  {
    "id": 1,
    "name": "Garagem",
    "type": "comodo",
    "full_path": "Garagem",
    "item_count": 0,
    "created_at": "2026-05-01T08:00:00Z"
  },
  {
    "id": 2,
    "name": "Armário de Ferramentas",
    "type": "armario",
    "parent_id": 1,
    "full_path": "Garagem > Armário de Ferramentas",
    "item_count": 8,
    "created_at": "2026-05-01T08:00:00Z"
  },
  {
    "id": 3,
    "name": "Caixa Vermelha",
    "type": "caixa",
    "parent_id": 2,
    "description": "Ferramentas manuais",
    "full_path": "Garagem > Armário de Ferramentas > Caixa Vermelha",
    "item_count": 5,
    "created_at": "2026-05-01T08:00:00Z"
  }
]
```

---

## POST /locations

Cria uma nova localização.

**Body**
```json
{
  "name": "Escritório",
  "type": "comodo",
  "description": "Sala de trabalho"
}
```

**Tipos válidos**: `comodo`, `movel`, `caixa`, `armario`, `outro`

**Exemplo — criar cômodo raiz**
```bash
curl -s -X POST http://localhost:8080/api/locations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Escritório","type":"comodo","description":"Sala de trabalho"}' | jq
```

**Exemplo — criar caixa dentro de um armário**
```bash
curl -s -X POST http://localhost:8080/api/locations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Caixa de Cabos",
    "type": "caixa",
    "parent_id": 4,
    "description": "Cabos HDMI, USB, energia"
  }' | jq
```

**Resposta 201**
```json
{
  "id": 8,
  "name": "Caixa de Cabos",
  "type": "caixa",
  "parent_id": 4,
  "description": "Cabos HDMI, USB, energia",
  "full_path": "Escritório > Estante > Caixa de Cabos",
  "created_at": "2026-05-22T14:30:00Z"
}
```

---

## PUT /locations/{id}

Atualiza uma localização existente.

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/locations/8 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Caixa de Cabos e Adaptadores","type":"caixa","parent_id":4}' | jq
```

---

## DELETE /locations/{id}

Remove uma localização. Itens vinculados têm `location_id` definido como NULL.

```bash
curl -s -X DELETE http://localhost:8080/api/locations/8 \
  -H "Authorization: Bearer $TOKEN"
# 204 No Content
```
