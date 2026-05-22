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
- **Autenticação JWT** multi-usuário (família)
- Dashboard com estatísticas e valor patrimonial estimado

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
├── backend/                  API Go
│   ├── cmd/api/main.go      Entrypoint
│   ├── internal/
│   │   ├── auth/            JWT + bcrypt
│   │   ├── config/          ENV loader
│   │   ├── database/        SQLite + migrations + seed
│   │   ├── handlers/        REST endpoints
│   │   ├── middleware/      Auth middleware
│   │   └── models/          DTOs
│   ├── data/                Banco SQLite (gerado)
│   ├── uploads/             Fotos (gerado)
│   ├── go.mod
│   └── .env.example
└── frontend/                 SPA React
    ├── src/
    │   ├── components/      UI reutilizável + Layout
    │   ├── pages/           Dashboard, Items, Categorias, Locais, Movimentações, Login
    │   ├── hooks/           useAuth
    │   ├── lib/             api axios, utils
    │   └── types/           Tipos TypeScript
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

## Endpoints principais

| Método | Rota                                  | Descrição |
|--------|---------------------------------------|-----------|
| POST   | `/api/auth/register`                  | Cria usuário e devolve JWT |
| POST   | `/api/auth/login`                     | Login → JWT |
| GET    | `/api/auth/me`                        | Usuário atual |
| GET    | `/api/dashboard`                      | Estatísticas |
| GET    | `/api/items?search=&category_id=&location_id=` | Lista itens com filtros |
| POST   | `/api/items`                          | Cria item |
| GET    | `/api/items/{id}`                     | Detalhes + fotos |
| PUT    | `/api/items/{id}`                     | Atualiza (registra movimentação se mudou local) |
| DELETE | `/api/items/{id}`                     | Exclui (incluindo fotos físicas) |
| POST   | `/api/items/{id}/photos`              | Upload foto (multipart, campo `photo`) |
| DELETE | `/api/items/{id}/photos/{photoId}`    | Remove foto |
| GET    | `/api/items/{id}/qrcode`              | PNG QR Code |
| GET    | `/api/items/{id}/movements`           | Histórico do item |
| GET    | `/api/movements`                      | Últimas 100 movimentações |
| GET    | `/api/categories` `POST` `PUT` `DELETE` | CRUD categorias |
| GET    | `/api/locations` `POST` `PUT` `DELETE`  | CRUD locais (hierárquicos) |
| GET    | `/api/export/csv`                     | Download CSV (UTF-8 BOM, separador `;`) |

Todos os endpoints (exceto `/auth/*` e `/health`) requerem header:
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
