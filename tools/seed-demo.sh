#!/usr/bin/env bash
# Popula (ou limpa) dados de demonstração no HomeEstoque.
# Uso:
#   ./tools/seed-demo.sh           → adiciona ~50 itens de demo
#   ./tools/seed-demo.sh --delete  → remove todos os itens/categorias/locais de demo

set -euo pipefail

API="http://localhost:8080/api"
DEMO_EMAIL="demo@homeestoque.dev"
DEMO_PASS="demo123456"
DEMO_NAME="Usuário Demo"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✔${NC} $1"; }
info() { echo -e "${CYAN}→${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
fail() { echo -e "${RED}✘ ERRO:${NC} $1"; exit 1; }

need_cmd() { command -v "$1" &>/dev/null || fail "comando '$1' não encontrado. Instale-o e tente novamente."; }
need_cmd curl
need_cmd jq

# ─── autenticação ──────────────────────────────────────────────────────────────

get_token() {
  local res
  res=$(curl -sf -X POST "$API/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$DEMO_EMAIL\",\"password\":\"$DEMO_PASS\"}" 2>/dev/null || true)

  local tok
  tok=$(echo "$res" | jq -r '.token // empty' 2>/dev/null)

  if [[ -z "$tok" ]]; then
    info "Usuário demo não existe — criando..."
    res=$(curl -sf -X POST "$API/auth/register" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"$DEMO_NAME\",\"email\":\"$DEMO_EMAIL\",\"password\":\"$DEMO_PASS\"}")
    tok=$(echo "$res" | jq -r '.token // empty')
    [[ -n "$tok" ]] || fail "Não foi possível criar/autenticar o usuário demo."
    ok "Usuário demo criado"
  fi
  echo "$tok"
}

post()   { curl -sf -X POST   "$API/$1" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2"; }
delete() { curl -sf -X DELETE "$API/$1" -H "Authorization: Bearer $TOKEN"; }
get()    { curl -sf -X GET    "$API/$1" -H "Authorization: Bearer $TOKEN"; }

# ─── modo delete ───────────────────────────────────────────────────────────────

if [[ "${1:-}" == "--delete" ]]; then
  echo -e "\n${RED}━━━ Removendo dados de demonstração ━━━${NC}\n"
  TOKEN=$(get_token)

  info "Buscando itens de demo..."
  ITEMS=$(get "items?limit=200" | jq -r '.items[]? | select(.notes | test("\\[DEMO\\]")) | .id')
  COUNT=0
  for id in $ITEMS; do
    delete "items/$id" &>/dev/null || true
    COUNT=$((COUNT + 1))
  done
  ok "Itens removidos: $COUNT"

  info "Buscando categorias de demo..."
  CATS=$(get "categories" | jq -r '.[]? | select(.name | test("^\\[D\\]")) | .id')
  for id in $CATS; do
    delete "categories/$id" &>/dev/null || true
  done
  ok "Categorias removidas"

  info "Buscando locais de demo..."
  LOCS=$(get "locations" | jq -r '.[]? | select(.name | test("^\\[D\\]")) | .id')
  for id in $LOCS; do
    delete "locations/$id" &>/dev/null || true
  done
  ok "Locais removidos"

  echo -e "\n${GREEN}Dados de demo removidos com sucesso.${NC}\n"
  exit 0
fi

# ─── seed ──────────────────────────────────────────────────────────────────────

echo -e "\n${CYAN}━━━ HomeEstoque — Seed de Demonstração ━━━${NC}\n"
TOKEN=$(get_token)
ok "Autenticado como $DEMO_EMAIL"

# ── categorias ────────────────────────────────────────────────────────────────
echo ""
info "Criando categorias..."

declare -A CAT
create_cat() {
  local name="$1" icon="$2" color="$3"
  local res
  res=$(post "categories" "{\"name\":\"[D] $name\",\"icon\":\"$icon\",\"color\":\"$color\"}")
  CAT["$name"]=$(echo "$res" | jq -r '.id')
}

create_cat "Eletrônicos"      "Cpu"          "#6366f1"
create_cat "Eletrodomésticos" "Zap"          "#f59e0b"
create_cat "Ferramentas"      "Wrench"       "#78716c"
create_cat "Cozinha"          "UtensilsCrossed" "#10b981"
create_cat "Limpeza"          "Sparkles"     "#06b6d4"
create_cat "Escritório"       "Monitor"      "#3b82f6"
create_cat "Jardim"           "Leaf"         "#22c55e"
create_cat "Brinquedos"       "Gamepad2"     "#ec4899"

ok "8 categorias criadas"

# ── locais ────────────────────────────────────────────────────────────────────
echo ""
info "Criando locais..."

declare -A LOC
create_loc() {
  local name="$1" type="$2" desc="$3"
  local res
  res=$(post "locations" "{\"name\":\"[D] $name\",\"type\":\"$type\",\"description\":\"$desc\"}")
  LOC["$name"]=$(echo "$res" | jq -r '.id')
}

create_loc "Sala de Estar"     "comodo"  "Sala principal com sofá e TV"
create_loc "Quarto Principal"  "comodo"  "Quarto do casal"
create_loc "Cozinha"           "comodo"  "Cozinha integrada à área de serviço"
create_loc "Garagem"           "comodo"  "Garagem coberta com bancada de trabalho"
create_loc "Escritório"        "comodo"  "Home office"
create_loc "Área de Serviço"   "comodo"  "Lavanderia e depósito de limpeza"
create_loc "Armário da Sala"   "armario" "Armário embutido na sala de estar"
create_loc "Estante Escritório" "movel"  "Estante com 5 prateleiras"
create_loc "Caixa Ferramentas" "caixa"   "Caixa plástica vermelha na garagem"
create_loc "Quintal"           "comodo"  "Área externa com jardim"

ok "10 locais criados"

# ── itens ─────────────────────────────────────────────────────────────────────
echo ""
info "Criando 50 itens..."

item() {
  local name="$1" desc="$2" brand="$3" model="$4" serial="$5"
  local qty="$6" unit="$7" date="$8" price="$9" cond="${10}"
  local notes="${11}" cat="${12}" loc="${13}"

  local cat_id="${CAT[$cat]:-null}"
  local loc_id="${LOC[$loc]:-null}"

  local payload
  payload=$(jq -cn \
    --arg  name   "$name" \
    --arg  desc   "$desc" \
    --arg  brand  "$brand" \
    --arg  model  "$model" \
    --arg  serial "$serial" \
    --argjson qty  "$qty" \
    --arg  unit   "$unit" \
    --arg  date   "$date" \
    --argjson price "$price" \
    --arg  cond   "$cond" \
    --arg  notes  "[DEMO] $notes" \
    --argjson cat_id  "$cat_id" \
    --argjson loc_id  "$loc_id" \
    '{name:$name, description:$desc, brand:$brand, model:$model,
      serial_number:$serial, quantity:$qty, unit:$unit,
      purchase_date:$date, purchase_price:$price, condition:$cond,
      notes:$notes, category_id:$cat_id, location_id:$loc_id}')

  post "items" "$payload" | jq -r '.id' &>/dev/null || warn "Falha ao criar: $name"
  printf "  %-45s %s\n" "$name" "✔"
}

# ── Eletrônicos ──
item "Smart TV 55\" 4K" \
  "Televisão OLED com HDR e controle remoto por voz" \
  "LG" "OLED55C3" "LG2024BR-001293" \
  1 "un" "2024-01-15" 4299.90 "novo" \
  "Garantia até jan/2026" "Eletrônicos" "Sala de Estar"

item "Notebook Gamer" \
  "Processador Intel i7, 16GB RAM, SSD 512GB, GPU RTX 3060" \
  "Dell" "G15 5525" "SN-DELL-G15-88821" \
  1 "un" "2023-06-20" 5499.00 "bom" \
  "Adaptador americano na gaveta" "Eletrônicos" "Escritório"

item "Tablet iPad" \
  "iPad Air 5ª geração, 64GB, WiFi" \
  "Apple" "iPad Air 5" "DMPR4X9Q1F" \
  1 "un" "2023-11-10" 3799.00 "bom" \
  "Com capa protetora azul" "Eletrônicos" "Sala de Estar"

item "Smartphone Reserva" \
  "Celular antigo em bom estado para emergências" \
  "Samsung" "Galaxy A52" "RZ8R21LXBKH" \
  1 "un" "2021-03-05" 1299.00 "regular" \
  "Carregador na gaveta do escritório" "Eletrônicos" "Armário da Sala"

item "Fone de Ouvido Bluetooth" \
  "Headphone over-ear com cancelamento de ruído ativo" \
  "Sony" "WH-1000XM5" "SN-SONY-00128XM5" \
  1 "un" "2023-09-01" 1799.00 "novo" \
  "Case incluída" "Eletrônicos" "Escritório"

item "Câmera de Segurança" \
  "Câmera IP externa, Full HD, visão noturna 30m" \
  "Intelbras" "VIP 1030 G4" "CAM-ITB-2024-003" \
  3 "un" "2024-02-01" 349.90 "novo" \
  "2 instaladas + 1 reserva" "Eletrônicos" "Garagem"

item "Roteador Wi-Fi" \
  "Roteador dual band AC1200, 4 portas Gigabit" \
  "TP-Link" "Archer C6" "TPL-2022-AC6-001" \
  1 "un" "2022-07-14" 259.90 "bom" \
  "Senha na etiqueta inferior" "Eletrônicos" "Sala de Estar"

item "Teclado Mecânico" \
  "Teclado sem fio com switches Brown, retroiluminado" \
  "Keychron" "K8 Pro" "KC-K8P-BR-0042" \
  1 "un" "2023-04-22" 699.00 "novo" \
  "Cabo USB-C incluso" "Eletrônicos" "Escritório"

item "Mouse Ergonômico" \
  "Mouse vertical sem fio, 1600 DPI" \
  "Logitech" "MX Vertical" "MXV-LGT-2023-07" \
  1 "un" "2023-04-22" 449.00 "novo" \
  "Pilhas AA incluídas (2 un.)" "Eletrônicos" "Escritório"

item "Carregador Portátil" \
  "Power bank 20000mAh, carga rápida 22.5W, 3 saídas USB" \
  "Xiaomi" "Redmi PB200LZM" "XI-PB-20K-00441" \
  2 "un" "2023-12-25" 179.90 "novo" \
  "1 na mochila, 1 reserva no armário" "Eletrônicos" "Armário da Sala"

# ── Eletrodomésticos ──
item "Geladeira Frost Free" \
  "Refrigerador duplex 460L, painel digital, dispensador de água" \
  "Brastemp" "BRM59AB" "BR-GEL-2022-10291" \
  1 "un" "2022-01-20" 3899.00 "bom" \
  "Filtro trocado em mar/2024" "Eletrodomésticos" "Cozinha"

item "Máquina de Lavar" \
  "Lavadora 12kg, 16 programas, inverter" \
  "Samsung" "WW12T504DAW" "SAM-LAV-2023-00781" \
  1 "un" "2023-03-10" 2799.00 "bom" \
  "Manual na gaveta da área de serviço" "Eletrodomésticos" "Área de Serviço"

item "Micro-ondas" \
  "Forno micro-ondas 30L, 10 níveis de potência, grill" \
  "Electrolux" "MEG41" "ELX-MO-2021-55392" \
  1 "un" "2021-08-15" 649.00 "bom" \
  "Prato de vidro sem trinca" "Eletrodomésticos" "Cozinha"

item "Aspirador de Pó" \
  "Aspirador vertical sem fio, 25V, 45 min autonomia" \
  "Dyson" "V8 Absolute" "DYS-V8-2023-UK441" \
  1 "un" "2023-07-04" 2199.00 "bom" \
  "Filtro lavado mensalmente" "Eletrodomésticos" "Área de Serviço"

item "Ventilador de Teto" \
  "Ventilador de teto 4 pás, controle remoto, LED integrado" \
  "Ventisol" "VT50 Gold" "VNT-50G-2022-0018" \
  2 "un" "2022-11-12" 399.90 "bom" \
  "1 sala + 1 quarto" "Eletrodomésticos" "Sala de Estar"

item "Ar-condicionado 12000 BTU" \
  "Split inverter quente/frio, wifi, filtragem HEPA" \
  "Midea" "MAW12F5" "MID-AC12-2023-7821" \
  1 "un" "2023-01-30" 2599.00 "novo" \
  "Revisão anual prevista jan/2025" "Eletrodomésticos" "Quarto Principal"

item "Purificador de Água" \
  "Purificador bancada, refrigerado, 3 temperaturas" \
  "Electrolux" "PA31G" "ELX-PA31-2022-1092" \
  1 "un" "2022-05-03" 799.00 "bom" \
  "Filtro trocado a cada 6 meses" "Eletrodomésticos" "Cozinha"

item "Sanduicheira e Grill" \
  "Sanduicheira/grill elétrico 1200W, antiaderente" \
  "Philips" "HD2395" "PHL-SG-2021-4401" \
  1 "un" "2021-12-25" 249.00 "bom" \
  "Placas removíveis e laváveis" "Eletrodomésticos" "Cozinha"

# ── Ferramentas ──
item "Furadeira de Impacto" \
  "Furadeira/parafusadeira de impacto 13mm, maleta completa" \
  "Bosch" "GSB 13 RE" "BSH-FUR-2022-GS13RE-001" \
  1 "un" "2022-04-18" 449.00 "bom" \
  "Brocas e bits na caixa junto" "Ferramentas" "Caixa Ferramentas"

item "Serra Circular" \
  "Serra circular 7.1/4\", 1400W, guia paralela" \
  "Makita" "5007MGA" "MKT-5007-2021-JP0042" \
  1 "un" "2021-09-05" 899.00 "bom" \
  "Lâmina reserva na gaveta" "Ferramentas" "Garagem"

item "Parafusadeira a Bateria" \
  "Parafusadeira 12V, 2 baterias, maleta" \
  "Dewalt" "DCD701D2" "DWL-PF12-2023-00381" \
  1 "un" "2023-02-14" 649.00 "novo" \
  "Baterias sempre carregadas" "Ferramentas" "Caixa Ferramentas"

item "Nível a Laser" \
  "Nível a laser autonivelante 2 linhas, alcance 15m" \
  "Bosch" "GLL 2-12" "BSH-NVL-2023-GLL212" \
  1 "un" "2023-08-20" 399.00 "novo" \
  "Inclui tripé e bolsa" "Ferramentas" "Caixa Ferramentas"

item "Jogo de Chaves de Fenda" \
  "Jogo com 12 chaves: fenda, Phillips e Torx" \
  "Stanley" "STMT74455-840" "STLY-CJ12-2020" \
  1 "jogo" "2020-03-01" 89.90 "bom" \
  "Caixa plástica com travas" "Ferramentas" "Caixa Ferramentas"

item "Trena Digital" \
  "Trena a laser, alcance 40m, memória 20 medidas" \
  "Bosch" "GLM 40" "BSH-TRN-2022-GLM40" \
  1 "un" "2022-10-10" 279.00 "novo" \
  "Pilhas incluso" "Ferramentas" "Caixa Ferramentas"

item "Compressor de Ar" \
  "Compressor 24L, 2HP, 8bar, kit com mangueira 10m" \
  "Schulz" "CSL 24 Air Plus" "SLZ-COMP-2021-CSL24" \
  1 "un" "2021-06-15" 1099.00 "bom" \
  "Óleo trocado anualmente" "Ferramentas" "Garagem"

item "Martelo de Borracha" \
  "Martelo de borracha 500g, cabo fibra de vidro" \
  "Tramontina" "PRO 78000" "TRM-MAR-500-PRO" \
  2 "un" "2020-01-10" 39.90 "bom" \
  "Uso geral na garagem" "Ferramentas" "Caixa Ferramentas"

# ── Cozinha ──
item "Jogo de Panelas" \
  "Jogo 5 peças em alumínio com revestimento antiaderente" \
  "Tramontina" "Serie Turim" "TRM-PAN-5PC-2022" \
  1 "jogo" "2022-02-20" 349.90 "bom" \
  "Tampa de vidro inclusa" "Cozinha" "Cozinha"

item "Liquidificador" \
  "Liquidificador 1000W, 3 velocidades, jarra inquebrável 2L" \
  "Philips" "HR2221" "PHL-LIQ-2023-HR2221" \
  1 "un" "2023-05-15" 179.00 "bom" \
  "Lâminas afiadas, sem arranhões" "Cozinha" "Cozinha"

item "Air Fryer 5L" \
  "Fritadeira elétrica sem óleo, display digital, timer" \
  "Mondial" "AF-48-DI" "MND-AF5L-2023-AF48" \
  1 "un" "2023-08-01" 269.90 "novo" \
  "Manual guardado na gaveta" "Cozinha" "Cozinha"

item "Cafeteira Expresso" \
  "Cafeteira espresso 15 bar, vaporizador de leite" \
  "Nespresso" "Vertuo Pop" "NSP-VPP-2024-BRZ01" \
  1 "un" "2024-01-10" 599.00 "novo" \
  "Cápsulas armazenadas no armário" "Cozinha" "Cozinha"

item "Faca do Chef" \
  "Faca profissional 20cm, aço inox alemão, cabo ergonômico" \
  "Tramontina" "Churrasco 22399/108" "TRM-FCH-2021" \
  3 "un" "2021-11-20" 129.90 "bom" \
  "Amolada a cada 3 meses" "Cozinha" "Cozinha"

item "Tábua de Corte" \
  "Tábua de madeira bambu, 40x25cm, sulco anti-transbordamento" \
  "Casa e Mesa" "TB-40B" "CEM-TAB-2022-001" \
  2 "un" "2022-07-08" 59.90 "bom" \
  "1 para carnes, 1 para verduras" "Cozinha" "Cozinha"

item "Balança de Cozinha" \
  "Balança digital 5kg, precisão 1g, plataforma inox" \
  "G-Tech" "Glass 5" "GTH-BAL-2022-GL5" \
  1 "un" "2022-03-25" 89.90 "bom" \
  "Pilhas CR2032 reserva no armário" "Cozinha" "Cozinha"

# ── Limpeza ──
item "Esfregão com Balde Centrífugo" \
  "Kit esfregão com cabo extensível e balde 14L" \
  "Perfecta" "Giratório Pro" "PFT-ESF-2022-GI14" \
  1 "un" "2022-09-03" 89.90 "bom" \
  "Repor refis quando necessário" "Limpeza" "Área de Serviço"

item "Detergente Concentrado" \
  "Detergente líquido neutro concentrado, caixa com 12 unidades" \
  "Ypê" "Clear 500ml" "YPE-DET-CX12-2024" \
  12 "un" "2024-03-01" 5.90 "novo" \
  "Estoque no armário da área de serviço" "Limpeza" "Área de Serviço"

item "Álcool 70° INPM" \
  "Álcool etílico hidratado 70°, galão 5L" \
  "Rioquímica" "ALC70-5L" "RQ-ALC70-2024-03" \
  2 "un" "2024-02-10" 29.90 "novo" \
  "Longe do fogo. Validade 2 anos." "Limpeza" "Área de Serviço"

item "Pano de Microfibra" \
  "Kit 10 panos de microfibra para limpeza geral" \
  "3M" "Scotch-Brite MFB" "3M-PNM-KIT10-2023" \
  10 "un" "2023-10-15" 8.90 "novo" \
  "Separados por cor: azul/cozinha, amarelo/banheiro" "Limpeza" "Área de Serviço"

item "Luva de Borracha" \
  "Luvas de borracha para limpeza, tamanho M" \
  "Condor" "Multiuso M" "CDR-LUV-M-2024" \
  4 "par" "2024-01-20" 12.90 "novo" \
  "2 pares em uso, 2 reserva" "Limpeza" "Área de Serviço"

# ── Escritório ──
item "Monitor 27\" QHD" \
  "Monitor 27 polegadas QHD 165Hz, IPS, HDR400" \
  "LG" "27GP850-B" "LGD-MON27-2023-00412" \
  1 "un" "2023-07-20" 1899.00 "novo" \
  "Cabo HDMI e DisplayPort incluídos" "Escritório" "Escritório"

item "Impressora Multifuncional" \
  "Impressora jato de tinta, WiFi, scanner, copiadora" \
  "HP" "DeskJet 3776" "HP-IMP-2022-DJ3776" \
  1 "un" "2022-06-12" 499.00 "bom" \
  "Cartuchos reserva na gaveta" "Escritório" "Escritório"

item "Cadeira de Escritório" \
  "Cadeira ergonômica, encosto mesh, altura ajustável, apoio lombar" \
  "Flexform" "Max Air" "FLX-CAD-2023-MAXAIR" \
  1 "un" "2023-01-05" 1199.00 "bom" \
  "Regulagem lombar no máximo" "Escritório" "Escritório"

item "Webcam Full HD" \
  "Webcam 1080p/30fps, microfone embutido, plug and play" \
  "Logitech" "C920s" "LGT-WEB-2022-C920S" \
  1 "un" "2022-04-01" 379.00 "bom" \
  "Tampa de privacidade incluída" "Escritório" "Estante Escritório"

item "Hub USB-C 7 em 1" \
  "Adaptador USB-C com HDMI 4K, 3x USB-A, SD, microSD, PD 100W" \
  "Baseus" "CAHUB-BSUR" "BSS-HUB-7N1-2023" \
  2 "un" "2023-03-18" 129.90 "novo" \
  "1 no escritório, 1 na mochila" "Escritório" "Escritório"

item "Resma de Papel A4" \
  "Papel A4 75g/m², 500 folhas, branco natural" \
  "Navigator" "A4 75g" "NAV-A4-500-2024Q1" \
  5 "resma" "2024-01-08" 32.90 "novo" \
  "Estoque para 3 meses" "Escritório" "Estante Escritório"

# ── Jardim ──
item "Mangueira de Jardim" \
  "Mangueira retrátil 30m, anti-torção, bico multifunção" \
  "Tramontina" "Flexível 30m" "TRM-MNG-30M-2022" \
  1 "un" "2022-10-05" 149.90 "bom" \
  "Guardada enrolada na sombra" "Jardim" "Quintal"

item "Regador de Aço" \
  "Regador de aço inox, 10 litros, bico removível" \
  "Vonder" "AÇO-10L" "VND-REG-10L-2021" \
  1 "un" "2021-05-20" 79.90 "bom" \
  "Uso no jardim interno" "Jardim" "Quintal"

item "Fertilizante NPK" \
  "Fertilizante granulado NPK 10-10-10, saco 10kg" \
  "Vitaplan" "NPK 10-10-10" "VTP-NPK10-2024-02" \
  2 "saco" "2024-02-28" 49.90 "novo" \
  "Aplicar mensalmente na terra" "Jardim" "Quintal"

item "Tesoura de Poda" \
  "Tesoura de poda telescópica 60-90cm, lâmina inox" \
  "Tramontina" "Poda Pro 78714001" "TRM-POD-2022-787" \
  1 "un" "2022-08-11" 69.90 "bom" \
  "Ólear lâminas mensalmente" "Jardim" "Quintal"

item "Vasos de Cerâmica" \
  "Vasos de cerâmica pintados à mão, conjunto 3 tamanhos" \
  "Cerâmica Pé de Serra" "Rústico P/M/G" "CPS-VAS-3PC-2023" \
  3 "un" "2023-04-30" 39.90 "bom" \
  "P=15cm, M=22cm, G=30cm de diâmetro" "Jardim" "Quintal"

# ── Brinquedos ──
item "LEGO Technic" \
  "Set LEGO Technic 42128 - Caminhão de Socorro (2017 peças)" \
  "LEGO" "42128" "LEGO-42128-2023-S01" \
  1 "un" "2023-12-25" 899.90 "novo" \
  "Caixa original lacrada" "Brinquedos" "Quarto Principal"

item "Bicicleta Infantil 20\"" \
  "Bicicleta aro 20 com marcha, freio a disco, capacete incluso" \
  "Caloi" "Ceci 20" "CAL-BCI-2022-CECI20" \
  1 "un" "2022-06-01" 849.00 "bom" \
  "Revisão freios necessária" "Brinquedos" "Garagem"

item "Console Videogame" \
  "Console com 2 controles e 3 jogos" \
  "Nintendo" "Switch OLED" "NTD-SWO-2023-BR0091" \
  1 "un" "2023-06-15" 2799.00 "bom" \
  "Caso de proteção e dock na sala" "Brinquedos" "Sala de Estar"

item "Quebra-cabeça 1000 peças" \
  "Quebra-cabeça paisagem montanha 1000 peças, 68x48cm" \
  "Grow" "Alpes Suíços" "GRW-QB1000-2022-ALP" \
  2 "un" "2022-12-20" 49.90 "novo" \
  "Ainda lacrado. Presente guardado." "Brinquedos" "Armário da Sala"

echo ""
ok "50 itens criados com sucesso!"
echo ""
echo -e "${CYAN}─────────────────────────────────────────${NC}"
echo -e "  Para remover: ${YELLOW}./tools/seed-demo.sh --delete${NC}"
echo -e "${CYAN}─────────────────────────────────────────${NC}"
echo ""
