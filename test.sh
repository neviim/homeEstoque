#!/usr/bin/env bash
# Roda toda a suíte de testes do HomeEstoque (backend Go + frontend Vitest + E2E Playwright).
#
# Uso:
#   ./test.sh              # roda tudo (output verboso de cada camada)
#   ./test.sh backend      # só backend Go
#   ./test.sh frontend     # só Vitest
#   ./test.sh e2e          # só Playwright (compila MCP se necessário)
#   ./test.sh --coverage   # roda tudo com relatórios de cobertura
#   ./test.sh --fast       # pula E2E (~2x mais rápido)
#   ./test.sh --quiet      # output só do resumo; em caso de falha imprime o detalhe da camada
#   ./test.sh -q           # alias curto de --quiet
#
# Códigos de saída:
#   0 — todos passaram
#   1 — alguma camada falhou (continua executando as próximas para ver tudo)

set -uo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"

# ─── Cores ──────────────────────────────────────────────────────────────
if [[ -t 1 ]]; then
  RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BLUE=$'\033[34m'
  BOLD=$'\033[1m'; RESET=$'\033[0m'
else
  RED=""; GREEN=""; YELLOW=""; BLUE=""; BOLD=""; RESET=""
fi

# ─── Flags ──────────────────────────────────────────────────────────────
TARGET="all"
COVERAGE=0
FAST=0
QUIET=0
for arg in "$@"; do
  case "$arg" in
    backend|frontend|e2e|all) TARGET="$arg" ;;
    --coverage)               COVERAGE=1 ;;
    --fast)                   FAST=1 ;;
    -q|--quiet)               QUIET=1 ;;
    -h|--help)
      # Imprime o cabeçalho até a primeira linha não-comentário
      awk 'NR>1 && /^#/ { sub(/^# ?/, ""); print; next } NR>1 { exit }' "$0"
      exit 0
      ;;
    *)
      echo "${RED}Argumento desconhecido: $arg${RESET}" >&2
      echo "Uso: $0 [backend|frontend|e2e|all] [--coverage] [--fast] [--quiet]" >&2
      exit 2
      ;;
  esac
done

# ─── Pré-requisitos ─────────────────────────────────────────────────────
GO_BIN="/home/neviim/go/bin/go"
NODE_BIN="/home/neviim/.nvm/versions/node/v24.13.0/bin/node"
NPM_BIN="/home/neviim/.nvm/versions/node/v24.13.0/bin/npm"
NPX_BIN="/home/neviim/.nvm/versions/node/v24.13.0/bin/npx"

if [[ "$TARGET" =~ ^(all|backend|e2e)$ ]] && [[ ! -x "$GO_BIN" ]]; then
  echo "${RED}go não encontrado em $GO_BIN${RESET}" >&2
  exit 3
fi
if [[ "$TARGET" =~ ^(all|frontend|e2e)$ ]] && [[ ! -x "$NPM_BIN" ]]; then
  echo "${RED}npm não encontrado em $NPM_BIN${RESET}" >&2
  exit 3
fi

# ─── Helpers ────────────────────────────────────────────────────────────
declare -A RESULT=()
declare -A DURATION=()
declare -A TEST_COUNT=()
TOTAL_TESTS=0

section() {
  echo
  echo "${BOLD}${BLUE}━━━ $1 ━━━${RESET}"
}

# Conta testes a partir do log capturado. Cada framework imprime de forma
# diferente — extraímos heuristicamente. Retorna número via stdout (0 se
# não conseguiu identificar — frameworks são tolerantes).
count_tests() {
  local logfile="$1" framework="$2" n=0
  case "$framework" in
    go)
      # `go test -v` emite uma linha "--- PASS: TestX" por teste passado e
      # "--- FAIL: TestX" por falho. Contamos os dois.
      n=$(grep -cE '^\s*--- (PASS|FAIL):' "$logfile" 2>/dev/null || echo 0)
      ;;
    vitest)
      # Vitest imprime no final algo como: "Tests  84 passed (84)" — o
      # segundo número é o total. Pegamos a última ocorrência.
      n=$(grep -oE 'Tests[[:space:]]+[0-9]+( failed \| )?[0-9]* ?passed[[:space:]]*\(([0-9]+)\)' "$logfile" \
          | tail -1 | grep -oE '\([0-9]+\)$' | tr -d '()' || true)
      [[ -z "$n" ]] && n=0
      ;;
    playwright)
      # Playwright imprime "  N passed (Ts)" no final. Pegamos o N da
      # primeira linha que casar.
      n=$(grep -oE '[0-9]+ passed' "$logfile" | head -1 | grep -oE '[0-9]+' || true)
      [[ -z "$n" ]] && n=0
      ;;
  esac
  echo "$n"
}

run_step() {
  local name="$1" framework="$2"
  shift 2
  local start status logfile count=0
  start=$(date +%s)
  logfile=$(mktemp)

  if [[ $QUIET -eq 1 ]]; then
    # Modo silencioso: captura output em arquivo temp.
    printf "  %s%s%s ... " "${BOLD}" "$name" "${RESET}"
    if "$@" >"$logfile" 2>&1; then
      status=0
      count=$(count_tests "$logfile" "$framework")
      RESULT["$name"]="${GREEN}PASS${RESET}"
      echo "${GREEN}✓${RESET}  $count testes"
    else
      status=$?
      count=$(count_tests "$logfile" "$framework")
      RESULT["$name"]="${RED}FAIL${RESET}"
      echo "${RED}✗${RESET}"
      echo
      echo "${RED}${BOLD}━━━ Output de '$name' (falhou) ━━━${RESET}"
      cat "$logfile"
      echo
    fi
  else
    section "$name"
    # Mesmo no modo verboso, vamos capturar em paralelo (via tee) pra extrair
    # o contador. tee duplica para stdout do user e para o arquivo.
    if "$@" 2>&1 | tee "$logfile"; then
      status=0
      RESULT["$name"]="${GREEN}PASS${RESET}"
    else
      status=${PIPESTATUS[0]}
      RESULT["$name"]="${RED}FAIL${RESET}"
    fi
    count=$(count_tests "$logfile" "$framework")
  fi

  TEST_COUNT["$name"]=$count
  TOTAL_TESTS=$((TOTAL_TESTS + count))
  DURATION["$name"]="$(( $(date +%s) - start ))s"
  rm -f "$logfile"
}

# ─── Backend ────────────────────────────────────────────────────────────
run_backend() {
  cd "$ROOT/backend"
  # -v é necessário para contar testes individuais (--- PASS:/FAIL:);
  # no modo verboso polui um pouco mas no quiet o log fica em temp.
  local args=(-race -v)
  if [[ $COVERAGE -eq 1 ]]; then
    args+=(-coverprofile=coverage.out)
  fi
  GOROOT=/home/neviim/go GOPATH=/home/neviim/go GOMODCACHE=/home/neviim/go/pkg/mod \
    "$GO_BIN" test "${args[@]}" ./...
  local rc=$?
  if [[ $COVERAGE -eq 1 && $rc -eq 0 ]]; then
    echo
    echo "${YELLOW}Cobertura backend:${RESET}"
    GOROOT=/home/neviim/go "$GO_BIN" tool cover -func=coverage.out | tail -1
    echo "Relatório HTML: backend/coverage.out (rode 'go tool cover -html=coverage.out')"
  fi
  return $rc
}

# ─── Frontend ───────────────────────────────────────────────────────────
run_frontend() {
  cd "$ROOT/frontend"
  if [[ ! -d node_modules ]]; then
    echo "${YELLOW}node_modules não encontrado em frontend/ — rodando npm ci${RESET}"
    "$NPM_BIN" ci || return $?
  fi
  if [[ $COVERAGE -eq 1 ]]; then
    "$NPM_BIN" run test:coverage
  else
    "$NPM_BIN" test
  fi
}

# ─── E2E ────────────────────────────────────────────────────────────────
run_e2e() {
  # 1. Compila o binário MCP (necessário para mcp.spec.ts)
  if [[ ! -x "$ROOT/bin/homeestoque-mcp" ]]; then
    section "E2E pre-step: build MCP binary"
    "$ROOT/tools/build-mcp.sh" || return $?
  fi

  cd "$ROOT/tests/e2e"
  if [[ ! -d node_modules ]]; then
    echo "${YELLOW}node_modules não encontrado em tests/e2e/ — rodando npm ci${RESET}"
    "$NPM_BIN" ci || return $?
  fi
  if [[ ! -d .playwright-cache ]]; then
    echo "${YELLOW}Browser do Playwright não instalado — instalando Chromium${RESET}"
    PLAYWRIGHT_BROWSERS_PATH="$ROOT/tests/e2e/.playwright-cache" \
      "$NPX_BIN" playwright install chromium || return $?
  fi
  PLAYWRIGHT_BROWSERS_PATH="$ROOT/tests/e2e/.playwright-cache" \
    "$NPX_BIN" playwright test
}

# ─── Orquestração ────────────────────────────────────────────────────────
echo "${BOLD}HomeEstoque — suíte de testes${RESET}"
flags=""
[[ $COVERAGE -eq 1 ]] && flags="$flags +coverage"
[[ $FAST     -eq 1 ]] && flags="$flags +fast"
[[ $QUIET    -eq 1 ]] && flags="$flags +quiet"
echo "Modo: $TARGET$flags"

case "$TARGET" in
  backend)  run_step "Backend Go"       go         run_backend ;;
  frontend) run_step "Frontend Vitest"  vitest     run_frontend ;;
  e2e)      run_step "E2E Playwright"   playwright run_e2e ;;
  all)
    run_step "Backend Go"       go         run_backend
    run_step "Frontend Vitest"  vitest     run_frontend
    if [[ $FAST -eq 0 ]]; then
      run_step "E2E Playwright" playwright run_e2e
    fi
    ;;
esac

# ─── Resumo final ────────────────────────────────────────────────────────
echo
echo "${BOLD}━━━ Resumo ━━━${RESET}"
fail_count=0
# Itera na ordem original (Backend → Frontend → E2E) para output consistente
for name in "Backend Go" "Frontend Vitest" "E2E Playwright"; do
  [[ -z "${RESULT[$name]:-}" ]] && continue
  printf "  %-25s %s  %4d testes  (%s)\n" \
    "$name" "${RESULT[$name]}" "${TEST_COUNT[$name]:-0}" "${DURATION[$name]}"
  [[ "${RESULT[$name]}" == *FAIL* ]] && ((fail_count++))
done
echo "  ${BOLD}Total: $TOTAL_TESTS testes${RESET}"

if [[ $fail_count -gt 0 ]]; then
  echo
  echo "${RED}${BOLD}$fail_count camada(s) falharam${RESET}"
  exit 1
fi

echo
echo "${GREEN}${BOLD}Todos os testes passaram ✓${RESET}"
