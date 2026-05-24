# Testes

O HomeEstoque tem **256 testes/cenários automatizados** cobrindo backend Go, frontend React e fluxos E2E. Esta página explica como rodar, onde estão organizados e o que cada camada cobre.

## Como rodar

### Atalho — script `./test.sh` na raiz

```bash
./test.sh              # roda tudo (backend + frontend + E2E)
./test.sh --fast       # pula E2E (~2x mais rápido)
./test.sh --quiet      # só uma linha por camada; detalhe completo só se falhar
./test.sh backend      # só backend Go
./test.sh frontend     # só Vitest
./test.sh e2e          # só Playwright (compila MCP se necessário)
./test.sh --coverage   # com relatórios de cobertura
```

Resumo colorido no final mostra `PASS`/`FAIL` e tempo por camada. Código de saída 0 = tudo passou; 1 = alguma camada falhou (todas as camadas rodam mesmo se uma falhar, pra você ver tudo de uma vez).

**Modo `--quiet`** (ou `-q`): ideal para uso em hooks e CI local. Cada camada vira uma linha `Nome ... ✓  N testes` ou `Nome ... ✗`. Quando alguma falhar, o output completo da camada quebrada é impresso. O resumo final mostra o **total de testes** somado entre as 3 camadas. Combine com `--fast` para feedback de ~30s sem E2E.

Exemplo de output:
```
HomeEstoque — suíte de testes
Modo: all +fast +quiet
  Backend Go ... ✓  172 testes
  Frontend Vitest ... ✓  84 testes

━━━ Resumo ━━━
  Backend Go                PASS   172 testes  (2s)
  Frontend Vitest           PASS    84 testes  (29s)
  Total: 256 testes

Todos os testes passaram ✓
```

### Manual — comandos individuais

#### Backend (Go)
```bash
cd backend
GOROOT=/home/neviim/go GOPATH=/home/neviim/go /home/neviim/go/bin/go test -race ./...

# Com cobertura
GOROOT=/home/neviim/go GOPATH=/home/neviim/go /home/neviim/go/bin/go test \
  -race -coverprofile=coverage.out ./...
GOROOT=/home/neviim/go /home/neviim/go/bin/go tool cover -html=coverage.out
```

#### Frontend (Vitest)
```bash
cd frontend
npm test                # roda uma vez
npm run test:watch      # watch mode (HMR de testes)
npm run test:coverage   # gera relatório HTML em coverage/
```

#### E2E (Playwright)
```bash
cd tests/e2e
npm test                # headless
npm run test:headed     # com browser visível
npm run test:ui         # UI mode (debugger interativo)
```

> O `webServer` config do Playwright sobe backend (porta 8090) e frontend (porta 5174) automaticamente — não precisa rodar `./start-dev.sh` em paralelo.

#### Tudo de uma vez (replica o CI)
```bash
# Backend
cd backend && GOROOT=/home/neviim/go GOPATH=/home/neviim/go /home/neviim/go/bin/go test -race ./...

# Frontend
cd frontend && npm test

# E2E (requer MCP binary compilado)
./tools/build-mcp.sh
cd tests/e2e && npm test
```

---

## Estrutura

```
homeEstoque/
├── backend/internal/
│   ├── auth/auth_test.go                    ← unit
│   ├── permissions/
│   │   ├── catalog_test.go                  ← unit
│   │   └── service_test.go                  ← integration (SQLite real)
│   ├── locpath/locpath_test.go              ← (futuro — não coberto ainda)
│   ├── middleware/middleware_test.go        ← integration (httptest)
│   ├── database/
│   │   ├── migrate_test.go                  ← integration
│   │   └── seed_test.go                     ← integration
│   ├── handlers/*_test.go                   ← integration (chi + httptest + SQLite)
│   ├── mcptools/*_test.go                   ← integration (chamadas diretas das tools)
│   └── testutil/                            ← helpers compartilhados
│       ├── db.go                            (NewTestDB, NewTestServer, TokenFor)
│       ├── fixtures.go                      (CreateUser, CreateItem, CreateCategory…)
│       └── http.go                          (Request, DecodeJSON)
├── frontend/src/
│   ├── hooks/__tests__/useAuth.test.tsx
│   ├── lib/__tests__/api.test.ts            (interceptors 401/403)
│   ├── lib/__tests__/utils.test.ts
│   ├── components/**/__tests__/*.test.tsx
│   ├── pages/__tests__/*.test.tsx
│   ├── __tests__/App.test.tsx               (guards de rota)
│   └── test/
│       ├── setup.ts                         (jest-dom matchers, polyfills, axios baseURL)
│       └── render.tsx                       (renderWithProviders helper)
└── tests/e2e/
    ├── playwright.config.ts                 (webServer config)
    ├── globalSetup.ts                       (registra admin de teste idempotente)
    ├── helpers/auth.ts                      (loginUI, apiLogin, apiPost…)
    ├── helpers/cleanup.ts                   (fullCleanup entre testes)
    ├── auth.spec.ts                         (3 cenários)
    ├── items.spec.ts                        (3 cenários)
    ├── permissions.spec.ts                  (3 cenários)
    └── mcp.spec.ts                          (3 cenários — smoke do binário via stdio)
```

---

## Cobertura por camada

### Backend Go — 207 testes

| Pacote | Testes | Cobertura |
|--------|--------|-----------|
| `internal/auth` | 10 | **88.2%** |
| `internal/permissions` | 13 | **85.7%** |
| `internal/middleware` | 12 | **100%** |
| `internal/database` (migrate + seed) | 12 | **85.0%** |
| `internal/handlers/auth_handler` | 20 | 84–86% |
| `internal/handlers/user_handler` | 14 | 65–77% |
| `internal/handlers/roles_handler` | 13 | 53–100% |
| `internal/handlers/item_handler` | 19 | 65–82% |
| `internal/handlers/category_handler` | 6 | ~78% |
| `internal/handlers/location_handler` | 6 | ~76% |
| `internal/handlers/extra_handler` | 9 | ~85% |
| `internal/mcptools` (items + categories + locations) | 26 | 79–93% |

**Cobertura média do backend: 76%**

### Frontend — 84 testes

| Arquivo | Testes | Cobertura |
|---------|--------|-----------|
| `hooks/useAuth.tsx` | 14 | **97.8%** |
| `lib/api.ts` (interceptors 401/403) | 9 | **100%** |
| `lib/utils.ts` | 12 | **92.8%** |
| `App.tsx` (guards) | 5 | — |
| `components/layout/Layout.tsx` | 10 | 50% |
| `components/ui/ProfileModal.tsx` | 4 | 47% |
| `pages/Login.tsx` | 6 | **93.3%** |
| `pages/Items.tsx` | 7 | 62% |
| `pages/Permissions.tsx` | 6 | 69% |
| `pages/Users.tsx` | 5 | 49% |
| `pages/Dashboard.tsx` | 3 | 69% |
| `pages/Categories.tsx` | 3 | 28% |

### E2E (Playwright) — 12 cenários

| Spec | Cenários | Cobertura |
|------|----------|-----------|
| `auth.spec.ts` | Register pending → aprovação → login; troca de senha completa | 3 |
| `items.spec.ts` | CRUD via UI; movement registrado ao mudar local | 3 |
| `permissions.spec.ts` | Página carrega; criar perfil custom + atribuir; mudança vale sem relogar | 3 |
| `mcp.spec.ts` | initialize; tools/list retorna 10; find_item_location via stdio | 3 |

---

## Stack de testes

| Camada | Framework | Por quê |
|--------|-----------|---------|
| Go unit | `testing` (stdlib) + `testify/assert` | Idiomático, assertions legíveis |
| Go integration | `net/http/httptest` + SQLite em `t.TempDir()` | Testa o stack real (router + middleware + DB), sem mocks frágeis |
| Frontend unit/component | **Vitest** + **@testing-library/react** + **@testing-library/user-event** | Reusa config do Vite, paralelismo nativo |
| Frontend API mock | **MSW** (Mock Service Worker) | Intercepta `fetch` na rede; testa o `axios` + interceptors real |
| E2E | **Playwright** (Chromium headless) | Multi-browser, fixtures, retries, traces para debug |
| CI | **GitHub Actions** | 3 jobs paralelos: backend, frontend, e2e |

---

## Convenções

### Backend Go

- **Co-localizar** testes: `auth_test.go` ao lado de `auth.go` (padrão da linguagem).
- Para tests que precisam de DB: use `testutil.NewSeededTestDB(t)` — abre SQLite em `t.TempDir()` com migrate + seed prontos. Cleanup automático via `t.Cleanup`.
- Para tests de handlers: use `testutil.NewTestServer(t, db)` — sobe `httptest.Server` com o mesmo `server.BuildRouter` da produção.
- Crie fixtures com `testutil.CreateUser`, `CreateItem`, `CreateCategory`, `CreateLocation`.
- Tokens JWT de teste via `testutil.TokenFor(t, uid, email)` — assinado com `TestJWTSecret` fixo.

### Frontend

- **Pasta `__tests__/`** ao lado do código testado (padrão Vitest/RTL).
- Use `renderWithProviders` de `@/test/render` — envolve QueryClient, MemoryRouter e AuthProvider.
- MSW intercepta `http://localhost/api/*` (a `setup.ts` força `api.defaults.baseURL` para essa URL).
- **NÃO use `vi.mock("react-hot-toast")`** entre arquivos — use `vi.spyOn(toast, "error")` direto. O mock de módulo não é confiável em pool multi-arquivo.
- Para preservar `window.location` entre testes: use `beforeEach`/`afterEach` com `Object.defineProperty(window, "location", ...)` e restore explícito.

### E2E

- Backend de teste roda em **porta 8090** (não conflita com `start-dev.sh`).
- DB efêmero em `/tmp/homeestoque-e2e.db` — globalSetup tenta reaproveitar; só recria se admin não existir.
- Use `apiLogin*` / `apiPost` para setup rápido; reserve UI clicks para o **comportamento sendo testado**.
- Após cada teste de mutação, `fullCleanup()` no `afterEach` apaga itens/usuários/roles customizados E restaura permissões padrão dos roles seed.

---

## CI (GitHub Actions)

Workflow `.github/workflows/ci.yml` roda em todo PR e push pra `main`. Três jobs em paralelo:

| Job | Comandos | Falha em |
|-----|----------|----------|
| `backend` | `go vet`, `go build`, `go test -race -coverprofile` | erro de compilação, suspeita de bug (vet), test fail, race condition |
| `frontend` | `tsc --noEmit`, `vitest run --coverage` | erro de tipo, test fail |
| `e2e` | `playwright install chromium`, `go build mcp`, `npx playwright test` | qualquer cenário falhar |

**Artefatos**: cobertura backend, cobertura frontend, e (em falha) o report HTML do Playwright com traces — ficam 7-14 dias no GitHub.

**Concurrency control** cancela runs anteriores no mesmo branch para economizar minutos.

---

## Adicionando novos testes

### Backend
1. Crie `<package>_test.go` ao lado do código.
2. Use helpers do `testutil` — não duplique setup de DB ou server.
3. Rode `go test -race ./...` antes de commitar.

### Frontend (component)
1. Crie `pages/__tests__/Foo.test.tsx`.
2. Use `renderWithProviders(<Foo />, { route: "/foo" })`.
3. Para chamadas de API, registre handlers MSW dentro do test (não no setup global).

### E2E
1. Crie `tests/e2e/<feature>.spec.ts`.
2. Use `apiLoginAdmin()` + `apiPost` para setup; reserve UI para o teste.
3. Sempre `afterEach(fullCleanup)` em specs com side effects.

---

## Bugs descobertos pelos testes (durante a implementação)

| # | Sprint | Problema | Solução |
|---|--------|----------|---------|
| 1 | 1 | `middleware.go` usava `http.Error()` que forçava `Content-Type: text/plain` — quebrava o toast 403 do frontend | Substituído por `writeJSONError` helper |
| 2 | 4 | `testutil.CreateItem` gerava code baseado em name → UNIQUE constraint fail em loops | Trocado para UUID (igual ao handler real) |

## Side effects positivos

A implementação dos testes também trouxe melhorias na arquitetura:

- **`server.BuildRouter`** extraído de `cmd/api/main.go` — agora reusado por testes de integração e fácil de instrumentar.
- **`vite.config.ts`** aceita `VITE_API_TARGET` env var — necessário para E2E e útil em ambientes de dev paralelos.
- **`_resetRateLimitForTesting`** exposto em `api.ts` para limpar estado entre testes — pequeno custo, isolamento robusto.
