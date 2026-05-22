# API HTTP — Referência

Base URL: `http://localhost:8080`

## Autenticação

Todos os endpoints protegidos requerem:
```
Authorization: Bearer <token>
```

O token é obtido em `POST /api/auth/login`.

## Endpoints

| Módulo | Documento |
|--------|-----------|
| Autenticação | [autenticacao.md](autenticacao.md) |
| Categorias | [categorias.md](categorias.md) |
| Localizações | [localizacoes.md](localizacoes.md) |
| Itens | [itens.md](itens.md) |
| Movimentações | [movimentacoes.md](movimentacoes.md) |
| Dashboard e Exportação | [dashboard.md](dashboard.md) |

## Tabela de rotas

| Método | Rota | Auth | Descrição |
|--------|------|------|-----------|
| GET | `/health` | — | Saúde da API |
| POST | `/api/auth/register` | — | Cadastrar usuário |
| POST | `/api/auth/login` | — | Login → token JWT |
| GET | `/api/auth/me` | ✓ | Dados do usuário logado |
| GET | `/api/items/{id}/qrcode` | — | QR Code do item (PNG) |
| GET | `/api/categories` | ✓ | Listar categorias |
| POST | `/api/categories` | ✓ | Criar categoria |
| PUT | `/api/categories/{id}` | ✓ | Atualizar categoria |
| DELETE | `/api/categories/{id}` | ✓ | Remover categoria |
| GET | `/api/locations` | ✓ | Listar localizações |
| POST | `/api/locations` | ✓ | Criar localização |
| PUT | `/api/locations/{id}` | ✓ | Atualizar localização |
| DELETE | `/api/locations/{id}` | ✓ | Remover localização |
| GET | `/api/items` | ✓ | Listar itens (paginado) |
| POST | `/api/items` | ✓ | Criar item |
| GET | `/api/items/{id}` | ✓ | Detalhe do item |
| PUT | `/api/items/{id}` | ✓ | Atualizar item |
| DELETE | `/api/items/{id}` | ✓ | Remover item |
| GET | `/api/items/{id}/movements` | ✓ | Histórico do item |
| POST | `/api/items/{id}/photos` | ✓ | Upload de foto |
| DELETE | `/api/items/{id}/photos/{photoId}` | ✓ | Remover foto |
| GET | `/api/movements` | ✓ | Todas as movimentações (paginado) |
| GET | `/api/dashboard` | ✓ | Estatísticas gerais |
| GET | `/api/export/csv` | ✓ | Exportar CSV |

## Configurar token no shell

```bash
# Fazer login e salvar token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"seu@email.com","password":"suasenha"}' \
  | jq -r '.token')

# Usar em qualquer requisição
curl -s http://localhost:8080/api/items \
  -H "Authorization: Bearer $TOKEN" | jq
```
