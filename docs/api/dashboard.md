# API — Dashboard e Exportação

Base URL: `http://localhost:8080/api`  
Todos os endpoints requerem `Authorization: Bearer <token>`.

---

## GET /dashboard

Retorna estatísticas gerais do inventário e dados para os gráficos da home.

**Exemplo**
```bash
curl -s http://localhost:8080/api/dashboard \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "total_items": 47,
  "total_quantity": 93,
  "total_categories": 8,
  "total_locations": 10,
  "total_value": 18750.30,
  "recent_items": [
    {
      "id": 55,
      "code": "EST-C8D2E1F4",
      "name": "Notebook Dell Inspiron",
      "condition": "bom",
      "category_name": "Eletrônicos",
      "location_path": "Escritório > Mesa",
      "created_at": "2026-05-22T14:45:00Z"
    }
  ],
  "top_categories": [
    { "id": 1, "name": "Eletrônicos", "item_count": 15 },
    { "id": 2, "name": "Ferramentas", "item_count": 12 },
    { "id": 4, "name": "Eletrodomésticos", "item_count": 8 }
  ]
}
```

---

## GET /export/csv

Exporta todos os itens em formato CSV com todos os campos.

**Exemplo**
```bash
# Baixar CSV
curl -s http://localhost:8080/api/export/csv \
  -H "Authorization: Bearer $TOKEN" \
  -o inventario.csv

# Ver primeiras linhas
curl -s http://localhost:8080/api/export/csv \
  -H "Authorization: Bearer $TOKEN" | head -5
```

**Formato do CSV**
```
id,code,name,description,brand,model,serial_number,quantity,unit,condition,purchase_date,purchase_price,notes,category,location
42,EST-A3F7B219,Furadeira de Impacto,Furadeira Bosch 650W,Bosch,GSB 650 RE,BSH2024001,1,un,bom,2023-03-15,289.90,,Ferramentas,Garagem > Armário de Ferramentas > Caixa Vermelha
55,EST-C8D2E1F4,Notebook Dell Inspiron,,Dell,Inspiron 15 3511,DELL2024-XK91,1,un,bom,2024-01-10,3299.00,,Eletrônicos,Escritório > Mesa
```

O header `Content-Disposition: attachment; filename="inventario.csv"` instrui o browser a fazer download automático ao acessar pelo frontend.
