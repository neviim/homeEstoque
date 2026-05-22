# MCP — Configuração do Servidor

O servidor MCP permite que o Claude (e outros clientes MCP) interajam com o inventário HomeEstoque diretamente em conversa natural — sem precisar abrir a UI web.

## Como funciona

```
Claude Code / Claude Desktop
        │
        │ spawna como subprocess
        ▼
bin/homeestoque-mcp  (stdio JSON-RPC)
        │
        │ lê/escreve SQLite
        ▼
backend/data/homeestoque.db  (compartilhado com a API HTTP via WAL)
```

O binário roda como subprocess do client MCP. Toda comunicação é via `stdin`/`stdout` usando JSON-RPC 2.0 (protocolo MCP). Logs vão para `stderr` e não interferem no protocolo.

## Compilar o binário

```bash
./tools/build-mcp.sh
# Saída: bin/homeestoque-mcp  (~14 MB)
```

## Configurar no Claude Code

```bash
# Adicionar o servidor (escopo local — só este projeto)
claude mcp add homeestoque \
  --scope local \
  -e DB_PATH=/home/neviim/developer/homeEstoque/backend/data/homeestoque.db \
  -- /home/neviim/developer/homeEstoque/bin/homeestoque-mcp

# Verificar
claude mcp list
```

A configuração é salva em `.claude/settings.local.json`:

```json
{
  "mcpServers": {
    "homeestoque": {
      "command": "/home/neviim/developer/homeEstoque/bin/homeestoque-mcp",
      "args": [],
      "env": {
        "DB_PATH": "/home/neviim/developer/homeEstoque/backend/data/homeestoque.db"
      }
    }
  }
}
```

## Configurar no Claude Desktop

Adicione em `~/.config/claude/claude_desktop_config.json` (Linux) ou equivalente no macOS:

```json
{
  "mcpServers": {
    "homeestoque": {
      "command": "/home/neviim/developer/homeEstoque/bin/homeestoque-mcp",
      "env": {
        "DB_PATH": "/home/neviim/developer/homeEstoque/backend/data/homeestoque.db"
      }
    }
  }
}
```

Reinicie o Claude Desktop após salvar.

## Variáveis de ambiente

| Variável | Obrigatória | Descrição |
|----------|-------------|-----------|
| `DB_PATH` | ✓ | Caminho absoluto para o SQLite |
| `JWT_SECRET` | — | Não usado pelo MCP |

O servidor usa `config.Load()` — mesma função da API HTTP — então todas as variáveis do `.env` são lidas automaticamente se o binário for executado na pasta `backend/`.

## Smoke test

Testar o servidor diretamente via stdio:

```bash
# Inicializar sessão
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}' \
  | DB_PATH=/home/neviim/developer/homeEstoque/backend/data/homeestoque.db \
    /home/neviim/developer/homeEstoque/bin/homeestoque-mcp 2>/dev/null \
  | jq '.result.serverInfo'
```

Resposta esperada:
```json
{
  "name": "homeestoque",
  "version": "0.1.0"
}
```

## Usuário MCP Assistant

O seed cria automaticamente um usuário especial `mcp@homeestoque.local` com o nome "MCP Assistant". Esse usuário:

- Nunca consegue fazer login via API HTTP (hash `!disabled!`)
- É o autor de todas as movimentações criadas pelo servidor MCP
- Aparece na coluna "Usuário" da página de Movimentações da UI

Isso permite distinguir movimentos feitos pelo Claude dos movimentos feitos por humanos na interface web.
