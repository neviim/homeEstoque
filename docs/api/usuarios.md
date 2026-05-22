# API — Usuários

Base URL: `http://localhost:8080/api`

Gestão de usuários do sistema. Todas as rotas exigem a permissão **`users.manage`**.

## Status do usuário

| Valor | Significado |
|-------|-------------|
| `active` | Usuário pode fazer login normalmente |
| `pending` | Aguardando aprovação de um admin (vai para esse status no auto-cadastro) |
| `inactive` | Conta desabilitada; login bloqueado |

O primeiro humano cadastrado vira `admin` + `active` automaticamente. Demais cadastros via `/auth/register` ficam `user` + `pending`.

---

## GET /users

Lista todos os usuários (exceto o usuário sintético do MCP).

**Exemplo**
```bash
curl -s http://localhost:8080/api/users \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "users": [
    {
      "id": 1,
      "name": "Jaime",
      "email": "jaime@test.com",
      "role": "admin",
      "status": "active",
      "created_at": "2026-05-21T13:00:00Z"
    }
  ]
}
```

---

## POST /users

Cria um novo usuário diretamente (já ativo).

**Body**
```json
{
  "name": "Maria",
  "email": "maria@x.com",
  "password": "senha-min-6",
  "role": "user"
}
```

O `role` deve ser o `name` de algum perfil existente (ver [Perfis e Permissões](permissoes.md)). Se omitido, default `user`.

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Maria","email":"maria@x.com","password":"trocar123","role":"user"}'
```

**Erros**
- `400` perfil inexistente · senha < 6 chars · nome/email vazios
- `409` email já cadastrado

---

## PUT /users/{id}

Atualiza nome e role do usuário. **O email é imutável** (não pode ser alterado após cadastro).

**Body**
```json
{
  "name": "Novo nome",
  "role": "auditor"
}
```

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/users/5 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Maria S.","role":"viewer"}'
```

**Regras**
- Você não pode rebaixar a si mesmo se for o último admin ativo (409)
- O role passado precisa existir na tabela `roles` (400 se não existir)

---

## PUT /users/{id}/status

Altera apenas o status do usuário — útil para aprovar/rejeitar cadastros pendentes e desativar contas.

**Body**
```json
{ "status": "active" }
```

Valores aceitos: `active`, `inactive`, `pending`.

**Exemplo — aprovar um cadastro pendente**
```bash
curl -s -X PUT http://localhost:8080/api/users/7/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"active"}'
```

**Regras**
- Você não pode alterar o próprio status (400)
- Inativar o último admin é rejeitado (409)

---

## PUT /users/{id}/password

Redefine a senha de qualquer usuário sem precisar da senha atual (uso administrativo).

**Body**
```json
{ "password": "nova-senha-min-6" }
```

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/users/5/password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"password":"trocar-tudo"}'
```

---

## DELETE /users/{id}

Remove o usuário definitivamente.

**Regras**
- Você não pode excluir a si mesmo (400)
- Não pode excluir o último admin ativo (409)

**Exemplo**
```bash
curl -s -X DELETE http://localhost:8080/api/users/5 \
  -H "Authorization: Bearer $TOKEN"
```

---

## Fluxo: aprovação de novo cadastro

```bash
# 1. Usuário se cadastra pelo /auth/register → entra como 'pending'
curl -X POST http://localhost:8080/api/auth/register \
  -d '{"name":"Ana","email":"ana@x.com","password":"trocar123"}'
# → 201 { "status": "pending", "message": "Conta criada..." }
# (sem token — login não funciona ainda)

# 2. Admin lista pendentes
curl -s http://localhost:8080/api/users \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.users[] | select(.status == "pending")'

# 3. Admin aprova
curl -s -X PUT http://localhost:8080/api/users/7/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"active"}'

# 4. Ana pode fazer login normalmente
curl -X POST http://localhost:8080/api/auth/login \
  -d '{"email":"ana@x.com","password":"trocar123"}'
```
