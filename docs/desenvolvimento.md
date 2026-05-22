# Desenvolvimento

## Pré-requisitos

| Ferramenta | Versão mínima | Localização |
|------------|---------------|-------------|
| Go | 1.25.0 | `/home/neviim/go/bin/go` |
| Node.js | 18+ | `node` |
| npm | 9+ | `npm` |
| Air (hot reload) | qualquer | instalado via `go install` |

> **Atenção**: o binário `go` **não está no PATH** do sistema. Todos os comandos Go devem usar o caminho completo ou as variáveis de ambiente `GOROOT`/`GOPATH`.

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

## Compilar apenas o backend (sem hot reload)

```bash
cd backend
GOROOT=/home/neviim/go \
GOPATH=/home/neviim/go \
GOMODCACHE=/home/neviim/go/pkg/mod \
  /home/neviim/go/bin/go build -o bin/api ./cmd/api
./bin/api
```

## Compilar o servidor MCP

```bash
./tools/build-mcp.sh
# Gera: bin/homeestoque-mcp
```

O script configura automaticamente `GOROOT`, `GOPATH` e `GOMODCACHE`.

## Dados de demonstração

```bash
# Adiciona 8 categorias, 10 localizações e 50 itens com todos os campos preenchidos
./tools/seed-demo.sh

# Remove apenas os dados de demo (não afeta seus dados reais)
./tools/seed-demo.sh --delete
```

Os dados demo têm prefixo `[D]` em categorias/localizações e `[DEMO]` no campo `notes` dos itens — o `--delete` usa isso para identificá-los com segurança.

## Verificar saúde da API

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"homeestoque-api"}
```

## Configuração do Air (hot reload)

`backend/.air.toml` — relevante apenas ao desenvolvimento:

```toml
[build]
  cmd = "GOROOT=/home/neviim/go GOPATH=/home/neviim/go /home/neviim/go/bin/go build -o ./tmp/main ./cmd/api"
  bin = "./tmp/main"
  include_ext = ["go"]
  exclude_dir = ["tmp", "data", "uploads"]
```

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

## Variáveis de ambiente

| Variável | Default | Descrição |
|----------|---------|-----------|
| `PORT` | `8080` | Porta do servidor HTTP |
| `DB_PATH` | `./data/homeestoque.db` | Caminho do arquivo SQLite |
| `JWT_SECRET` | `dev-secret-change-me` | Segredo para assinar tokens JWT |
| `UPLOAD_DIR` | `./uploads` | Diretório para fotos dos itens |
| `CORS_ORIGINS` | `http://localhost:5173` | Origens permitidas (separadas por vírgula) |

## Adicionar dependência Go

```bash
cd backend
GOROOT=/home/neviim/go \
GOPATH=/home/neviim/go \
GOMODCACHE=/home/neviim/go/pkg/mod \
  /home/neviim/go/bin/go get github.com/exemplo/pacote@v1.0.0

# Limpar e atualizar go.sum
GOROOT=/home/neviim/go \
GOPATH=/home/neviim/go \
GOMODCACHE=/home/neviim/go/pkg/mod \
  /home/neviim/go/bin/go mod tidy
```
