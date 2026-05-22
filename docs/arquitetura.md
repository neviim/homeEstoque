# Arquitetura

## Visão geral

```
┌──────────────────────────────────────────────────────────────┐
│  Browser                                                      │
│  React SPA (Vite :5173)                                       │
│  TanStack Query → /api/*                                      │
└──────────────────┬───────────────────────────────────────────┘
                   │ HTTP REST (CORS: localhost:5173)
┌──────────────────▼───────────────────────────────────────────┐
│  Go HTTP Server (:8080)                                       │
│  chi · JWT middleware · handlers                              │
│                          │                                    │
│  ┌────────────────────────▼────────────────────────────────┐  │
│  │  SQLite (WAL mode)                                      │  │
│  │  backend/data/homeestoque.db                            │  │
│  └────────────────────────▲────────────────────────────────┘  │
│                          │                                    │
│  MCP Server (stdio)       │                                    │
│  bin/homeestoque-mcp ─────┘                                   │
└──────────────────────────────────────────────────────────────┘
         ▲
         │ stdio JSON-RPC
┌────────┴─────────────────────────────────────────────────────┐
│  Claude Code / Claude Desktop                                 │
│  subprocess do binário MCP                                    │
└──────────────────────────────────────────────────────────────┘
```

## Estrutura de diretórios

```
homeEstoque/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go          # entry point: HTTP server
│   │   └── mcp/main.go          # entry point: MCP server stdio
│   ├── internal/
│   │   ├── auth/                # geração e validação JWT
│   │   ├── config/              # variáveis de ambiente (.env)
│   │   ├── database/            # Open() + migrate() + Seed()
│   │   ├── handlers/            # handlers HTTP (chi)
│   │   ├── locpath/             # construção de caminho hierárquico de local
│   │   ├── mcptools/            # implementação das 10 tools MCP
│   │   ├── middleware/          # middleware JWT para chi
│   │   └── models/              # structs compartilhados (Item, Movement…)
│   ├── data/homeestoque.db      # arquivo SQLite (gerado automaticamente)
│   ├── uploads/                 # fotos enviadas
│   ├── go.mod
│   └── .env
├── frontend/
│   ├── src/
│   │   ├── components/          # Layout, Pagination, Modal, PageHeader…
│   │   ├── hooks/useAuth.tsx    # contexto de autenticação
│   │   ├── lib/api.ts           # cliente HTTP (fetch + token)
│   │   ├── pages/               # Items, Movements, Dashboard, Login…
│   │   └── types/index.ts       # interfaces TypeScript
│   └── vite.config.ts           # proxy /api → :8080
├── bin/
│   └── homeestoque-mcp          # binário MCP compilado
├── tools/
│   ├── build-mcp.sh             # compila o servidor MCP
│   └── seed-demo.sh             # popula/limpa dados de demonstração
└── docs/                        # esta documentação
```

## Banco de dados

SQLite com WAL mode — permite múltiplos readers e um writer simultâneo, essencial para rodar API HTTP e servidor MCP ao mesmo tempo sem erro de lock.

### Tabelas

```sql
users        -- autenticação; email único; "MCP Assistant" criado pelo seed
categories   -- hierárquica (parent_id self-ref); icon e color opcionais
locations    -- hierárquica; type: comodo|movel|caixa|armario|outro
items        -- inventário principal; code único "EST-XXXXXXXX"; refs a category/location
item_photos  -- fotos dos itens; CASCADE delete com o item
movements    -- log de movimentações; from_location → to_location; user_id
```

### Índices

```sql
idx_items_category   ON items(category_id)
idx_items_location   ON items(location_id)
idx_items_name       ON items(name)
idx_movements_item   ON movements(item_id)
```

### Condições de item

| Valor | Significado |
|-------|-------------|
| `novo` | Item nunca usado |
| `bom` | Bom estado geral |
| `regular` | Funcionando, com desgaste |
| `ruim` | Danificado / precisando reparo |

### Tipos de localização

| Valor | Significado |
|-------|-------------|
| `comodo` | Cômodo da casa (quarto, garagem…) |
| `movel` | Móvel (guarda-roupa, estante…) |
| `caixa` | Caixa ou container |
| `armario` | Armário específico |
| `outro` | Qualquer outro tipo |

## Geração de código SKU

Todo item criado recebe um código único no formato `EST-XXXXXXXX`, onde os 8 caracteres são os primeiros 8 chars de um UUID v4 em maiúsculo:

```go
"EST-" + strings.ToUpper(uuid.New().String()[:8])
// Exemplo: EST-A3F7B219
```

## Rastreabilidade de movimentações via MCP

O seed cria um usuário especial `mcp@homeestoque.local` ("MCP Assistant"). O servidor MCP resolve o ID desse usuário no startup e o usa como `user_id` em todos os `INSERT INTO movements`. Isso torna os movimentos feitos pelo Claude visíveis e distinguíveis na UI em "Movimentações".

## Pacote `locpath`

Extraído de `handlers/location_handler.go` para ser compartilhado entre handlers HTTP e mcptools sem duplicação. Expõe:

```go
locpath.LoadLocationMap(db)              // lê toda a tabela locations em memória
locpath.BuildFullPath(db, id)            // constrói "Garagem > Caixa Ferramentas" para um ID
locpath.BuildFullPathFromMap(map, id)    // versão batch (usa mapa já carregado)
```
