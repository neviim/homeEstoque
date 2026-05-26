# HomeEstoque — Documentação

Sistema de inventário doméstico com backend Go, frontend React e servidor MCP para integração com Claude.

## Estrutura da documentação

| Documento | Conteúdo |
|-----------|----------|
| [Arquitetura](arquitetura.md) | Visão geral, banco de dados, sistema de permissões |
| [Desenvolvimento](desenvolvimento.md) | Setup com mise, hot reload, build, scripts utilitários |
| [Instalação em servidor](instalacao.md) | Docker (recomendado: 1 comando) ou bare-metal (systemd, nginx, HTTPS) |
| [Testes](testes.md) | Como rodar, estrutura, cobertura por camada, CI |
| [API HTTP](api/) | Referência completa dos endpoints REST com exemplos curl |
| [Servidor MCP](mcp/) | Configuração e uso das 10 ferramentas MCP |

## Destaques

- **Instalador Docker** — 1 comando instala tudo, HTTPS automático via Caddy. Veja [Instalação](instalacao.md).
- **Sistema de permissões granulares (estilo Discord)** — 15 capacidades nomeadas, perfis customizáveis, mudanças aplicadas em tempo real. Veja [Perfis e Permissões](api/permissoes.md).
- **Gestão de usuários com aprovação** — auto-cadastros entram como `pending` até serem aprovados por um admin. Veja [Usuários](api/usuarios.md).
- **MCP Server** — Claude pode consultar e modificar o estoque por linguagem natural. Veja [MCP](mcp/).
- **313 testes automatizados** — 203 Go + 98 Vitest + 12 E2E Playwright; CI no GitHub Actions.

## Início rápido

### Produção (Docker — recomendado)

```bash
git clone https://github.com/neviim/homeEstoque
cd homeEstoque
./install.sh         # interativo: pergunta domínio, porta, gera JWT_SECRET
```

Abre http://localhost:8080 (ou https://seu-dominio.com com HTTPS automático).

### Desenvolvimento local

```bash
# Pré-requisito: mise instalado (curl https://mise.run | sh)
mise install          # baixa Go 1.25.0 conforme mise.toml

cp backend/.env.example backend/.env
./start-dev.sh        # backend (Air hot-reload) + frontend (Vite HMR)

# Frontend: http://localhost:5173
# API:      http://localhost:8080
```

## Stack

- **Backend**: Go 1.25 · chi · SQLite (WAL mode) · JWT
- **Frontend**: React 18 · TypeScript · Vite · TanStack Query · Tailwind CSS
- **MCP**: `github.com/modelcontextprotocol/go-sdk` · stdio transport
- **Deploy**: Docker Compose · nginx · Caddy (HTTPS) · GoReleaser (releases)

## Portas padrão

| Serviço | Porta |
|---------|-------|
| API HTTP | 8080 |
| Frontend Vite (dev) | 5173 |
| Frontend Docker | 8080 (nginx via Compose) |
| MCP | stdio (sem porta) |
