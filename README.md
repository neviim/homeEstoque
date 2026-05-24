# HomeEstoque

Sistema completo de **controle de estoque doméstico** com backend em Go e frontend em React. Cadastre tudo o que você tem em casa — eletrônicos, cabos, ferramentas, louças, alimentos, hardware, chaves — e saiba exatamente em qual cômodo, bancada ou caixa está guardado.

## Funcionalidades

- Cadastro completo de itens com marca, modelo, número de série, condição, fotos
- Hierarquia de **categorias** com cores personalizáveis (12 categorias pré-cadastradas)
- Hierarquia de **locais** (cômodo > móvel > caixa) com caminho completo
- **Fotos** múltiplas por item (upload local)
- **QR Code** automático para imprimir e colar em caixas/itens
- **Histórico de movimentações** quando itens trocam de local
- **Busca** por nome, código, descrição, marca, modelo
- Filtros por categoria e local
- **Exportação CSV** do inventário completo
- **Autenticação JWT** multi-usuário com aprovação de cadastros pendentes
- **Sistema de permissões granulares** estilo Discord — perfis customizáveis com 15 capacidades editáveis (UI em `/sistema/permissoes`)
- **Servidor MCP** — Claude pode consultar e movimentar itens por linguagem natural
- Dashboard com estatísticas e valor patrimonial estimado
- **256 testes automatizados** (Go + Vitest + Playwright E2E) — rode `./test.sh` na raiz; CI no GitHub Actions; veja [docs/testes.md](docs/testes.md)

## Stack

| Camada   | Tecnologia |
|----------|-----------|
| Backend  | Go 1.22+, chi router, SQLite (modernc puro Go) |
| Auth     | JWT + bcrypt |
| Frontend | React 18, Vite, TypeScript, TailwindCSS, TanStack Query, React Router |
| Outros   | go-qrcode, react-hot-toast, lucide-react |

A comunicação é **100% via API REST** — backend e frontend podem rodar/serem deployados de forma independente.

## Estrutura

```
homeEstoque/
├── backend/                  API Go + MCP Server
│   ├── cmd/
│   │   ├── api/main.go      Entrypoint HTTP
│   │   └── mcp/main.go      Entrypoint MCP (stdio)
│   ├── internal/
│   │   ├── auth/            JWT + bcrypt
│   │   ├── config/          ENV loader
│   │   ├── database/        SQLite + migrations + seed (incl. roles)
│   │   ├── handlers/        REST endpoints (incl. users e roles)
│   │   ├── locpath/         Caminho hierárquico de localização
│   │   ├── mcptools/        Implementação das 10 tools MCP
│   │   ├── middleware/      JWT + RequirePermission(db, key)
│   │   ├── models/          DTOs
│   │   └── permissions/     Catálogo + service de permissões
│   ├── data/                Banco SQLite (gerado)
│   ├── uploads/             Fotos (gerado)
│   ├── go.mod
│   └── .env.example
└── frontend/                 SPA React
    ├── src/
    │   ├── components/      UI reutilizável + Layout + ProfileModal
    │   ├── pages/           Dashboard, Items, Categorias, Locais, Movimentações, Login, Users, Permissions
    │   ├── hooks/           useAuth + hasPermission
    │   ├── lib/             api axios (com interceptors 401/403), utils
    │   └── types/           Tipos TypeScript (User, Role, Permission)
    ├── vite.config.ts
    └── package.json
```

## Como rodar

### 1. Backend (porta 8080)

```bash
cd backend
cp .env.example .env             # opcional - ajuste JWT_SECRET em produção
go mod tidy
go run ./cmd/api
```

O servidor cria automaticamente:
- `./data/homeestoque.db` (SQLite)
- `./uploads/` (fotos)
- Categorias e locais iniciais

### 2. Frontend (porta 5173)

```bash
cd frontend
npm install
npm run dev
```

Abra http://localhost:5173 — crie sua conta na tela inicial (botão **Criar conta**).

> O Vite já está configurado com proxy `/api` → `localhost:8080`, então não precisa configurar CORS para desenvolvimento.

### Build de produção

```bash
# Frontend (gera /frontend/dist)
cd frontend && npm run build

# Backend
cd backend && go build -o bin/api ./cmd/api
```

Você pode servir o `dist/` por qualquer estático (nginx, Caddy) ou apontar o backend para servi-lo.

## Servidor MCP (Claude Code / Claude Desktop)

Além da API HTTP, o projeto inclui um servidor **MCP** (Model Context Protocol) que permite ao Claude consultar e movimentar itens do estoque por linguagem natural ("onde está minha furadeira?", "crie um item chamado X no Quarto"). Ele compartilha o mesmo SQLite da API via WAL — podem rodar simultaneamente.

```bash
# 1. Compilar (gera ./bin/homeestoque-mcp)
./tools/build-mcp.sh

# 2. Registrar no Claude Code
claude mcp add homeestoque \
  --scope local \
  -e DB_PATH=$(pwd)/backend/data/homeestoque.db \
  -- $(pwd)/bin/homeestoque-mcp

# 3. Reiniciar Claude Code e listar
/mcp
# Deve aparecer "homeestoque" com 10 tools
```

10 ferramentas disponíveis: `find_item_location`, `list_items`, `get_item`, `list_categories`, `list_locations`, `create_item`, `create_category`, `create_location`, `update_item`, `move_item` (sem delete, por segurança).

**Documentação completa:** [docs/mcp/](docs/mcp/) — inclui [servidor.md](docs/mcp/servidor.md) (compilação e configuração), [ferramentas.md](docs/mcp/ferramentas.md) (referência das 10 tools) e [exemplos-claude-code.md](docs/mcp/exemplos-claude-code.md) (12 casos de uso reais).

## Endpoints principais

| Método | Rota | Permissão |
|--------|------|-----------|
| POST | `/api/auth/register` | público — entra como `pending` (exceto primeiro user) |
| POST | `/api/auth/login` | público |
| GET | `/api/auth/me`, `/api/permissions`, `/api/roles` | autenticado |
| PUT | `/api/auth/profile`, `/api/auth/password` | autenticado |
| GET | `/api/dashboard` | `dashboard.view` |
| GET | `/api/items` (lista, detalhe, movimentos) | `items.view` |
| POST/PUT/DELETE | `/api/items/{id}` | `items.create` / `items.update` / `items.delete` |
| POST/DELETE | `/api/items/{id}/photos/...` | `items.upload_photo` |
| GET | `/api/items/{id}/qrcode` | público (sem auth) |
| GET | `/api/categories`, `/api/locations` | respectivo `.view` |
| POST/PUT/DELETE | categorias e locais | respectivo `.manage` |
| GET | `/api/movements` | `movements.view` |
| GET | `/api/export/csv` | `export.csv` |
| GET/POST/PUT/DELETE | `/api/users/...` | `users.manage` |
| GET/POST/PUT/DELETE | `/api/roles/...` | `roles.manage` |

Documentação detalhada com exemplos curl em [docs/api/](docs/api/).

Todos os endpoints protegidos requerem header:
```
Authorization: Bearer <token>
```

## Categorias pré-cadastradas

Eletrônicos · Computadores · Cabos e Adaptadores · Hardware/Componentes · Ferramentas · Chaves · Louças e Cozinha · Alimentos · Livros · Roupas · Documentos · Outros

## Locais pré-cadastrados

Sala · Quarto · Cozinha · Escritório · Garagem · Bancada · Armário · Caixa Organizadora

## Configuração (backend/.env)

```
PORT=8080
DB_PATH=./data/homeestoque.db
JWT_SECRET=change-me-to-a-long-random-secret
UPLOAD_DIR=./uploads
CORS_ORIGINS=http://localhost:5173
```

## Imprimir etiquetas com QR Code

1. Acesse o item → botão **QR Code**
2. Abra em nova aba ou imprima diretamente
3. Cole a etiqueta na caixa/item — escaneando, abre direto a página de detalhes
