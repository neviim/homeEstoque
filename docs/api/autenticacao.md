# API — Autenticação

Base URL: `http://localhost:8080/api`

Todos os endpoints protegidos exigem o header:
```
Authorization: Bearer <token>
```

O token é obtido no login e não tem expiração configurada na implementação atual.

---

## POST /auth/register

Cria uma nova conta de usuário.

**Body**
```json
{
  "name": "Maria Silva",
  "email": "maria@exemplo.com",
  "password": "minhasenha123"
}
```

**Exemplo**
```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Maria Silva","email":"maria@exemplo.com","password":"minhasenha123"}' | jq
```

**Resposta 201**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 2,
    "name": "Maria Silva",
    "email": "maria@exemplo.com",
    "created_at": "2026-05-22T10:30:00Z"
  }
}
```

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

echo $TOKEN
```

**Resposta 200**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 2,
    "name": "Maria Silva",
    "email": "maria@exemplo.com",
    "created_at": "2026-05-22T10:30:00Z"
  }
}
```

---

## GET /auth/me

Retorna os dados do usuário autenticado.

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
  "created_at": "2026-05-22T10:30:00Z"
}
```

---

## Erros comuns

| Código | Causa |
|--------|-------|
| 400 | Body inválido ou campos obrigatórios ausentes |
| 401 | Email/senha incorretos; token ausente ou inválido |
| 409 | Email já cadastrado (registro) |
