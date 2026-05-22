# API — Autenticação

Base URL: `http://localhost:8080/api`

Todos os endpoints protegidos exigem o header:
```
Authorization: Bearer <token>
```

O token JWT tem validade de **7 dias** (configurado em `backend/internal/auth/auth.go`). Após expirar, o cliente deve refazer login.

---

## POST /auth/register

Auto-cadastro de novo usuário. **O primeiro usuário humano cadastrado vira `admin` + `active` automaticamente.** Cadastros subsequentes entram como `user` + `pending` — precisam ser aprovados por um admin via [PUT /users/{id}/status](usuarios.md#put-usersidstatus) antes de poderem logar.

**Body**
```json
{
  "name": "Maria Silva",
  "email": "maria@exemplo.com",
  "password": "minhasenha123"
}
```

Validação: senha mínima de 6 caracteres.

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Maria Silva","email":"maria@exemplo.com","password":"minhasenha123"}' | jq
```

**Resposta 201 — primeiro usuário (admin imediato)**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "Maria Silva",
    "email": "maria@exemplo.com",
    "role": "admin",
    "status": "active",
    "created_at": "2026-05-22T10:30:00Z",
    "permissions": ["categories.manage", "categories.view", "..."]
  }
}
```

**Resposta 201 — usuário subsequente (aguarda aprovação)**
```json
{
  "status": "pending",
  "message": "Conta criada com sucesso. Aguardando aprovação de um administrador."
}
```

Note: nesse caso **não vem token** — o usuário não consegue logar até ser aprovado.

---

## POST /auth/login

Autentica e retorna token JWT.

**Body**
```json
{
  "email": "maria@exemplo.com",
  "password": "minhasenha123"
}
```

**Exemplo**
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"maria@exemplo.com","password":"minhasenha123"}' \
  | jq -r '.token')
```

**Resposta 200**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 2,
    "name": "Maria Silva",
    "email": "maria@exemplo.com",
    "role": "user",
    "status": "active",
    "created_at": "2026-05-22T10:30:00Z",
    "permissions": [
      "categories.manage", "categories.view",
      "dashboard.view",
      "export.csv",
      "items.create", "items.delete", "items.update", "items.upload_photo", "items.view",
      "locations.manage", "locations.view",
      "movements.view"
    ]
  }
}
```

**Erros**

| Código | Causa |
|--------|-------|
| 401 | Credenciais inválidas |
| 403 | Conta `pending` (aguardando aprovação) ou `inactive` |

---

## GET /auth/me

Retorna os dados do usuário autenticado com **permissions atualizadas**. O frontend chama esse endpoint no boot e após qualquer mudança de permissão para hidratar o contexto.

**Exemplo**
```bash
curl -s http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Resposta 200**
```json
{
  "id": 2,
  "name": "Maria Silva",
  "email": "maria@exemplo.com",
  "role": "user",
  "status": "active",
  "created_at": "2026-05-22T10:30:00Z",
  "permissions": ["categories.view", "items.view", "..."]
}
```

---

## PUT /auth/profile

Atualiza apenas o **nome** do próprio usuário. O email é imutável após o cadastro.

**Body**
```json
{ "name": "Maria S. Silva" }
```

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/auth/profile \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Maria S. Silva"}' | jq
```

Resposta: mesmo formato de `/auth/me`.

---

## PUT /auth/password

Troca a própria senha. Exige a senha atual para confirmação.

**Body**
```json
{
  "current_password": "minhasenha123",
  "new_password": "novasenha456"
}
```

**Erros**
- `400` nova senha < 6 caracteres
- `401` senha atual incorreta

**Exemplo**
```bash
curl -s -X PUT http://localhost:8080/api/auth/password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"minhasenha123","new_password":"novasenha456"}'
```

---

## Erros comuns

| Código | Causa |
|--------|-------|
| 400 | Body inválido ou campos obrigatórios ausentes |
| 401 | Email/senha incorretos; token ausente, inválido ou expirado |
| 403 | Conta `pending` (no login) ou sem permissão necessária |
| 409 | Email já cadastrado (registro) |
