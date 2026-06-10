#!/usr/bin/env bash
# Pré-vol démo : vérifie que tout est opérationnel AVANT une démonstration.
# À lancer juste avant de présenter :  scripts/preflight.sh
set -uo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$ROOT"

c_grn=$'\033[32m'; c_red=$'\033[31m'; c_yel=$'\033[33m'; c_off=$'\033[0m'
FAIL=0
ok()   { printf "  ${c_grn}✓${c_off} %s\n" "$1"; }
ko()   { printf "  ${c_red}✗${c_off} %s\n" "$1"; FAIL=1; }
warn() { printf "  ${c_yel}!${c_off} %s\n" "$1"; }
httpcode() { curl -sk -o /dev/null -w "%{http_code}" --max-time "${2:-5}" "$1" 2>/dev/null; }
listening() { lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1; }

echo "── Services locaux ──"
[ "$(httpcode http://127.0.0.1:50055/metrics)" = "200" ] && ok "Control center — REST/metrics (:50055)" || ko "Control center :50055 injoignable"
listening 50051 && ok "Control center — gRPC (:50051)"          || ko "Control center gRPC :50051 absent"
listening 50052 && ok "Microservice OpenStack (:50052)"          || ko "Microservice :50052 absent (provisionnement KO)"
listening 5173  && ok "Frontend (:5173)"                         || warn "Frontend :5173 absent"
# Caddy tourne en root → lsof ne le voit pas sans sudo ; on sonde en HTTPS (000 = absent).
caddy_code="$(httpcode https://127.0.0.1/ 4)"
[ -n "$caddy_code" ] && [ "$caddy_code" != "000" ] && ok "Caddy reverse proxy (:443)" || warn "Caddy :443 injoignable"
listening 18080 && ok "Tunnel Guacamole (:18080)"                || warn "Tunnel Guacamole absent → terminaux web indisponibles"

echo "── Base de données ──"
eval "$(grep -E '^POSTGRES_(USER|PASSWORD|DB|HOST|PORT)=' .env 2>/dev/null | sed 's/^/PG_/')"
if PGPASSWORD="${PG_POSTGRES_PASSWORD:-}" psql -h "${PG_POSTGRES_HOST:-localhost}" -p "${PG_POSTGRES_PORT:-5432}" \
     -U "${PG_POSTGRES_USER:-admin}" -d "${PG_POSTGRES_DB:-control_center}" -tAc "select 1" >/dev/null 2>&1; then
  ok "PostgreSQL joignable"
else
  ko "PostgreSQL injoignable"
fi

echo "── OpenStack (auth) ──"
eval "$(grep -E '^(OS_CLOUD|INFRA_OS_CLOUD)=' .env 2>/dev/null | sed 's/^/X_/')"
for cloud in "${X_OS_CLOUD:-ipp-idcs-vmpool}" "${X_INFRA_OS_CLOUD:-ipp-idcs-vmpoolmanager}"; do
  if openstack --os-cloud "$cloud" token issue -f value -c expires >/dev/null 2>&1; then
    ok "Auth OpenStack OK ($cloud)"
  else
    ko "Auth OpenStack KO ($cloud) — vérifier clouds.yaml / app-credential"
  fi
done
if [ -f .devlogs/backend.log ] && tail -40 .devlogs/backend.log | grep -q "401"; then
  warn "401 récents dans backend.log (token expiré ? redémarrer le backend)"
fi

echo "── Moodle (optionnel) ──"
if grep -q '^MOODLE_URL=' .env 2>/dev/null; then
  eval "$(grep -E '^MOODLE_(URL|TOKEN)=' .env | sed 's/^/M_/')"
  [ "$(httpcode "$M_MOODLE_URL/login/index.php" 6)" = "200" ] && ok "Moodle UI ($M_MOODLE_URL)" || warn "Moodle UI injoignable"
  if curl -s --max-time 8 "$M_MOODLE_URL/webservice/rest/server.php" \
       --data-urlencode "wstoken=${M_MOODLE_TOKEN:-}" --data-urlencode "wsfunction=core_webservice_get_site_info" \
       --data-urlencode "moodlewsrestformat=json" 2>/dev/null | grep -q sitename; then
    ok "Token Moodle Web Services valide"
  else
    warn "Token Moodle Web Services invalide/absent"
  fi
else
  warn "Moodle non configuré (MOODLE_URL absent) — fonctionnalités Moodle désactivées"
fi

echo
if [ "$FAIL" = "0" ]; then
  printf "${c_grn}✓ Prêt pour la démo.${c_off}\n"
else
  printf "${c_red}✗ Points bloquants ci-dessus — à corriger avant la démo (ex: ./dev.sh restart).${c_off}\n"
fi
exit "$FAIL"
