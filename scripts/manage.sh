#!/usr/bin/env bash
# Model Bay Gateway  —  new-api + sub2api + Caddy  —  Linux Docker manager
# Usage:  ./manage.sh <command> [service]
#   service: newapi | sub2api | caddy | all (default: all)
# Commands: install | update | start | stop | restart | status | version | logs | uninstall

set -euo pipefail

NEWAPI_COMPOSE_URL="https://raw.githubusercontent.com/diamond261/new-api/main/docker-compose.yml"
NEWAPI_GITHUB_REPO="diamond261/new-api"
SUB2API_GITHUB_REPO="nitezs/sub2api"
NEWAPI_IMAGE_BASE="ghcr.io/diamond261/new-api"
SUB2API_IMAGE_BASE="ghcr.io/nitezs/sub2api"

# ── Directories ──────────────────────────────────────────────────────────
GATEWAY_DIR="${GATEWAY_DIR:-/root/model-bay-gateway}"
NEWAPI_DATA_DIR="/root/newapi-data"
SUB2API_DATA_DIR="/root/sub2api"
CADDY_DIR="$GATEWAY_DIR/caddy"
STATIC_DIR="$GATEWAY_DIR/static"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
die()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

ensure_dirs() {
  mkdir -p "$GATEWAY_DIR" "$NEWAPI_DATA_DIR" "$SUB2API_DATA_DIR" \
           "$CADDY_DIR" "$STATIC_DIR"
}

require_docker() {
  command -v docker &>/dev/null || die "Docker not found. Install it first: https://docs.docker.com/engine/install/"
  docker info &>/dev/null       || die "Docker daemon is not running."
  if docker compose version &>/dev/null; then
    COMPOSE_CMD="docker compose"
  elif command -v docker-compose &>/dev/null; then
    COMPOSE_CMD="docker-compose"
  else
    die "Docker Compose not found. Install it or ensure Docker includes the compose plugin."
  fi
}

fetch_latest_version() {
  local repo="$1"
  local version
  if command -v curl &>/dev/null; then
    version=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" \
      | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
  else
    version=$(wget -qO- "https://api.github.com/repos/$repo/releases/latest" \
      | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
  fi
  echo "${version:-latest}"
}

get_running_version() {
  local container="$1"
  docker inspect --format '{{index .Config.Image}}' "$container" 2>/dev/null \
    | sed 's/.*://' || echo "unknown"
}

# ── Caddy ────────────────────────────────────────────────────────────────

write_caddyfile() {
  local domain="${DOMAIN:-model-bay.com}"
  cat > "$CADDY_DIR/Caddyfile" <<CADDY
{
  email admin@${domain}
}

${domain} {
  # API gateway — new-api backend
  reverse_proxy localhost:3000

  # Static assets served by Caddy directly
  handle_path /privacy* {
    root * ${STATIC_DIR}
    rewrite * /privacy.html
    file_server
  }
  handle_path /terms* {
    root * ${STATIC_DIR}
    rewrite * /terms.html
    file_server
  }
  handle_path /brand/* {
    root * ${STATIC_DIR}/brand
    file_server
  }
}
CADDY
  ok "Caddyfile written to $CADDY_DIR/Caddyfile"
}

install_caddy() {
  info "Installing Caddy → $CADDY_DIR"
  mkdir -p "$CADDY_DIR" "$STATIC_DIR/brand"
  write_caddyfile
  cat > "$CADDY_DIR/docker-compose.yml" <<YAML
services:
  caddy:
    image: caddy:latest
    container_name: caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ${STATIC_DIR}:${STATIC_DIR}:ro
      - caddy_data:/data
      - caddy_config:/config
    network_mode: host
volumes:
  caddy_data:
  caddy_config:
YAML
  $COMPOSE_CMD -f "$CADDY_DIR/docker-compose.yml" pull
  $COMPOSE_CMD -f "$CADDY_DIR/docker-compose.yml" up -d
  ok "Caddy started. Static files served from $STATIC_DIR"
  info "Place privacy.html, terms.html in $STATIC_DIR"
  info "Place logo.png in $STATIC_DIR/brand/"
}

update_caddy() {
  [[ -f "$CADDY_DIR/docker-compose.yml" ]] || die "Caddy not installed. Run: $0 install caddy"
  info "Updating Caddy…"
  write_caddyfile
  $COMPOSE_CMD -f "$CADDY_DIR/docker-compose.yml" pull
  $COMPOSE_CMD -f "$CADDY_DIR/docker-compose.yml" up -d
  ok "Caddy updated."
}

# ── new-api ──────────────────────────────────────────────────────────────

install_newapi() {
  local ver="${1:-}"
  if [[ -z "$ver" ]]; then
    info "Fetching latest new-api version…"
    ver=$(fetch_latest_version "$NEWAPI_GITHUB_REPO")
  fi
  local image="$NEWAPI_IMAGE_BASE:$ver"
  info "Installing new-api $ver → $GATEWAY_DIR"
  mkdir -p "$GATEWAY_DIR" "$NEWAPI_DATA_DIR"
  if [[ ! -f "$GATEWAY_DIR/docker-compose.yml" ]]; then
    if command -v curl &>/dev/null; then
      curl -fsSL "$NEWAPI_COMPOSE_URL" -o "$GATEWAY_DIR/docker-compose.yml"
    else
      wget -qO "$GATEWAY_DIR/docker-compose.yml" "$NEWAPI_COMPOSE_URL"
    fi
    # Pin the image to the resolved version
    sed -i "s|image:.*new-api.*|image: $image|" "$GATEWAY_DIR/docker-compose.yml" || true
    # Mount data to persistent directory
    sed -i "s|./data:/data|${NEWAPI_DATA_DIR}:/data|" "$GATEWAY_DIR/docker-compose.yml" || true
    ok "docker-compose.yml downloaded (image: $image)."
  else
    warn "docker-compose.yml already exists — skipping download."
  fi

  if [[ ! -f "$GATEWAY_DIR/.env" ]]; then
    cat > "$GATEWAY_DIR/.env" <<'ENV'
# Edit these values before starting new-api
SQL_DSN=                     # leave blank to use SQLite
REDIS_CONN_STRING=           # leave blank to skip Redis
SESSION_SECRET=change_me_please
SYNC_FREQUENCY=60
ENV
  fi

  $COMPOSE_CMD -f "$GATEWAY_DIR/docker-compose.yml" pull
  $COMPOSE_CMD -f "$GATEWAY_DIR/docker-compose.yml" up -d
  ok "new-api $ver started. Dashboard: http://localhost:3000"
  info "Data directory: $NEWAPI_DATA_DIR"
}

update_newapi() {
  [[ -f "$GATEWAY_DIR/docker-compose.yml" ]] || die "new-api not installed at $GATEWAY_DIR. Run: $0 install newapi"
  info "Fetching latest new-api version…"
  local ver
  ver=$(fetch_latest_version "$NEWAPI_GITHUB_REPO")
  local image="$NEWAPI_IMAGE_BASE:$ver"
  info "Updating new-api → $ver"
  sed -i "s|image:.*new-api.*|image: $image|" "$GATEWAY_DIR/docker-compose.yml" || true
  $COMPOSE_CMD -f "$GATEWAY_DIR/docker-compose.yml" pull
  $COMPOSE_CMD -f "$GATEWAY_DIR/docker-compose.yml" up -d
  ok "new-api updated to $ver."
}

# ── sub2api ───────────────────────────────────────────────────────────────

install_sub2api() {
  local ver="${1:-}"
  if [[ -z "$ver" ]]; then
    info "Fetching latest sub2api version…"
    ver=$(fetch_latest_version "$SUB2API_GITHUB_REPO")
  fi
  local image="$SUB2API_IMAGE_BASE:$ver"
  info "Installing sub2api $ver → $SUB2API_DATA_DIR"
  mkdir -p "$SUB2API_DATA_DIR"
  cat > "$SUB2API_DATA_DIR/docker-compose.yml" <<YAML
services:
  sub2api:
    image: $image
    container_name: sub2api
    restart: unless-stopped
    ports:
      - "18080:8080"
    volumes:
      - ./data:/app/data
YAML
  $COMPOSE_CMD -f "$SUB2API_DATA_DIR/docker-compose.yml" pull
  $COMPOSE_CMD -f "$SUB2API_DATA_DIR/docker-compose.yml" up -d
  ok "sub2api $ver started on port 18080."
}

update_sub2api() {
  [[ -f "$SUB2API_DATA_DIR/docker-compose.yml" ]] || die "sub2api not installed at $SUB2API_DATA_DIR. Run: $0 install sub2api"
  info "Fetching latest sub2api version…"
  local ver
  ver=$(fetch_latest_version "$SUB2API_GITHUB_REPO")
  local image="$SUB2API_IMAGE_BASE:$ver"
  info "Updating sub2api → $ver"
  sed -i "s|image:.*sub2api.*|image: $image|" "$SUB2API_DATA_DIR/docker-compose.yml" || true
  $COMPOSE_CMD -f "$SUB2API_DATA_DIR/docker-compose.yml" pull
  $COMPOSE_CMD -f "$SUB2API_DATA_DIR/docker-compose.yml" up -d
  ok "sub2api updated to $ver."
}

# ── version info ──────────────────────────────────────────────────────────

version_service() {
  local name="$1" repo="$2" container="$3"
  echo -e "\n${CYAN}── $name ──────────────────────────${NC}"
  local latest running
  latest=$(fetch_latest_version "$repo")
  running=$(get_running_version "$container")
  echo -e "  Latest available : ${GREEN}$latest${NC}"
  echo -e "  Running image    : ${YELLOW}$running${NC}"
  if [[ "$running" == "$latest" ]]; then
    ok "Up to date."
  else
    warn "Update available: $0 update ${name//[-]/_}"
  fi
}

# ── generic compose helpers ───────────────────────────────────────────────

compose_cmd() {
  local dir="$1"; shift
  [[ -f "$dir/docker-compose.yml" ]] || die "No docker-compose.yml in $dir"
  $COMPOSE_CMD -f "$dir/docker-compose.yml" "$@"
}

status_service() {
  local name="$1" dir="$2"
  echo -e "\n${BLUE}── $name ──────────────────────────${NC}"
  if [[ -f "$dir/docker-compose.yml" ]]; then
    $COMPOSE_CMD -f "$dir/docker-compose.yml" ps
  else
    warn "Not installed at $dir"
  fi
}

logs_service() {
  local name="$1" dir="$2"
  [[ -f "$dir/docker-compose.yml" ]] || die "$name not installed at $dir"
  info "Tailing logs for $name  (Ctrl+C to stop)"
  $COMPOSE_CMD -f "$dir/docker-compose.yml" logs -f --tail=100
}

uninstall_service() {
  local name="$1" dir="$2"
  [[ -f "$dir/docker-compose.yml" ]] || { warn "$name not installed at $dir"; return; }
  read -rp "$(echo -e "${YELLOW}Remove $name containers + volumes? [y/N] ${NC}")" confirm
  [[ "$confirm" =~ ^[Yy]$ ]] || { info "Aborted."; return; }
  $COMPOSE_CMD -f "$dir/docker-compose.yml" down -v
  rm -rf "$dir"
  ok "$name removed."
}

# ── dispatch ──────────────────────────────────────────────────────────────

CMD="${1:-help}"
SVC="${2:-all}"

require_docker
ensure_dirs

case "$CMD" in
  install)
    case "$SVC" in
      newapi)  install_newapi ;;
      sub2api) install_sub2api ;;
      caddy)   install_caddy ;;
      all)     install_newapi; install_sub2api; install_caddy ;;
      *) die "Unknown service: $SVC  (newapi | sub2api | caddy | all)" ;;
    esac ;;

  update)
    case "$SVC" in
      newapi)  update_newapi ;;
      sub2api) update_sub2api ;;
      caddy)   update_caddy ;;
      all)     update_newapi; update_sub2api; update_caddy ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  start)
    case "$SVC" in
      newapi)  compose_cmd "$GATEWAY_DIR"      up -d ;;
      sub2api) compose_cmd "$SUB2API_DATA_DIR" up -d ;;
      caddy)   compose_cmd "$CADDY_DIR"        up -d ;;
      all)     compose_cmd "$GATEWAY_DIR"      up -d; compose_cmd "$SUB2API_DATA_DIR" up -d; compose_cmd "$CADDY_DIR" up -d ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  stop)
    case "$SVC" in
      newapi)  compose_cmd "$GATEWAY_DIR"      stop ;;
      sub2api) compose_cmd "$SUB2API_DATA_DIR" stop ;;
      caddy)   compose_cmd "$CADDY_DIR"        stop ;;
      all)     compose_cmd "$GATEWAY_DIR"      stop; compose_cmd "$SUB2API_DATA_DIR" stop; compose_cmd "$CADDY_DIR" stop ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  restart)
    case "$SVC" in
      newapi)  compose_cmd "$GATEWAY_DIR"      restart ;;
      sub2api) compose_cmd "$SUB2API_DATA_DIR" restart ;;
      caddy)   compose_cmd "$CADDY_DIR"        restart ;;
      all)     compose_cmd "$GATEWAY_DIR"      restart; compose_cmd "$SUB2API_DATA_DIR" restart; compose_cmd "$CADDY_DIR" restart ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  status)
    case "$SVC" in
      newapi)  status_service "new-api"  "$GATEWAY_DIR" ;;
      sub2api) status_service "sub2api"  "$SUB2API_DATA_DIR" ;;
      caddy)   status_service "caddy"    "$CADDY_DIR" ;;
      all)     status_service "new-api"  "$GATEWAY_DIR"; status_service "sub2api" "$SUB2API_DATA_DIR"; status_service "caddy" "$CADDY_DIR" ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  version)
    case "$SVC" in
      newapi)  version_service "new-api"  "$NEWAPI_GITHUB_REPO"  "new-api" ;;
      sub2api) version_service "sub2api"  "$SUB2API_GITHUB_REPO" "sub2api" ;;
      all)     version_service "new-api"  "$NEWAPI_GITHUB_REPO"  "new-api"
               version_service "sub2api"  "$SUB2API_GITHUB_REPO" "sub2api" ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  logs)
    case "$SVC" in
      newapi)  logs_service "new-api"  "$GATEWAY_DIR" ;;
      sub2api) logs_service "sub2api"  "$SUB2API_DATA_DIR" ;;
      caddy)   logs_service "caddy"    "$CADDY_DIR" ;;
      all)     die "Specify a service for logs: $0 logs newapi|sub2api|caddy" ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  uninstall)
    case "$SVC" in
      newapi)  uninstall_service "new-api"  "$GATEWAY_DIR" ;;
      sub2api) uninstall_service "sub2api"  "$SUB2API_DATA_DIR" ;;
      caddy)   uninstall_service "caddy"    "$CADDY_DIR" ;;
      all)     uninstall_service "caddy"    "$CADDY_DIR"; uninstall_service "new-api" "$GATEWAY_DIR"; uninstall_service "sub2api" "$SUB2API_DATA_DIR" ;;
      *) die "Unknown service: $SVC" ;;
    esac ;;

  help|--help|-h|*)
    cat <<HELP

  Model Bay Gateway  —  new-api + sub2api + Caddy on Docker

  Usage:  $0 <command> [service]

  Commands:
    install   [newapi|sub2api|caddy|all]   Pull latest release + start containers
    update    [newapi|sub2api|caddy|all]   Fetch latest release + recreate
    version   [newapi|sub2api|all]         Show running vs latest available version
    start     [newapi|sub2api|caddy|all]   Start stopped containers
    stop      [newapi|sub2api|caddy|all]   Stop running containers
    restart   [newapi|sub2api|caddy|all]   Restart containers
    status    [newapi|sub2api|caddy|all]   Show container status
    logs       newapi|sub2api|caddy        Tail live logs (Ctrl+C to stop)
    uninstall [newapi|sub2api|caddy|all]   Stop + remove containers and data

  Directories:
    /root/model-bay-gateway          Gateway configs, Caddy, compose files
    /root/model-bay-gateway/static   Static files (privacy.html, terms.html, brand/logo.png)
    /root/model-bay-gateway/caddy    Caddyfile + Caddy compose
    /root/newapi-data                new-api persistent data (DB, uploads)
    /root/sub2api                    sub2api persistent data

  Environment:
    GATEWAY_DIR   Override gateway base directory (default: /root/model-bay-gateway)
    DOMAIN        Domain for Caddy HTTPS (default: model-bay.com)

  Examples:
    $0 install                          # install everything (latest releases)
    $0 update newapi                    # update new-api to latest release
    $0 version                          # check running vs available versions
    DOMAIN=api.example.com $0 install caddy  # install Caddy with custom domain
    $0 logs caddy                       # tail Caddy logs
    $0 status                           # show status of all services

HELP
    ;;
esac
