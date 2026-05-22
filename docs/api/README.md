# API HTTP — Referência

Base URL: `http://localhost:8080`

## Autenticação

Todos os endpoints protegidos requerem:
```
Authorization: Bearer <token>
```

O token é obtido em `POST /api/auth/login`.

## Modelo de autorização

Cada endpoint é protegido por uma **permissão nomeada** (ex: `items.create`, `users.manage`). O usuário tem acesso se seu **perfil (role)** inclui essa permissão. A verificação consulta o banco em **todo request** — alterações de permissão valem imediatamente, sem precisar relogar.

Veja [Perfis e Permissões](permissoes.md) para o catálogo completo das 15 permissões e a API de gerenciamento.

## Endpoints

| Módulo | Documento |
|--------|-----------|
| Autenticação | [autenticacao.md](autenticacao.md) |
| Usuários | [usuarios.md](usuarios.md) |
| Perfis e Permissões | [permissoes.md](permissoes.md) |
| Categorias | [categorias.md](categorias.md) |
| Localizações | [localizacoes.md](localizacoes.md) |
| Itens | [itens.md](itens.md) |
| Movimentações | [movimentacoes.md](movimentacoes.md) |
| Dashboard e Exportação | [dashboard.md](dashboard.md) |

## Tabela de rotas

| Método | Rota | Permissão | Descrição |
|--------|------|-----------|-----------|
| GET | `/health` | — | Saúde da API |
| POST | `/api/auth/register` | — | Auto-cadastro (entra como `pending`) |
| POST | `/api/auth/login` | — | Login → token JWT |
| GET | `/api/items/{id}/qrcode` | — | QR Code do item (PNG) |
| GET | `/api/auth/me` | autenticado | Dados do usuário logado + permissions |
| PUT | `/api/auth/profile` | autenticado | Atualizar próprio nome |
| PUT | `/api/auth/password` | autenticado | Trocar a própria senha |
| GET | `/api/permissions` | autenticado | Catálogo de permissões |
| GET | `/api/roles` | autenticado | Listar perfis e suas permissões |
| GET | `/api/dashboard` | `dashboard.view` | Estatísticas gerais |
| GET | `/api/categories` | `categories.view` | Listar categorias |
| POST/PUT/DELETE | `/api/categories` | `categories.manage` | Gerenciar categorias |
| GET | `/api/locations` | `locations.view` | Listar localizações |
| POST/PUT/DELETE | `/api/locations` | `locations.manage` | Gerenciar localizações |
| GET | `/api/items`, `/api/items/{id}`, `/api/items/{id}/movements` | `items.view` | Listar / detalhe / histórico |
| POST | `/api/items` | `items.create` | Criar item |
| PUT | `/api/items/{id}` | `items.update` | Atualizar item |
| DELETE | `/api/items/{id}` | `items.delete` | Remover item |
| POST/DELETE | `/api/items/{id}/photos/...` | `items.upload_photo` | Anexar/remover foto |
| GET | `/api/movements`, `/api/movements/users` | `movements.view` | Histórico geral |
| GET | `/api/export/csv` | `export.csv` | Exportar CSV |
| GET/POST/PUT/DELETE | `/api/users/...` | `users.manage` | Gerenciar usuários |
| POST/PUT/DELETE | `/api/roles/...` | `roles.manage` | Gerenciar perfis |

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

## Códigos de resposta padrão

| Código | Significado |
|--------|-------------|
| 200 / 201 | OK / Criado |
| 400 | Body inválido, validação falhou |
| 401 | Sem token / token inválido |
| 403 | Sem permissão para essa ação (perfil não inclui a permissão necessária) |
| 404 | Recurso não encontrado |
| 409 | Conflito (email duplicado, último admin, perfil em uso) |
| 500 | Erro interno |
