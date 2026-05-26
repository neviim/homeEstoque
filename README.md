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
| Backend  | Go 1.25 (gerenciado via [mise](https://mise.jdx.dev/)), chi router, SQLite (modernc puro Go) |
| Auth     | JWT + bcrypt |
| Frontend | React 18, Vite, TypeScript, TailwindCSS, TanStack Query, React Router |
| Outros   | go-qrcode, react-hot-toast, lucide-react |
| Release  | GoReleaser → binários Linux/macOS/Windows publicados automaticamente em GitHub Releases ao empurrar tag `vX.Y.Z` |

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

## Pré-requisitos

- **Node.js 20+** com `npm`.
- **Go 1.25** — instale via [mise](https://mise.jdx.dev/) (o projeto fixa a versão em `mise.toml` e o shell ativa automaticamente ao entrar no diretório):

  ```bash
  curl https://mise.run | sh
  echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc   # ou ~/.zshrc
  ```

  Depois, na primeira vez no projeto: `mise install` (baixa Go 1.25 isoladamente — não interfere em outros projetos com versões diferentes de Go).

## Como rodar

### Opção A — Binários pré-compilados (mais rápido)

Cada tag `vX.Y.Z` publica binários em GitHub Releases. Pegue o último direto, sem clonar nem compilar:

```bash
# Backend (escolha sua plataforma; URLs com /latest/ apontam sempre para a versão atual)
curl -L -o homeestoque.tar.gz \
  https://github.com/neviim/homeEstoque/releases/latest/download/homeestoque_linux_amd64.tar.gz
tar xzf homeestoque.tar.gz
./homeestoque                      # sobe API em :8080

# Frontend (SPA estática)
curl -L -o frontend.tar.gz \
  https://github.com/neviim/homeEstoque/releases/latest/download/homeestoque_frontend.tar.gz
tar xzf frontend.tar.gz            # extrai ./frontend/ com a build pronta
```

Sirva o `frontend/` com nginx/Caddy fazendo proxy de `/api` → backend `:8080`.

Assets disponíveis (sem versão no nome, URLs estáveis):
- `homeestoque_linux_amd64.tar.gz` · `homeestoque_linux_arm64.tar.gz`
- `homeestoque_darwin_amd64.tar.gz` · `homeestoque_darwin_arm64.tar.gz`
- `homeestoque_windows_amd64.zip`
- `homeestoque_frontend.tar.gz`
- `checksums.txt` (SHA-256 dos arquivos acima)

### Opção B — Rodar do código (desenvolvimento)

```bash
git clone https://github.com/neviim/homeEstoque
cd homeEstoque
mise install                       # baixa o Go 1.25 conforme mise.toml

# Backend (porta 8080)
cd backend
cp .env.example .env               # opcional — ajuste JWT_SECRET em produção
go mod tidy
go run ./cmd/api

# Frontend (porta 5173) — em outro terminal
cd frontend
npm install
npm run dev
```

Abra http://localhost:5173 — crie sua conta na tela inicial (botão **Criar conta**).

> O Vite já vem com proxy `/api` → `localhost:8080`, sem precisar de CORS pra dev.

O backend cria automaticamente:
- `./data/homeestoque.db` (SQLite)
- `./uploads/` (fotos)
- Categorias e locais iniciais

#### Atalho: subir tudo em um terminal

```bash
./start-dev.sh                     # backend com hot-reload (Air) + frontend
```

### Build de produção (local)

```bash
./build.sh                         # bump patch + compila tudo
# Gera:
#   bin/api                        backend HTTP
#   bin/homeestoque-mcp            servidor MCP
#   frontend/dist/                 SPA build

./build.sh --minor                 # bump X.(Y+1).0
./build.sh --major                 # bump (X+1).0.0
```

### Disparar um release automático

Empurre uma tag — o workflow `.github/workflows/release.yml` roda os testes, compila multi-plataforma e publica em GitHub Releases:

```bash
git tag -a v0.1.2 -m "Release v0.1.2"
git push origin v0.1.2
```

Em ~3 minutos, os artefatos da Opção A ficam disponíveis na nova tag.

### Resetar a senha de um usuário (admin local)

```bash
./tools/reset-password.sh <email> <nova-senha>
```

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
