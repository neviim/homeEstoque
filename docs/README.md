# HomeEstoque — Documentação

Sistema de inventário doméstico com backend Go, frontend React e servidor MCP para integração com Claude.

## Estrutura da documentação

| Documento | Conteúdo |
|-----------|----------|
| [Arquitetura](arquitetura.md) | Visão geral, banco de dados, decisões de design |
| [Desenvolvimento](desenvolvimento.md) | Setup, hot reload, build, scripts utilitários |
| [API HTTP](api/) | Referência completa dos endpoints REST com exemplos curl |
| [Servidor MCP](mcp/) | Configuração e uso das 10 ferramentas MCP |

## Início rápido

```bash
# 1. Clonar e configurar
cp backend/.env.example backend/.env
# Editar backend/.env com seus valores

# 2. Subir tudo (backend + frontend com hot reload)
./start-dev.sh

# 3. Acessar
# Frontend:  http://localhost:5173
# API:       http://localhost:8080
# Saúde:     curl http://localhost:8080/health
```

## Stack

- **Backend**: Go 1.25 · chi · SQLite (WAL mode) · JWT
- **Frontend**: React 18 · TypeScript · Vite · TanStack Query · Tailwind CSS
- **MCP**: `github.com/modelcontextprotocol/go-sdk` v1.6.1 · stdio transport

## Portas padrão

| Serviço | Porta |
|---------|-------|
| API HTTP | 8080 |
| Frontend Vite | 5173 |
| MCP | stdio (sem porta) |
