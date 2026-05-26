# HomeEstoque — Documentação

Sistema de inventário doméstico com backend Go, frontend React e servidor MCP para integração com Claude.

## Estrutura da documentação

| Documento | Conteúdo |
|-----------|----------|
| [Arquitetura](arquitetura.md) | Visão geral, banco de dados, sistema de permissões |
| [Desenvolvimento](desenvolvimento.md) | Setup, hot reload, build, scripts utilitários |
| [Instalação em servidor](instalacao.md) | Passo a passo para deploy em VPS (binários, systemd, nginx, HTTPS, backups) |
| [Testes](testes.md) | Como rodar, estrutura, cobertura por camada, CI |
| [API HTTP](api/) | Referência completa dos endpoints REST com exemplos curl |
| [Servidor MCP](mcp/) | Configuração e uso das 10 ferramentas MCP |

## Destaques

- **Sistema de permissões granulares (estilo Discord)** — 15 capacidades nomeadas, perfis customizáveis, mudanças aplicadas em tempo real. Veja [Perfis e Permissões](api/permissoes.md).
- **Gestão de usuários com aprovação** — auto-cadastros entram como `pending` até serem aprovados por um admin. Veja [Usuários](api/usuarios.md).
- **MCP Server** — Claude pode consultar e modificar o estoque por linguagem natural. Veja [MCP](mcp/).

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
