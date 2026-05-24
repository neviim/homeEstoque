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
│   │   ├── backup/              # Create, Restore, Verify, Scheduler, Manager
│   │   ├── config/              # variáveis de ambiente (.env) + BACKUP_DIR
│   │   ├── database/            # Open() + migrate() + Seed() (inclui seed de roles)
│   │   ├── handlers/            # handlers HTTP (chi) — inclui backup_handler
│   │   ├── locpath/             # construção de caminho hierárquico de local
│   │   ├── mcptools/            # implementação das 10 tools MCP
│   │   ├── middleware/          # JWT + RequirePermission + MaintenanceGate
│   │   ├── models/              # structs compartilhados (Item, Movement…)
│   │   └── permissions/         # catálogo + service (HasPermission, UserPermissions)
│   ├── data/homeestoque.db      # arquivo SQLite (gerado automaticamente)
│   ├── data/backups/            # backups .tar.gz (BACKUP_DIR, default ./data/backups)
│   ├── uploads/                 # fotos enviadas
│   ├── go.mod
│   └── .env
├── frontend/
│   ├── src/
│   │   ├── components/          # Layout, Pagination, Modal, PageHeader, ProfileModal
│   │   ├── hooks/useAuth.tsx    # contexto + hasPermission(key)
│   │   ├── lib/api.ts           # cliente HTTP (axios + interceptors 401/403)
│   │   ├── pages/               # Items, Movements, Dashboard, Login, Users, Permissions, Backup
│   │   └── types/index.ts       # interfaces TypeScript (User, Role, Permission)
│   └── vite.config.ts           # proxy /api → :8080
├── bin/
│   └── homeestoque-mcp          # binário MCP compilado
├── tools/
│   ├── build-mcp.sh             # compila o servidor MCP
│   ├── reset-password.sh        # redefine senha de usuário direto no DB
│   └── seed-demo.sh             # popula/limpa dados de demonstração
└── docs/                        # esta documentação
```

## Banco de dados

SQLite com WAL mode — permite múltiplos readers e um writer simultâneo, essencial para rodar API HTTP e servidor MCP ao mesmo tempo sem erro de lock.

### Tabelas

```sql
users             -- autenticação; email único; role (FK por nome → roles.name); status; "MCP Assistant" criado pelo seed
roles             -- perfis customizáveis; name (slug único); is_system=1 protege contra exclusão/renomeação
role_permissions  -- N:N entre roles e permissions; (role_id, permission) é PK
categories        -- hierárquica (parent_id self-ref); icon e color opcionais
locations         -- hierárquica; type: comodo|movel|caixa|armario|outro
items             -- inventário principal; code único "EST-XXXXXXXX"; refs a category/location
item_photos       -- fotos dos itens; CASCADE delete com o item
movements         -- log de movimentações; from_location → to_location; user_id
backups           -- registro de cada arquivo .tar.gz; sha256, status (ok/corrupted/missing/orphan/unverified)
backup_schedule   -- singleton (id=1); frequência/horário/retenção do agendamento automático
```

### Índices

```sql
idx_items_category         ON items(category_id)
idx_items_location         ON items(location_id)
idx_items_name             ON items(name)
idx_movements_item         ON movements(item_id)
idx_role_permissions_role  ON role_permissions(role_id)
```

### Status de usuário

| Valor | Significado |
|-------|-------------|
| `active` | Pode fazer login |
| `pending` | Aguardando aprovação de admin |
| `inactive` | Desativado; login bloqueado |

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

## Sistema de permissões (estilo Discord)

Modelo de autorização granular: cada endpoint exige uma **permissão nomeada**. Cada usuário tem um **perfil (role)** que agrupa as permissões concedidas. Admin pode criar/editar/excluir perfis customizados e ativar/desativar permissões via `/sistema/permissoes` na UI.

### Pacote `permissions`

```go
permissions.Catalog                       // []Permission — fonte da verdade das 19 capacidades
permissions.Keys()                        // []string — todas as keys (usado no seed do admin)
permissions.Exists(key)                   // valida key contra o catálogo
permissions.HasPermission(db, uid, key)   // usado pelo middleware
permissions.UserPermissions(db, uid)      // []string — todas as perms efetivas do usuário
permissions.RolePermissions(db, role)     // []string — perms diretas do role
```

### Middleware

```go
middleware.RequirePermission(db, "items.create")  // 403 se faltar a perm
```

Substitui os antigos `RequireWriter` e `RequireAdmin`. Cada rota declara sua permissão específica em `cmd/api/main.go`. **A consulta vai sempre ao DB** — mudanças de permissão valem no próximo request, sem precisar invalidar o JWT.

### Seed idempotente

Em `seed.go::seedRoles()`:

1. Cria `admin` (is_system=1), `user`, `viewer` se ainda não existirem (`INSERT OR IGNORE`).
2. Sincroniza `admin` com **todas as keys do catálogo** a cada startup — assim novas permissões adicionadas no código são automaticamente concedidas ao admin.
3. Para `user` e `viewer`: aplica defaults **apenas se o role estiver sem nenhuma permissão** (não sobrescreve configurações customizadas do admin).

### Garantias

| Invariante | Onde é garantida |
|------------|------------------|
| Admin sempre tem todas as permissões | Seed (startup) + `roles_handler.UpdatePermissions` (força inclusão) |
| Admin nunca pode ser excluído | `roles_handler.Delete` (`is_system=1` → 403) |
| Admin nunca pode ser renomeado | `roles_handler.Update` (campo `name` ignorado se `is_system=1`) |
| Perfil só pode ser excluído sem usuários | `roles_handler.Delete` (409 se há users atribuídos) |
| Último admin não pode ser inativado/excluído | `user_handler.UpdateStatus`/`Delete` |
| `users.role` sempre referencia perfil existente | `user_handler.Create/Update` (`roleExists()`) + transação no rename de role |

## Módulo de Backup (`internal/backup`)

### Visão geral

O módulo produz arquivos `.tar.gz` contendo o DB SQLite (snapshot via `VACUUM INTO`) e a pasta `uploads/`. Cada arquivo é registrado na tabela `backups` com sha256, tamanho e status. O admin pode criar backups manuais, baixar, verificar integridade, restaurar e configurar agendamento automático pela UI em **Sistema → Backup**.

### Componentes

| Arquivo | Responsabilidade |
|---------|------------------|
| `backup.go` | `Manager` struct (ponto central); `Create(ctx, kind)` — snapshot + tar.gz + sha256 + INSERT; `List`, `GetByID`, `Delete`, `Verify`, `PrepareRestore` |
| `restore.go` | `Restore(ctx, id, token)` — valida token, cria snapshot de segurança, ativa maintenance mode, fecha DB, extrai arquivos, chama `RestartFunc` |
| `scheduler.go` | `StartScheduler` / `StopScheduler` / `Reload`; cron `robfig/cron/v3`; `UpdateSchedule` persiste e recarrega a quente; poda automática por `retention_count` |

### Snapshot consistente com `VACUUM INTO`

`modernc.org/sqlite` (pure-Go) não expõe `sqlite3_backup_init`. `VACUUM INTO 'path'` é uma única SQL statement que produz um snapshot sem os arquivos WAL sidecar, bloqueando escritas apenas brevemente.

### Restore e reinicialização

Após extrair o arquivo, o handler chama uma `RestartFunc` injetada (`os.Exit(0)` com goroutine delayed). Em desenvolvimento, Air detecta a saída e reinicia o processo; em produção, systemd ou Docker faz o mesmo. Isso evita a complexidade de swapping de `*sql.DB` em runtime.

### Maintenance mode

`middleware.MaintenanceGate` lê um `atomic.Bool` exposto pelo `Manager`. Quando ativo (durante restore), todas as rotas retornam `503 Service Unavailable`, exceto os prefixos autorizados (`/api/backups/`). Isso impede requests ao DB enquanto os arquivos estão sendo substituídos.

### Permissões de backup

| Chave | Concedida a (seed) | Operações |
|-------|-------------------|-----------|
| `backup.create` | admin | Criar, listar, verificar, excluir |
| `backup.restore` | admin | Preparar e executar restore |
| `backup.download` | admin | Download do `.tar.gz` |
| `backup.schedule` | admin | Ler e atualizar agendamento |

### Rotas HTTP

| Método | Rota | Permissão |
|--------|------|-----------|
| `GET` | `/api/backups` | `backup.create` |
| `POST` | `/api/backups` | `backup.create` |
| `POST` | `/api/backups/{id}/verify` | `backup.create` |
| `GET` | `/api/backups/{id}/download` | `backup.download` |
| `POST` | `/api/backups/{id}/restore/prepare` | `backup.restore` |
| `POST` | `/api/backups/{id}/restore` | `backup.restore` |
| `DELETE` | `/api/backups/{id}` | `backup.create` |
| `GET` | `/api/backup/schedule` | `backup.schedule` |
| `PUT` | `/api/backup/schedule` | `backup.schedule` |
