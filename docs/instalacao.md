# Instalação em servidor

Guia passo a passo para instalar o HomeEstoque em um servidor Linux dedicado (VPS, máquina caseira, Raspberry Pi, etc.).

---

## Caminho recomendado: Docker (1 comando)

Pré-requisito único: **Docker + Docker Compose v2** instalados no servidor.

```bash
git clone https://github.com/neviim/homeEstoque
cd homeEstoque
./install.sh
```

O script interativo:
1. Verifica Docker e Compose no PATH
2. Pergunta se você tem um domínio (para HTTPS automático via Caddy) ou quer rodar em modo local
3. Gera `JWT_SECRET` com `openssl rand -hex 32`
4. Grava `.env` e sobe a stack com `docker compose up -d --build`
5. Aguarda a API ficar saudável e faz smoke tests
6. Exibe a URL e o próximo passo (criar o primeiro usuário admin)

O primeiro usuário registrado vira admin automaticamente — abra a URL e clique em **Criar conta**.

### Comandos úteis após instalação

```bash
./install.sh --update          # atualiza imagens para nova versão do release
./install.sh --down            # para containers (volume de dados preservado)
./install.sh --reset           # para E apaga todos os dados
docker compose logs -f         # logs em tempo real
docker compose logs -f api     # só a API
```

### Instalar Docker no Ubuntu/Debian

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker                  # ou reconecte o SSH
docker compose version         # deve mostrar v2.x
```

### Stack do Docker

| Container | Função |
|-----------|--------|
| `api` | Backend Go — porta interna 8080, monta volume `/data` |
| `web` | nginx — SPA + proxy `/api` e `/uploads` → api |
| `caddy` *(opcional)* | Caddy — HTTPS automático via Let's Encrypt, ativo só com domínio |

Volume único `homeestoque_data` persiste: banco SQLite, fotos e backups.

### HTTPS automático (Caddy)

Quando você fornece um domínio no `install.sh`, o perfil `https` é ativado:

```
internet ─443→ Caddy ─80→ web (nginx) ─http→ api (Go)
```

Requisitos: porta 80 e 443 abertas no firewall; DNS do domínio apontando para o IP do servidor. O Caddy emite e renova o certificado Let's Encrypt automaticamente.

### Atualizando para nova versão

```bash
./install.sh --update
# ou manualmente:
VERSION=v0.2.0 docker compose up -d --build
```

As migrations do banco rodam automaticamente no startup — downtime de ~5 segundos.

---

## Caminho alternativo: deploy em bare-metal (sem Docker)

> Pré-requisito de conhecimento: noções básicas de shell (`ssh`, `sudo`, editar arquivos com `nano`/`vim`).

## Visão geral

| Etapa | O que faz |
|------|----------|
| 1 | Prepara o servidor (usuário, diretórios) |
| 2 | Baixa os binários do GitHub Releases |
| 3 | Cria o arquivo `.env` (senhas, paths) |
| 4 | Sobe a API uma vez para inicializar o banco |
| 5 | Configura `systemd` para rodar como serviço |
| 6 | Configura `nginx` para servir o frontend e proxy da API |
| 7 | HTTPS automático com Let's Encrypt |
| 8 | Cria o primeiro usuário admin |
| 9 | Habilita backups automáticos |

Ao final, você terá:
- API rodando em `127.0.0.1:8080` (apenas localhost)
- Frontend servido pelo nginx em `https://seu-dominio.com/`
- Proxy: `https://seu-dominio.com/api/*` → API
- Banco SQLite com backups automáticos em `/var/lib/homeestoque/data/backups/`

---

## Pré-requisitos do servidor

- Linux x86_64 ou arm64 (Ubuntu 22.04+, Debian 12+, ou similar)
- Acesso `sudo`
- `curl`, `tar`, `nginx`, `systemd` — quase sempre já instalados ou pacotes apt
- Um domínio apontando para o IP do servidor (para HTTPS)

Instale o que faltar:

```bash
sudo apt update
sudo apt install -y curl tar nginx
```

---

## Passo 1 — Preparar o servidor

Crie um usuário dedicado e a estrutura de diretórios:

```bash
sudo useradd --system --shell /usr/sbin/nologin --home /var/lib/homeestoque homeestoque
sudo mkdir -p /opt/homeestoque /var/lib/homeestoque/data /var/lib/homeestoque/uploads /var/www/homeestoque
sudo chown -R homeestoque:homeestoque /var/lib/homeestoque
```

Estrutura final:

```
/opt/homeestoque/            ← binário e arquivos estáticos do app
  homeestoque                  (binário do backend)
/var/lib/homeestoque/        ← dados persistentes (DB, fotos, backups)
  data/
    homeestoque.db
    backups/
  uploads/
  .env
/var/www/homeestoque/        ← frontend (SPA estática)
  index.html
  assets/
```

---

## Passo 2 — Baixar os binários

Identifique sua arquitetura (`amd64` ou `arm64`):

```bash
dpkg --print-architecture
```

Baixe os artefatos do release mais recente. As URLs com `/latest/` apontam sempre para a versão atual:

```bash
# Substitua "linux_amd64" por "linux_arm64" se for ARM
ARCH=linux_amd64

# Backend
curl -L -o /tmp/homeestoque.tar.gz \
  "https://github.com/neviim/homeEstoque/releases/latest/download/homeestoque_${ARCH}.tar.gz"
sudo tar xzf /tmp/homeestoque.tar.gz -C /opt/homeestoque/
sudo chmod +x /opt/homeestoque/homeestoque

# Frontend
curl -L -o /tmp/frontend.tar.gz \
  "https://github.com/neviim/homeEstoque/releases/latest/download/homeestoque_frontend.tar.gz"
sudo tar xzf /tmp/frontend.tar.gz -C /tmp/
sudo cp -r /tmp/frontend/* /var/www/homeestoque/
sudo chown -R www-data:www-data /var/www/homeestoque
```

Confira a versão instalada:

```bash
/opt/homeestoque/homeestoque --help 2>&1 | head -5   # ou rode e olhe /api/version depois
```

---

## Passo 3 — Configurar variáveis de ambiente

Crie `/var/lib/homeestoque/.env`. **O `JWT_SECRET` precisa ser aleatório e longo** — qualquer pessoa que o tenha pode forjar sessões.

```bash
# Gera um segredo seguro de 64 chars hexadecimais
JWT=$(openssl rand -hex 32)

sudo tee /var/lib/homeestoque/.env > /dev/null <<EOF
PORT=8080
DB_PATH=/var/lib/homeestoque/data/homeestoque.db
JWT_SECRET=${JWT}
UPLOAD_DIR=/var/lib/homeestoque/uploads
CORS_ORIGINS=https://seu-dominio.com
EOF

sudo chown homeestoque:homeestoque /var/lib/homeestoque/.env
sudo chmod 600 /var/lib/homeestoque/.env
```

> Substitua `seu-dominio.com` pelo seu domínio real. Se for usar apenas IP em LAN, coloque `http://192.168.x.x` ou `*` (menos seguro).

---

## Passo 4 — Inicializar o banco

Suba o backend manualmente uma vez para criar o `homeestoque.db` com seeds (categorias e locais iniciais):

```bash
sudo -u homeestoque bash -c 'cd /var/lib/homeestoque && set -a && source .env && set +a && /opt/homeestoque/homeestoque' &
sleep 3
curl http://localhost:8080/health   # deve responder "ok"
sudo pkill -f /opt/homeestoque/homeestoque
```

Conferindo:

```bash
ls -lh /var/lib/homeestoque/data/
# Deve aparecer: homeestoque.db
```

---

## Passo 5 — Rodar como serviço systemd

Crie o unit file:

```bash
sudo tee /etc/systemd/system/homeestoque.service > /dev/null <<'EOF'
[Unit]
Description=HomeEstoque API
After=network.target

[Service]
Type=simple
User=homeestoque
Group=homeestoque
WorkingDirectory=/var/lib/homeestoque
EnvironmentFile=/var/lib/homeestoque/.env
ExecStart=/opt/homeestoque/homeestoque
Restart=on-failure
RestartSec=3

# Segurança
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/homeestoque
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now homeestoque
sudo systemctl status homeestoque    # deve mostrar "active (running)"
```

Logs em tempo real:

```bash
sudo journalctl -u homeestoque -f
```

---

## Passo 6 — Configurar nginx

Sirva o frontend e faça proxy da API:

```bash
sudo tee /etc/nginx/sites-available/homeestoque > /dev/null <<'EOF'
server {
    listen 80;
    listen [::]:80;
    server_name seu-dominio.com;        # ← AJUSTE

    # Limite de upload de fotos (padrão nginx é 1MB)
    client_max_body_size 10M;

    # SPA — index.html para qualquer rota
    root /var/www/homeestoque;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # API
    location /api/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }

    # Fotos enviadas pelos itens
    location /uploads/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_set_header   Host $host;
    }

    # Health (opcional, útil pra monitoring)
    location /health {
        proxy_pass http://127.0.0.1:8080;
        access_log off;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/homeestoque /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx
```

Teste sem HTTPS:

```bash
curl -I http://seu-dominio.com/health    # deve retornar 200
```

---

## Passo 7 — HTTPS com Let's Encrypt

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d seu-dominio.com
```

O certbot edita o `homeestoque` do nginx automaticamente, adicionando o bloco `listen 443 ssl` e redirecionando 80 → 443. Renovação é automática via `systemctl status certbot.timer`.

Ajuste o `CORS_ORIGINS` no `.env` para `https://seu-dominio.com` (se ainda estava com `http://`) e reinicie:

```bash
sudo systemctl restart homeestoque
```

---

## Passo 8 — Criar o primeiro usuário admin

O **primeiro usuário registrado** entra automaticamente como `admin` ativo (não precisa de aprovação). Acesse `https://seu-dominio.com/login`, clique em **Criar conta**, preencha nome/email/senha.

A partir do segundo cadastro, novos usuários entram como `pending` e o admin precisa aprovar em **Sistema → Usuários**.

**Esqueceu a senha do admin?** Use o script de reset diretamente no servidor (precisa do binário Go disponível ou usar o utilitário do release):

```bash
# Aborda alternativa rápida sem Go instalado: edite o hash diretamente.
# Mais simples: clone temporário do código e rode o script.
sudo apt install -y git
git clone https://github.com/neviim/homeEstoque /tmp/homeestoque-src
cd /tmp/homeestoque-src

# Precisa do Go — instale via mise (uma vez):
curl https://mise.run | sh
eval "$(~/.local/bin/mise activate bash)"
mise install

# Reseta
DB=/var/lib/homeestoque/data/homeestoque.db
sudo -u homeestoque ./tools/reset-password.sh seu@email.com nova-senha
```

> O `reset-password.sh` lê o DB em `backend/data/homeestoque.db` por padrão. Para apontar para o DB de produção, ajuste o caminho dentro do script ou copie o DB temporariamente.

---

## Passo 9 — Backups automáticos

O sistema tem **backup nativo** acessível em **Sistema → Backup** (após login como admin):
- Dispare backups manualmente com 1 clique
- Agende backups recorrentes (por hora, diário, semanal)
- Restore com confirmação dupla

Os arquivos são salvos em `/var/lib/homeestoque/data/backups/` por padrão.

**Backup adicional fora do servidor** (opcional, recomendado):

```bash
# Crontab do usuário homeestoque
sudo crontab -u homeestoque -e
```

Adicione (exemplo: rsync diário para outro host):

```cron
0 3 * * * rsync -az /var/lib/homeestoque/data/ backup-user@backup-host:/backups/homeestoque/
```

---

## Atualizando para uma nova versão

Quando sair uma nova release:

```bash
ARCH=linux_amd64
sudo systemctl stop homeestoque

# Backend
curl -L -o /tmp/homeestoque.tar.gz \
  "https://github.com/neviim/homeEstoque/releases/latest/download/homeestoque_${ARCH}.tar.gz"
sudo tar xzf /tmp/homeestoque.tar.gz -C /opt/homeestoque/

# Frontend
curl -L -o /tmp/frontend.tar.gz \
  "https://github.com/neviim/homeEstoque/releases/latest/download/homeestoque_frontend.tar.gz"
sudo rm -rf /tmp/frontend && sudo tar xzf /tmp/frontend.tar.gz -C /tmp/
sudo cp -r /tmp/frontend/* /var/www/homeestoque/

sudo systemctl start homeestoque
```

Migrations do banco rodam automaticamente no startup — você só precisa do downtime de ~5 segundos para reiniciar.

Confira:

```bash
curl https://seu-dominio.com/api/version
```

---

## Troubleshooting

| Sintoma | Verifique |
|---------|-----------|
| `502 Bad Gateway` no nginx | `sudo systemctl status homeestoque` — backend caiu. Cheque logs com `journalctl -u homeestoque -n 50` |
| `401 Unauthorized` em todas requisições logadas | `JWT_SECRET` mudou desde o login — usuários precisam re-logar |
| Fotos não aparecem | `/uploads/` precisa estar no proxy do nginx. Cheque `ls /var/lib/homeestoque/uploads/` |
| Login não funciona pós-deploy | `CORS_ORIGINS` no `.env` precisa bater com o domínio do frontend exatamente (`https://` vs `http://` importa) |
| API responde mas frontend não carrega | nginx servindo o `dist/` errado. Confirme `ls /var/www/homeestoque/index.html` |
| Backup falha | Permissão. `chown -R homeestoque:homeestoque /var/lib/homeestoque/data/backups` |

Logs úteis:

```bash
sudo journalctl -u homeestoque -f         # API em tempo real
sudo journalctl -u nginx -f               # nginx
sudo tail -f /var/log/nginx/access.log    # acessos HTTP
sudo tail -f /var/log/nginx/error.log     # erros nginx
```

---

## Checklist final

- [ ] Backend rodando: `sudo systemctl is-active homeestoque` → `active`
- [ ] nginx servindo: `curl -I https://seu-dominio.com` → `200`
- [ ] API respondendo: `curl https://seu-dominio.com/api/version` → JSON com versão
- [ ] Primeiro admin criado via UI
- [ ] HTTPS válido (cadeado verde no navegador)
- [ ] Backup automático agendado em Sistema → Backup
- [ ] (recomendado) Backup off-site via rsync/restic/borgbackup

---

## Próximos passos

- [Arquitetura](arquitetura.md) — entenda como banco, permissões e auth se encaixam
- [API HTTP](api/) — referência dos endpoints para integrações
- [Servidor MCP](mcp/) — habilitar acesso via Claude (mesmo SQLite)
