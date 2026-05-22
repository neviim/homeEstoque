# API — Perfis e Permissões

Base URL: `http://localhost:8080/api`

Sistema de permissões granulares estilo Discord: cada perfil tem um conjunto de **capacidades** (permissões nomeadas) que o administrador pode ativar/desativar. As mudanças passam a valer no **próximo request** — sem precisar relogar.

## Conceitos

- **Perfil (role)**: grupo de permissões; cada usuário tem exatamente um perfil
- **Permissão (permission)**: capacidade nomeada (ex: `items.create`, `categories.manage`)
- **Catálogo**: lista fixa de 15 permissões definidas em `backend/internal/permissions/catalog.go`
- **Perfis semente**: `admin`, `user`, `viewer` — criados automaticamente no primeiro startup
- **Perfil de sistema**: marcado com `is_system: true` — o perfil `admin` é o único e não pode ser excluído nem renomeado, e sempre tem todas as permissões

## Catálogo completo de permissões

| Categoria | Key | Label |
|-----------|-----|-------|
| Visualização | `dashboard.view` | Ver Dashboard |
| Visualização | `dashboard.view_value` | Ver valor patrimonial |
| Itens | `items.view` | Ver Itens |
| Itens | `items.create` | Criar Itens |
| Itens | `items.update` | Editar Itens |
| Itens | `items.delete` | Excluir Itens |
| Itens | `items.upload_photo` | Anexar Fotos |
| Categorias | `categories.view` | Ver Categorias |
| Categorias | `categories.manage` | Gerenciar Categorias |
| Locais | `locations.view` | Ver Locais |
| Locais | `locations.manage` | Gerenciar Locais |
| Movimentações | `movements.view` | Ver Movimentações |
| Exportação | `export.csv` | Exportar CSV |
| Sistema | `users.manage` | Gerenciar Usuários |
| Sistema | `roles.manage` | Gerenciar Perfis |

## Permissões padrão dos perfis semente

| Perfil | Permissões |
|--------|------------|
| `admin` | **todas as 15** (sempre, automaticamente) |
| `user` | tudo exceto `users.manage`, `roles.manage`, `dashboard.view_value` (total: 12) |
| `viewer` | apenas `dashboard.view` e `items.view` (total: 2) |

---

## GET /permissions

Retorna o catálogo completo de permissões — usado pela UI para montar a tela de configuração.

**Permissão necessária:** apenas autenticado

**Exemplo**
```bash
curl -s http://localhost:8080/api/permissions \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "permissions": [
    {
      "key": "dashboard.view",
      "label": "Ver Dashboard",
      "description": "Acessar a página inicial do sistema",
      "category": "Visualização"
    },
    {
      "key": "items.create",
      "label": "Criar Itens",
      "description": "Cadastrar novos itens",
      "category": "Itens"
    }
  ]
}
```

---

## GET /roles

Lista todos os perfis com suas permissões atuais e contagem de usuários atribuídos.

**Permissão necessária:** apenas autenticado

**Exemplo**
```bash
curl -s http://localhost:8080/api/roles \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "roles": [
    {
      "id": 1,
      "name": "admin",
      "label": "Administrador",
      "description": "Acesso total ao sistema. Não pode ser editado nem excluído.",
      "is_system": true,
      "user_count": 2,
      "permissions": ["categories.manage", "categories.view", "..."],
      "created_at": "2026-05-22T10:00:00Z"
    },
    {
      "id": 2,
      "name": "user",
      "label": "Usuário",
      "description": "Pode gerenciar itens, categorias, locais e movimentações.",
      "is_system": false,
      "user_count": 1,
      "permissions": ["categories.view", "categories.manage", "..."],
      "created_at": "2026-05-22T10:00:00Z"
    }
  ]
}
```

---

## POST /roles

Cria um novo perfil customizado (sem permissões inicialmente — defina depois via `PUT /roles/{id}/permissions`).

**Permissão necessária:** `roles.manage`

**Body**
```json
{
  "name": "auditor",
  "label": "Auditor",
  "description": "Lê tudo, não escreve nada"
}
```

| Campo | Validação |
|-------|-----------|
| `name` | obrigatório · slug em snake_case · `^[a-z][a-z0-9_]{1,49}$` · único |
| `label` | obrigatório · texto livre exibido na UI |
| `description` | opcional |

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"auditor","label":"Auditor","description":"Somente leitura"}' | jq
```

**Resposta 201**
```json
{
  "id": 4,
  "name": "auditor",
  "label": "Auditor",
  "description": "Somente leitura",
  "is_system": false,
  "user_count": 0,
  "permissions": [],
  "created_at": "2026-05-22T20:07:36Z"
}
```

---

## PUT /roles/{id}

Atualiza `label`, `description` e (para perfis não-system) o `name`. Ao renomear o `name`, todos os usuários atribuídos têm seu `users.role` atualizado na mesma transação.

**Permissão necessária:** `roles.manage`

**Body**
```json
{
  "name": "novo_slug",
  "label": "Novo Nome",
  "description": "..."
}
```

Para perfis de sistema (`admin`), o campo `name` é ignorado.

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/roles/4 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"auditor_v2","label":"Auditor (v2)","description":"Atualizado"}' | jq
```

---

## DELETE /roles/{id}

Exclui um perfil customizado. **Rejeita** se:
- O perfil tem `is_system: true` → 403
- Há usuários atribuídos ao perfil → 409 (reatribua-os antes)

**Permissão necessária:** `roles.manage`

**Exemplo**
```bash
curl -s -X DELETE http://localhost:8080/api/roles/4 \
  -H "Authorization: Bearer $TOKEN"
```

---

## PUT /roles/{id}/permissions

**Substitui completamente** o conjunto de permissões do perfil. Para o perfil `admin` (sistema), o servidor **ignora o body** e força a inclusão de todas as 15 permissões do catálogo.

**Permissão necessária:** `roles.manage`

**Body**
```json
{
  "permissions": [
    "dashboard.view",
    "items.view",
    "categories.view",
    "locations.view",
    "movements.view"
  ]
}
```

Cada key deve existir no catálogo (`/permissions`). Keys desconhecidas → 400.

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/roles/4/permissions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"permissions":["dashboard.view","items.view","export.csv"]}' | jq
```

**Resposta 200** — retorna o perfil atualizado com a nova lista de permissões.

---

## Erros comuns

| Código | Causa |
|--------|-------|
| 400 | `name` em formato inválido; permissão desconhecida no body |
| 401 | Sem token |
| 403 | Sem permissão `roles.manage`; ou tentando excluir/renomear perfil de sistema |
| 404 | Perfil não encontrado |
| 409 | Nome de perfil duplicado; ou exclusão com usuários atribuídos |
