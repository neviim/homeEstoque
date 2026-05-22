# MCP — Exemplos reais com Claude Code

Todos os exemplos assumem que o servidor `homeestoque` está configurado e ativo (`/mcp` mostra "homeestoque • 10 tools").

---

## 1. Encontrar onde um item está

**Você digita:**
> onde está minha furadeira?

**Claude chama:** `find_item_location {"query": "furadeira"}`

**Claude responde:**
> A **Furadeira de Impacto Bosch GSB 650 RE** (EST-A3F7B219) está em:
> **Garagem › Armário de Ferramentas › Caixa Vermelha**
> Quantidade: 1 un — Condição: bom

---

## 2. Busca por marca

**Você digita:**
> quais itens da Samsung tenho cadastrados?

**Claude chama:** `list_items {"search": "samsung"}`

**Claude responde:**
> Encontrei 3 itens Samsung:
> 1. **Smart TV Samsung 55"** (EST-B5C3A7D2) — Sala › Rack TV — bom
> 2. **Samsung Galaxy S21** (EST-F2A8C3E1) — Quarto Principal › Gaveta da Cômoda — bom
> 3. **Monitor Samsung 27"** (EST-D4B7F9C2) — Escritório › Mesa — novo

---

## 3. Cadastrar um item novo de forma rápida

**Você digita:**
> cadastre um carregador portátil Anker PowerCore 20000 mAh, R$ 189,90, que está na gaveta do escritório. Categoria: Eletrônicos

**Claude chama (em sequência):**
1. `list_categories` — descobre que "Eletrônicos" é id 1
2. `list_locations` — descobre que "Escritório › Gaveta" é id 9
3. `create_item {"name":"Carregador Portátil Anker PowerCore","brand":"Anker","model":"PowerCore 20000","quantity":1,"unit":"un","purchase_price":189.90,"condition":"novo","category_id":1,"location_id":9}`

**Claude responde:**
> Item cadastrado com sucesso!
> **Código:** EST-E7C1D3F9
> **Local:** Escritório › Gaveta
> **Categoria:** Eletrônicos
> Um registro de entrada foi criado automaticamente em Movimentações.

---

## 4. Mover um item para outro local

**Você digita:**
> movi a furadeira para a bancada da garagem

**Claude chama:**
1. `find_item_location {"query": "furadeira"}` — confirma id 42
2. `list_locations` — confirma que "Bancada" (dentro de Garagem) é id 7
3. `move_item {"item_id": 42, "to_location_id": 7, "reason": "Uso na bancada"}`

**Claude responde:**
> Feito! **Furadeira de Impacto** movida de
> Garagem › Armário de Ferramentas › Caixa Vermelha
> para **Garagem › Bancada**.
> Movimentação registrada — você pode ver o histórico completo na aba Movimentações da UI.

---

## 5. Atualizar condição e adicionar observação

**Você digita:**
> a furadeira está com defeito na embreagem, marca como condição "ruim" e adicione uma nota: "embreagem com folga, precisa revisão"

**Claude chama:** `update_item {"id": 42, "condition": "ruim", "notes": "embreagem com folga, precisa revisão"}`

**Claude responde:**
> Atualizado! **Furadeira de Impacto** (EST-A3F7B219)
> Condição: ~~bom~~ → **ruim**
> Nota salva: "embreagem com folga, precisa revisão"

---

## 6. Inventário de um cômodo específico

**Você digita:**
> o que tenho guardado na garagem?

**Claude chama:**
1. `list_locations` — identifica todos os IDs dentro de "Garagem"
2. `list_items {"location_id": 1}` (repete para sub-locais se necessário)

**Claude responde:**
> Na **Garagem** e suas sub-localizações encontrei 14 itens:
>
> **Armário de Ferramentas › Caixa Vermelha** (5 itens)
> - Furadeira de Impacto Bosch — ruim
> - Martelo Stanley 500g — bom
> - Jogo de Chaves Philips — bom
> - Alicate Universal — bom
> - Trena 5m — novo
>
> **Bancada** (3 itens)
> - Serra Circular Makita — bom
> - Lixadeira Orbital — bom
> - Furadeira de Impacto (em uso) — ruim
> _(... e mais 6 em outros locais)_

---

## 7. Criar uma nova localização e já cadastrar itens nela

**Você digita:**
> criei uma nova caixa plástica transparente no armário do corredor (id 5) para guardar pilhas e baterias. Depois cadastre 4 pilhas AA e 6 pilhas AAA lá.

**Claude chama (em sequência):**
1. `create_location {"name":"Caixa de Pilhas e Baterias","type":"caixa","parent_id":5,"description":"Pilhas alcalinas e recarregáveis"}`
   — retorna id 11
2. `create_item {"name":"Pilhas AA","brand":"Duracell","quantity":4,"unit":"un","condition":"novo","location_id":11}`
3. `create_item {"name":"Pilhas AAA","brand":"Duracell","quantity":6,"unit":"un","condition":"novo","location_id":11}`

**Claude responde:**
> Localização criada: **Corredor › Armário do Corredor › Caixa de Pilhas e Baterias**
> Itens cadastrados:
> - Pilhas AA Duracell × 4 (EST-A1B2C3D4)
> - Pilhas AAA Duracell × 6 (EST-E5F6G7H8)
> Todos com entrada registrada em Movimentações (autor: MCP Assistant).

---

## 8. Consulta de valor do inventário por categoria

**Você digita:**
> qual o valor total dos meus eletrônicos? liste os 5 mais caros

**Claude chama:** `list_items {"category_id": 1, "limit": 50}`

**Claude responde (calculando com os dados retornados):**
> **Eletrônicos — valor total: R$ 14.320,80** (15 itens)
>
> Top 5 por valor:
> | Item | Valor |
> |------|-------|
> | Notebook Dell Inspiron 15 3511 | R$ 3.299,00 |
> | Smart TV Samsung 55" | R$ 2.999,00 |
> | MacBook Air M1 | R$ 2.800,00 |
> | Monitor Samsung 27" | R$ 1.199,00 |
> | Carregador Portátil Anker | R$ 189,90 |

---

## 9. Verificar número de série de um item

**Você digita:**
> qual o número de série do notebook?

**Claude chama:** `find_item_location {"query": "notebook"}`

**Claude responde:**
> Encontrei 2 notebooks:
> 1. **Notebook Dell Inspiron** (EST-C8D2E1F4) — S/N: **DELL2024-XK91** — Escritório › Mesa
> 2. **MacBook Air M1** (EST-7A3B2C1D) — S/N: **C02XK1ABMD6T** — Quarto Principal › Escrivaninha
>
> Qual você precisava?

---

## 10. Registro de item emprestado

**Você digita:**
> emprestei a furadeira pro João. Mova ela para "Emprestados" com o motivo "Emprestada para João – reforma do banheiro"

**Claude chama:**
1. `list_locations` — confirma id de "Emprestados" (ou sugere criar se não existir)
2. `move_item {"item_id": 42, "to_location_id": 12, "reason": "Emprestada para João – reforma do banheiro"}`

**Claude responde:**
> Registrado! **Furadeira de Impacto** movida para **Emprestados**.
> Motivo salvo: "Emprestada para João – reforma do banheiro"
> Quando devolver, me avise que movo de volta para a Caixa Vermelha.

---

## 11. Auditoria rápida — itens sem localização

**Você digita:**
> tem algum item sem localização definida?

**Claude chama:** `list_items {"limit": 50}` (e filtra os que têm `location_path` vazio)

**Claude responde:**
> Sim, encontrei 3 itens sem localização:
> - **Chave de Fenda Phillips** (EST-2B4D6F8A)
> - **Extensão Elétrica 10m** (EST-1C3E5G7H)
> - **Caixa de Pregos** (EST-9A8B7C6D)
>
> Quer que eu os mova para algum local agora?

---

## 12. Cadastro em lote por conversa natural

**Você digita:**
> comprei hoje na ferreteria: 1 serrote Tramontina R$45, 2 lixas de parede grão 80 R$3,50 cada, 1 fita isolante 3M R$8. Tudo vai para a Caixa Vermelha da garagem.

**Claude chama (3 vezes `create_item`):**

```
create_item {
  "name": "Serrote",
  "brand": "Tramontina",
  "quantity": 1,
  "purchase_price": 45.00,
  "purchase_date": "2026-05-22",
  "condition": "novo",
  "category_id": 2,
  "location_id": 3
}

create_item {
  "name": "Lixa de Parede Grão 80",
  "quantity": 2,
  "purchase_price": 3.50,
  "purchase_date": "2026-05-22",
  "condition": "novo",
  "category_id": 2,
  "location_id": 3
}

create_item {
  "name": "Fita Isolante",
  "brand": "3M",
  "quantity": 1,
  "purchase_price": 8.00,
  "purchase_date": "2026-05-22",
  "condition": "novo",
  "category_id": 2,
  "location_id": 3
}
```

**Claude responde:**
> 3 itens cadastrados em **Garagem › Armário de Ferramentas › Caixa Vermelha**:
> - Serrote Tramontina (EST-A9B8C7D6) — R$ 45,00
> - Lixa de Parede Grão 80 × 2 (EST-F5E4D3C2) — R$ 3,50/un
> - Fita Isolante 3M (EST-B1C2D3E4) — R$ 8,00
>
> Total gasto hoje: **R$ 60,00**

---

## Dicas de uso

- **Seja descritivo nas buscas**: "furadeira bosch" encontra mais rápido que "furadeira"
- **Confirme IDs antes de mover**: Claude geralmente chama `list_locations` para confirmar, mas você pode informar o ID direto para acelerar
- **Use o motivo nas movimentações**: fica registrado no histórico e aparece na UI
- **Veja o resultado na UI**: todas as operações feitas via MCP aparecem em tempo real em `http://localhost:5173` — útil para confirmar o que foi feito
- **Movimentos do MCP**: na página Movimentações, filtre pelo usuário "MCP Assistant" para ver apenas o que o Claude fez
