# MCP — Ferramentas (Tools)

O servidor expõe 10 ferramentas agrupadas em três categorias.

---

## Consulta

### `find_item_location` ⭐

A ferramenta central. Responde à pergunta "onde está X?" com busca fuzzy em nome, código, marca, modelo e descrição.

**Input**
```json
{ "query": "furadeira" }
```

**Exemplo de uso no Claude**
> "onde está minha furadeira?"

**Output**
```json
{
  "query": "furadeira",
  "matches": [
    {
      "name": "Furadeira de Impacto",
      "code": "EST-A3F7B219",
      "quantity": 1,
      "unit": "un",
      "location_path": "Garagem > Armário de Ferramentas > Caixa Vermelha",
      "category_name": "Ferramentas"
    }
  ],
  "total": 1
}
```

Retorna até 20 matches, ordenados por relevância (nome primeiro, depois código, depois outros campos).

---

### `list_items`

Lista itens com filtros opcionais e paginação.

**Input**
```json
{
  "search": "samsung",
  "category_id": 1,
  "location_id": null,
  "page": 1,
  "limit": 20
}
```

Todos os campos são opcionais. `limit` máximo: 50.

**Exemplo de uso no Claude**
> "liste todos os eletrônicos"  
> "quais itens estão na garagem?"  
> "mostre todos os itens da Samsung"

**Output**
```json
{
  "items": [
    {
      "id": 12,
      "code": "EST-B5C3A7D2",
      "name": "Smart TV Samsung 55\"",
      "brand": "Samsung",
      "model": "UN55TU8000",
      "quantity": 1,
      "unit": "un",
      "condition": "bom",
      "category_name": "Eletrônicos",
      "location_path": "Sala > Rack TV"
    }
  ],
  "total": 3,
  "page": 1,
  "limit": 20,
  "total_pages": 1
}
```

---

### `get_item`

Busca um item específico pelo `id` ou `code`.

**Input (por id)**
```json
{ "id": 42 }
```

**Input (por código)**
```json
{ "code": "EST-A3F7B219" }
```

**Exemplo de uso no Claude**
> "me dê os detalhes do item EST-A3F7B219"  
> "qual é o número de série do item 42?"

**Output**
```json
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
  "category_id": 2,
  "category_name": "Ferramentas",
  "location_id": 3,
  "location_path": "Garagem > Armário de Ferramentas > Caixa Vermelha"
}
```

---

### `list_categories`

Lista todas as categorias com contagem de itens.

**Input**: sem parâmetros

**Exemplo de uso no Claude**
> "quais categorias existem?"

**Output**
```json
[
  { "id": 1, "name": "Eletrônicos", "icon": "cpu", "color": "#3b82f6", "item_count": 15 },
  { "id": 2, "name": "Ferramentas", "icon": "wrench", "color": "#f59e0b", "item_count": 8 },
  { "id": 3, "name": "Eletrodomésticos", "icon": "home", "color": "#10b981", "item_count": 6 }
]
```

---

### `list_locations`

Lista todas as localizações com caminho completo e contagem de itens.

**Input**: sem parâmetros

**Exemplo de uso no Claude**
> "quais localizações estão cadastradas?"  
> "onde posso guardar um item novo?"

**Output**
```json
[
  { "id": 1, "name": "Garagem", "type": "comodo", "full_path": "Garagem", "item_count": 0 },
  { "id": 2, "name": "Armário de Ferramentas", "type": "armario", "parent_id": 1, "full_path": "Garagem > Armário de Ferramentas", "item_count": 8 },
  { "id": 3, "name": "Caixa Vermelha", "type": "caixa", "parent_id": 2, "full_path": "Garagem > Armário de Ferramentas > Caixa Vermelha", "item_count": 5 }
]
```

---

## Criação

### `create_item`

Cria um novo item no inventário. Gera código SKU automaticamente.

**Input**
```json
{
  "name": "Aspirador Robô iRobot",
  "description": "Roomba 676 com dock de carregamento",
  "brand": "iRobot",
  "model": "Roomba 676",
  "quantity": 1,
  "unit": "un",
  "purchase_price": 1499.00,
  "condition": "novo",
  "category_id": 3,
  "location_id": 6
}
```

Obrigatório: `name`. Se `location_id` for informado, registra movement inicial com reason `"Cadastro via MCP"`.

**Exemplo de uso no Claude**
> "cadastre um aspirador robô iRobot Roomba na despensa, preço R$ 1.499"

**Output**: item completo com código e `location_path`.

---

### `create_category`

Cria uma nova categoria, opcionalmente hierárquica.

**Input**
```json
{
  "name": "Robôs Domésticos",
  "icon": "bot",
  "color": "#8b5cf6",
  "parent_id": 3
}
```

**Exemplo de uso no Claude**
> "crie uma categoria 'Robôs Domésticos' dentro de Eletrodomésticos"

---

### `create_location`

Cria uma nova localização com hierarquia opcional.

**Input**
```json
{
  "name": "Caixa de Ferramentas Elétricas",
  "type": "caixa",
  "parent_id": 2,
  "description": "Ferramentas elétricas e suas baterias"
}
```

Tipos válidos: `comodo`, `movel`, `caixa`, `armario`, `outro`

**Exemplo de uso no Claude**
> "crie uma caixa chamada 'Caixa de Ferramentas Elétricas' dentro do armário da garagem (id 2)"

---

## Atualização / Movimentação

### `update_item`

Atualização **parcial** — apenas os campos enviados são alterados. Registra movement automaticamente se `location_id` mudar.

**Input**
```json
{
  "id": 42,
  "condition": "regular",
  "notes": "Broca de 8mm precisa de troca"
}
```

**Exemplo de uso no Claude**
> "atualize a condição da furadeira (id 42) para 'regular' e adicione uma nota sobre a broca"  
> "adicione o número de série DELL2024-XK91 ao notebook (id 55)"

**Output**: item atualizado completo.

---

### `move_item`

Move um item para outra localização. Atalho dedicado que registra o movimento com reason customizável.

**Input**
```json
{
  "item_id": 42,
  "to_location_id": 8,
  "reason": "Emprestada para o vizinho João"
}
```

`quantity` é opcional (default: quantidade atual do item).  
`reason` é opcional (default: `"Movimentação via MCP"`).

**Exemplo de uso no Claude**
> "mova a furadeira para o escritório"  
> "transfira o aspirador para o quarto principal com o motivo 'limpeza semanal'"

**Output**: item com nova `location_path`.

---

## Sessão exemplo completa no Claude

```
Usuário: onde está minha furadeira?
Claude: [chama find_item_location {"query": "furadeira"}]
        A Furadeira de Impacto Bosch GSB 650 RE está em:
        Garagem > Armário de Ferramentas > Caixa Vermelha

Usuário: mova ela para a bancada (id 7)
Claude: [chama move_item {"item_id": 42, "to_location_id": 7}]
        Feito! A Furadeira de Impacto foi movida para Garagem > Bancada.
        Um registro de movimentação foi criado.

Usuário: cadastre um novo item: martelo Stanley, Ferramentas, Caixa Vermelha
Claude: [chama list_categories para confirmar id de Ferramentas]
        [chama list_locations para confirmar id de Caixa Vermelha]
        [chama create_item {...}]
        Item criado! Código: EST-D9E1F3A2
        Martelo Stanley > Ferramentas > Garagem > Armário de Ferramentas > Caixa Vermelha
```
