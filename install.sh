#!/usr/bin/env bash
# install.sh — Instalador interativo do HomeEstoque via Docker Compose
#
# Uso:
#   ./install.sh              — modo interativo (recomendado)
#   ./install.sh --update     — rebuilda imagens e reinicia containers
#   ./install.sh --down       — para e remove containers (mantém dados)
#   ./install.sh --reset      — para, remove containers E o volume de dados
#
# Variáveis de ambiente (pre-set para CI/automação):
#   DOMAIN, LETSENCRYPT_EMAIL, HTTP_PORT, VERSION, JWT_SECRET, CORS_ORIGINS

set -euo pipefail

# ─── Cores ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; RESET='\033[0m'

info()    { echo -e "${BLUE}ℹ${RESET}  $*"; }
success() { echo -e "${GREEN}✓${RESET}  $*"; }
warn()    { echo -e "${YELLOW}⚠${RESET}  $*"; }
error()   { echo -e "${RED}✗${RESET}  $*" >&2; }
die()     { error "$*"; exit 1; }
header()  { echo -e "\n${BOLD}${BLUE}▶ $*${RESET}"; }

# ─── Diretório do projeto ─────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ENV_FILE="$SCRIPT_DIR/.env"

# ─── Modo especial (--update / --down / --reset) ──────────────────────────────
MODE="${1:-install}"

if [[ "$MODE" == "--update" ]]; then
    header "Atualizando HomeEstoque"
    [[ -f "$ENV_FILE" ]] || die ".env não encontrado. Rode ./install.sh primeiro."
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    COMPOSE_PROFILE=""
    [[ -n "${DOMAIN:-}" ]] && COMPOSE_PROFILE="--profile https"
    docker compose $COMPOSE_PROFILE pull --ignore-pull-failures 2>/dev/null || true
    docker compose $COMPOSE_PROFILE up -d --build
    success "Atualização concluída."
    exit 0
fi

if [[ "$MODE" == "--down" ]]; then
    header "Parando HomeEstoque"
    docker compose --profile https down 2>/dev/null || docker compose down
    success "Containers parados. Dados preservados no volume."
    exit 0
fi

if [[ "$MODE" == "--reset" ]]; then
    header "Reset completo (containers + dados)"
    warn "ATENÇÃO: todos os dados (banco, fotos, backups) serão apagados!"
    read -r -p "  Confirma? (digite 'sim' para continuar): " CONFIRM
    [[ "$CONFIRM" == "sim" ]] || die "Operação cancelada."
    docker compose --profile https down -v 2>/dev/null || docker compose down -v
    success "Reset concluído."
    exit 0
fi

# ─── Verificação de pré-requisitos ────────────────────────────────────────────
header "Verificando pré-requisitos"

check_cmd() {
    command -v "$1" &>/dev/null || die "'$1' não encontrado. Instale Docker: https://docs.docker.com/engine/install/"
}
check_cmd docker

# Docker Compose v2 (plugin)
if ! docker compose version &>/dev/null; then
    die "'docker compose' (v2) não encontrado. Instale o plugin: https://docs.docker.com/compose/install/"
fi

COMPOSE_VERSION=$(docker compose version --short 2>/dev/null || echo "?")
DOCKER_VERSION=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "?")
success "Docker $DOCKER_VERSION  |  Compose $COMPOSE_VERSION"

# openssl para gerar JWT_SECRET
if ! command -v openssl &>/dev/null; then
    warn "'openssl' não encontrado — JWT_SECRET precisará ser gerado manualmente."
    CAN_GEN_JWT=false
else
    CAN_GEN_JWT=true
fi

# ─── Reusar .env existente? ───────────────────────────────────────────────────
header "Configuração"

REUSE_ENV=false
if [[ -f "$ENV_FILE" ]]; then
    echo -e "  ${YELLOW}Arquivo .env existente encontrado.${RESET}"
    read -r -p "  Usar configuração existente? [S/n]: " RESP
    RESP="${RESP:-S}"
    if [[ "${RESP,,}" == "s" || "${RESP,,}" == "y" || -z "$RESP" ]]; then
        REUSE_ENV=true
        # shellcheck disable=SC1090
        source "$ENV_FILE"
        success "Usando .env existente."
    fi
fi

if [[ "$REUSE_ENV" == false ]]; then
    # ── Domínio ──────────────────────────────────────────────────────────────
    echo ""
    echo "  Você tem um domínio público apontando para este servidor?"
    echo "  Deixe em branco para modo local (HTTP, acesso por IP/localhost)."
    read -r -p "  Domínio (ex: estoque.meusite.com) [enter = modo local]: " INPUT_DOMAIN
    DOMAIN="${INPUT_DOMAIN:-${DOMAIN:-}}"

    if [[ -n "$DOMAIN" ]]; then
        # HTTPS mode
        read -r -p "  Email para Let's Encrypt (avisos de renovação): " INPUT_EMAIL
        LETSENCRYPT_EMAIL="${INPUT_EMAIL:-${LETSENCRYPT_EMAIL:-}}"
        [[ -n "$LETSENCRYPT_EMAIL" ]] || die "Email é obrigatório para HTTPS."
        HTTP_PORT=""
        CORS_ORIGINS="https://${DOMAIN}"
    else
        # Local mode
        read -r -p "  Porta HTTP no host [8080]: " INPUT_PORT
        HTTP_PORT="${INPUT_PORT:-${HTTP_PORT:-8080}}"
        LETSENCRYPT_EMAIL=""
        CORS_ORIGINS="http://localhost:${HTTP_PORT}"
    fi

    # ── Versão ────────────────────────────────────────────────────────────────
    read -r -p "  Versão do release [latest]: " INPUT_VERSION
    VERSION="${INPUT_VERSION:-${VERSION:-latest}}"

    # ── JWT_SECRET ────────────────────────────────────────────────────────────
    if [[ "$CAN_GEN_JWT" == true ]]; then
        JWT_SECRET=$(openssl rand -hex 32)
        success "JWT_SECRET gerado automaticamente."
    else
        read -r -p "  JWT_SECRET (string aleatória longa): " JWT_SECRET
        [[ -n "$JWT_SECRET" ]] || die "JWT_SECRET não pode ser vazio."
    fi

    # ── Gravar .env ───────────────────────────────────────────────────────────
    cat > "$ENV_FILE" <<EOF
# Gerado por install.sh em $(date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION=${VERSION}
JWT_SECRET=${JWT_SECRET}
DOMAIN=${DOMAIN}
LETSENCRYPT_EMAIL=${LETSENCRYPT_EMAIL}
HTTP_PORT=${HTTP_PORT:-8080}
CORS_ORIGINS=${CORS_ORIGINS}
EOF
    chmod 600 "$ENV_FILE"
    success ".env criado em $ENV_FILE"
fi

# ─── Garantir variáveis carregadas ────────────────────────────────────────────
# shellcheck disable=SC1090
source "$ENV_FILE"

[[ -n "${JWT_SECRET:-}" ]] || die "JWT_SECRET não está definido no .env."

# ─── Subir a stack ────────────────────────────────────────────────────────────
header "Subindo a stack Docker"

COMPOSE_ARGS=()
if [[ -n "${DOMAIN:-}" ]]; then
    COMPOSE_ARGS+=(--profile https)
    info "Modo HTTPS — Caddy emitirá certificado para ${DOMAIN}"
else
    info "Modo local — acesso em http://localhost:${HTTP_PORT:-8080}"
fi

docker compose "${COMPOSE_ARGS[@]}" up -d --build

# ─── Aguardar healthcheck da API ──────────────────────────────────────────────
header "Aguardando API ficar saudável"

TIMEOUT=90
ELAPSED=0
INTERVAL=5

while true; do
    STATUS=$(docker inspect --format='{{.State.Health.Status}}' \
             "$(docker compose ps -q api 2>/dev/null)" 2>/dev/null || echo "unknown")

    if [[ "$STATUS" == "healthy" ]]; then
        success "API saudável!"
        break
    fi

    if [[ $ELAPSED -ge $TIMEOUT ]]; then
        error "Timeout de ${TIMEOUT}s — API não ficou saudável."
        echo ""
        echo "  Logs da API:"
        docker compose logs --tail=30 api
        die "Verifique os logs acima e tente novamente."
    fi

    echo -ne "  Aguardando... ${ELAPSED}s / ${TIMEOUT}s (status: ${STATUS})  \r"
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

# ─── Smoke tests ──────────────────────────────────────────────────────────────
header "Smoke tests"

# Detectar base URL
if [[ -n "${DOMAIN:-}" ]]; then
    BASE_URL="https://${DOMAIN}"
else
    BASE_URL="http://localhost:${HTTP_PORT:-8080}"
fi

# /health via API container (evita depender de rede pública no smoke)
HEALTH=$(docker compose exec -T api wget -q -O- http://localhost:8080/health 2>/dev/null || echo "FAIL")
if echo "$HEALTH" | grep -q '"status"'; then
    success "/health → OK"
else
    warn "/health retornou: $HEALTH"
fi

# /api/version
VERSION_RESP=$(docker compose exec -T api wget -q -O- http://localhost:8080/api/version 2>/dev/null || echo "FAIL")
if echo "$VERSION_RESP" | grep -q '"version"'; then
    RUNNING_VER=$(echo "$VERSION_RESP" | grep -o '"version":"[^"]*"' | cut -d'"' -f4 || echo "?")
    success "/api/version → ${RUNNING_VER}"
else
    warn "/api/version retornou: $VERSION_RESP"
fi

# ─── Resumo final ─────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${GREEN}════════════════════════════════════════════════════════════${RESET}"
echo -e "${BOLD}${GREEN}  HomeEstoque instalado com sucesso!${RESET}"
echo -e "${BOLD}${GREEN}════════════════════════════════════════════════════════════${RESET}"
echo ""
echo -e "  ${BOLD}URL:${RESET}   ${BASE_URL}"
echo ""
echo -e "  ${BOLD}Próximos passos:${RESET}"
echo -e "  1. Abra ${BASE_URL} no navegador"
echo -e "  2. Clique em \"Criar conta\" — o primeiro usuário vira admin automaticamente"
echo -e "  3. Configure backup automático em Sistema → Backup"
echo ""
echo -e "  ${BOLD}Comandos úteis:${RESET}"
echo -e "  ./install.sh --update   Atualiza para nova versão"
echo -e "  ./install.sh --down     Para os containers (mantém dados)"
echo -e "  ./install.sh --reset    Remove tudo incluindo dados"
echo -e "  docker compose logs -f  Logs em tempo real"
echo ""
