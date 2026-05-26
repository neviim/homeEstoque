# Desenvolvimento

## Pré-requisitos

| Ferramenta | Versão | Como instalar |
|------------|--------|---------------|
| **Go** | 1.25.0 (fixado via `mise.toml`) | `curl https://mise.run \| sh` → `mise install` na raiz do projeto |
| **Node.js** | 20+ | via nvm, mise ou empacotamento da distro |
| **npm** | 9+ | vem junto com Node.js |
| **Air** | qualquer | `go install github.com/air-verse/air@latest` |

### Instalar mise (gerenciador de versão Go)

```bash
# 1. Instalar mise
curl https://mise.run | sh
echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc  # ou ~/.zshrc
source ~/.bashrc

# 2. Na raiz do projeto, baixar o Go 1.25.0 configurado em mise.toml
cd homeEstoque
mise install
```

A partir daí, ao entrar no diretório do projeto o `go` correto já está disponível no PATH — nenhum caminho absoluto necessário.

---

## Configuração inicial

```bash
# 1. Copiar arquivo de configuração
cp backend/.env.example backend/.env

# 2. Editar as variáveis (mínimo: JWT_SECRET)
# backend/.env:
PORT=8080
DB_PATH=./data/homeestoque.db
JWT_SECRET=troque-por-string-longa-e-aleatoria
UPLOAD_DIR=./uploads
CORS_ORIGINS=http://localhost:5173

# 3. Instalar dependências do frontend
cd frontend && npm install
```

---

## Subir ambiente de desenvolvimento

```bash
# Da raiz do projeto — sobe backend (Air) + frontend (Vite) em paralelo
./start-dev.sh
```

O script usa `Air` para o backend (recompila ao salvar qualquer `.go`) e `Vite` para o frontend (HMR instantâneo).

```
[air]   running...
[vite]  Local: http://localhost:5173/
```

Acesse `http://localhost:5173` e faça login com a conta criada via seed ou registro.

---

## Compilar apenas o backend (sem hot reload)

```bash
cd backend
go build -o bin/api ./cmd/api
./bin/api
```

---

## Compilar o servidor MCP

```bash
./tools/build-mcp.sh
# Gera: bin/homeestoque-mcp
```

---

## Dados de demonstração

```bash
# Adiciona 8 categorias, 10 localizações e 50 itens com todos os campos preenchidos
./tools/seed-demo.sh

# Remove apenas os dados de demo (não afeta seus dados reais)
./tools/seed-demo.sh --delete
```

Os dados demo têm prefixo `[D]` em categorias/localizações e `[DEMO]` no campo `notes` dos itens — o `--delete` usa isso para identificá-los com segurança.

---

## Verificar saúde da API

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"homeestoque-api"}
```

---

## Proxy Vite → API

`frontend/vite.config.ts` redireciona `/api` e `/uploads` para o backend:

```ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
    '/uploads': 'http://localhost:8080',
  }
}
```

Isso permite que o frontend faça `fetch('/api/items')` sem CORS durante o desenvolvimento.

---

## Variáveis de ambiente

| Variável | Default | Descrição |
|----------|---------|-----------|
| `PORT` | `8080` | Porta do servidor HTTP |
| `DB_PATH` | `./data/homeestoque.db` | Caminho do arquivo SQLite |
| `JWT_SECRET` | `dev-secret-change-me` | Segredo para assinar tokens JWT |
| `UPLOAD_DIR` | `./uploads` | Diretório para fotos dos itens |
| `CORS_ORIGINS` | `http://localhost:5173` | Origens permitidas (separadas por vírgula) |

---

## Adicionar dependência Go

```bash
cd backend
go get github.com/exemplo/pacote@v1.0.0
go mod tidy
```

---

## Build de produção

```bash
# Na raiz do projeto
./build.sh              # bump patch (0.1.0 → 0.1.1) + compila tudo
./build.sh --minor      # bump minor (0.1.0 → 0.2.0)
./build.sh --major      # bump major (0.1.0 → 1.0.0)
```

Gera:
- `bin/api` — backend HTTP
- `bin/homeestoque-mcp` — servidor MCP
- `frontend/dist/` — SPA buildada

---

## Release automático

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

O workflow `.github/workflows/release.yml` roda testes, compila multi-plataforma (Linux/macOS/Windows) e publica em GitHub Releases em ~3 minutos. Veja [../README.md](../README.md) para a lista de artefatos.
