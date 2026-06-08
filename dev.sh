#!/usr/bin/env bash
# dev.sh — démarre / arrête / redémarre tous les services en arrière-plan, SANS tmux.
#
#   ./dev.sh start            # tout démarrer
#   ./dev.sh stop             # tout arrêter
#   ./dev.sh restart          # tout redémarrer
#   ./dev.sh restart control  # redémarrer un seul service
#   ./dev.sh status           # état de chaque service
#   ./dev.sh logs control     # suivre les logs d'un service (Ctrl-C pour quitter)
#
# Services : backend (microservice OpenStack), control (control center),
#            frontend (Vite), caddy (reverse proxy 443, sudo), auth (Dex/GLAuth docker).
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOGDIR="$ROOT/.devlogs"
mkdir -p "$LOGDIR"
CADDY_BIN="${CADDY_BIN:-$HOME/caddy-grpc}"
GUAC_HOST="${GUAC_HOST:-ubuntu@157.136.249.205}"   # hôte du Guacamole distant (tunnel)

SERVICES=(backend control frontend caddy guac)

# Port "santé" de chaque service (pour status / stop fiable)
port_of() { case "$1" in
  backend) echo 50052 ;; control) echo 50051 ;; frontend) echo 5173 ;; caddy) echo 443 ;; guac) echo 18080 ;; *) echo "" ;;
esac; }

c_grn=$'\033[32m'; c_red=$'\033[31m'; c_yel=$'\033[33m'; c_dim=$'\033[2m'; c_off=$'\033[0m'

port_pid() { lsof -tiTCP:"$1" -sTCP:LISTEN 2>/dev/null | head -1; }

# is_up teste si le port accepte une connexion — marche sans sudo et quel que
# soit le propriétaire (caddy tourne en root, invisible à lsof non-root).
is_up() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null && { exec 3>&- 3<&-; return 0; }; return 1; }

kill_port() { # $1=port — tue le process qui écoute (sudo pour caddy/443)
  local pids; pids=$(lsof -tiTCP:"$1" -sTCP:LISTEN 2>/dev/null)
  [ -z "$pids" ] && return 0
  if [ "$1" = 443 ]; then echo "$pids" | xargs -r sudo kill 2>/dev/null
  else echo "$pids" | xargs -r kill 2>/dev/null; fi
}

start_one() {
  local svc="$1"
  case "$svc" in
    backend)
      ( cd "$ROOT/microservices/openstack" && go build -o bin/app . ) \
        || { echo "${c_red}✗ build backend échoué${c_off}"; return 1; }
      nohup bash -c "cd '$ROOT/microservices/openstack' && PORT=8080 DOTENV_PATH=.env exec ./bin/app" \
        >"$LOGDIR/backend.log" 2>&1 & echo $! >"$LOGDIR/backend.pid" ;;
    control)
      ( cd "$ROOT/control_center" && go build -o bin/cc . ) \
        || { echo "${c_red}✗ build control échoué${c_off}"; return 1; }
      nohup bash -c "cd '$ROOT/control_center' && DOTENV_PATH=.env exec ./bin/cc" \
        >"$LOGDIR/control.log" 2>&1 & echo $! >"$LOGDIR/control.pid" ;;
    frontend)
      nohup bash -c "cd '$ROOT/frontend' && exec npm run dev -- --host --port 5173" \
        >"$LOGDIR/frontend.log" 2>&1 & echo $! >"$LOGDIR/frontend.pid" ;;
    caddy)
      [ -x "$CADDY_BIN" ] || { echo "${c_red}✗ caddy introuvable: $CADDY_BIN${c_off}"; return 1; }
      sudo nohup "$CADDY_BIN" run --config "$ROOT/caddy/Caddyfile.native" --adapter caddyfile \
        >"$LOGDIR/caddy.log" 2>&1 & ;;
    guac)
      # Tunnel SSH auto-reconnectant vers le Guacamole distant (0.0.0.0:18080 -> remote:8080).
      # La boucle relance ssh dès qu'il tombe (coupure réseau, timeout…).
      pkill -f "18080:127.0.0.1:8080" 2>/dev/null || true
      nohup bash -c "while true; do \
        echo \"[guac] \$(date '+%H:%M:%S') connexion du tunnel…\"; \
        ssh -N -o StrictHostKeyChecking=no -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes \
          -L 0.0.0.0:18080:127.0.0.1:8080 $GUAC_HOST; \
        echo \"[guac] \$(date '+%H:%M:%S') tunnel tombé, reconnexion dans 5s…\"; \
        sleep 5; \
      done" >"$LOGDIR/guac.log" 2>&1 & echo $! >"$LOGDIR/guac.pid" ;;
    auth)
      ( cd "$ROOT" && task auth >"$LOGDIR/auth.log" 2>&1 ) \
        && echo "${c_grn}✓ auth (docker)${c_off}" || echo "${c_yel}! auth: voir .devlogs/auth.log${c_off}"
      return 0 ;;
    *) echo "service inconnu: $svc"; return 1 ;;
  esac
  echo "${c_grn}✓ $svc démarré${c_off} ${c_dim}(.devlogs/$svc.log)${c_off}"
}

stop_one() {
  local svc="$1" p; p="$(port_of "$svc")"
  [ "$svc" = auth ] && { ( cd "$ROOT" && task auth:stop >/dev/null 2>&1 ); echo "→ auth arrêté"; return 0; }
  # guac = boucle de reconnexion + ssh ; le motif du tunnel matche les deux.
  [ "$svc" = guac ] && { pkill -f "18080:127.0.0.1:8080" 2>/dev/null; rm -f "$LOGDIR/guac.pid"; echo "→ guac arrêté"; return 0; }
  [ -f "$LOGDIR/$svc.pid" ] && { kill "$(cat "$LOGDIR/$svc.pid")" 2>/dev/null; rm -f "$LOGDIR/$svc.pid"; }
  [ -n "$p" ] && kill_port "$p"
  echo "→ $svc arrêté"
}

needs_sudo() { for s in "$@"; do [ "$s" = caddy ] && return 0; done; return 1; }

cmd_start() {
  local list=("$@"); [ ${#list[@]} -eq 0 ] && list=("${SERVICES[@]}")
  needs_sudo "${list[@]}" && { echo "${c_dim}(sudo requis pour Caddy/443)${c_off}"; sudo -v || return 1; }
  for s in "${list[@]}"; do start_one "$s"; done
  echo ""; sleep 2; cmd_status
}

cmd_stop() {
  local list=("$@"); [ ${#list[@]} -eq 0 ] && list=("${SERVICES[@]}")
  for s in "${list[@]}"; do stop_one "$s"; done
}

cmd_restart() {
  local list=("$@"); [ ${#list[@]} -eq 0 ] && list=("${SERVICES[@]}")
  needs_sudo "${list[@]}" && { sudo -v || return 1; }
  for s in "${list[@]}"; do stop_one "$s"; done
  sleep 2
  for s in "${list[@]}"; do start_one "$s"; done
  echo ""; sleep 2; cmd_status
}

cmd_status() {
  echo "${c_dim}service    port    état${c_off}"
  for s in "${SERVICES[@]}"; do
    local p pid; p="$(port_of "$s")"
    if is_up "$p"; then
      pid="$(port_pid "$p")"; [ -n "$pid" ] && pid="(pid $pid)" || pid="${c_dim}(root)${c_off}"
      printf "%-10s %-7s ${c_grn}● up${c_off}   %b\n" "$s" "$p" "$pid"
    else printf "%-10s %-7s ${c_red}○ down${c_off}\n" "$s" "$p"; fi
  done
}

cmd_logs() {
  local svc="${1:-}"; [ -z "$svc" ] && { echo "usage: ./dev.sh logs <backend|control|frontend|caddy|auth>"; return 1; }
  local f="$LOGDIR/$svc.log"; [ -f "$f" ] || { echo "pas de log pour $svc"; return 1; }
  echo "${c_dim}== $f (Ctrl-C pour quitter) ==${c_off}"; tail -n 40 -f "$f"
}

case "${1:-}" in
  start)   shift; cmd_start "$@" ;;
  stop)    shift; cmd_stop "$@" ;;
  restart) shift; cmd_restart "$@" ;;
  status)  cmd_status ;;
  logs)    shift; cmd_logs "$@" ;;
  *) cat <<EOF
Usage: ./dev.sh <commande> [service...]

  start [svc...]    démarre tout (ou les services nommés)
  stop  [svc...]    arrête
  restart [svc...]  redémarre
  status            état de chaque service
  logs <svc>        suit les logs d'un service

Services: ${SERVICES[*]} auth
Exemples:
  ./dev.sh start            # tout lancer
  ./dev.sh restart control  # redémarrer juste le control center
  ./dev.sh logs backend     # voir les logs du microservice
EOF
  ;;
esac
