#!/bin/bash
set -euo pipefail

# monitor.sh — healthcheck + WhatsApp session reset + Postgres backup helpers.
#
# Application database is PostgreSQL (DB_CONNECTION). Backups use pg_dump;
# the SQLite helpers were removed with the SQLite app-DB cutover. The
# WhatsApp session store (SESSION_PATH) is still SQLite and is managed only
# by reset-session.

ENV_FILE="${ENV_FILE:-/opt/fasting-bot/.env}"

read_env_value() {
  local key="$1"
  local value=""

  if [ -f "$ENV_FILE" ]; then
    value="$(grep -E "^[[:space:]]*$key=" "$ENV_FILE" | tail -n 1 | cut -d= -f2- || true)"
  fi
  value="${value%%#*}"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  printf '%s' "$value"
}

DATA_DIR="${DATA_DIR:-/opt/fasting-bot/data}"
LOG="${LOG:-$DATA_DIR/monitor.log}"
BACKUP_DIR="${BACKUP_DIR:-$DATA_DIR/backups}"
DB_URL="${DB_CONNECTION:-$(read_env_value DB_CONNECTION)}"
SESSION_DB="${SESSION_PATH:-$(read_env_value SESSION_PATH)}"
SESSION_DB="${SESSION_DB:-$DATA_DIR/whatsapp-session.db}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-$(read_env_value BACKUP_RETENTION_DAYS)}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"

mkdir -p "$BACKUP_DIR"

log() {
  echo "[$(date '+%F %T')] $*" >> "$LOG"
}

require_db_url() {
  if [ -z "$DB_URL" ]; then
    log "ERROR - DB_CONNECTION is not set (env or $ENV_FILE)"
    echo "DB_CONNECTION is not set" >&2
    exit 1
  fi
}

case "${1:-}" in
  healthcheck)
    if ! systemctl is-active --quiet fasting-bot; then
      log "DOWN - restarting"
      sudo systemctl restart fasting-bot
      sleep 5
      systemctl is-active --quiet fasting-bot && log "UP after restart"
    fi
    ;;
  backup)
    require_db_url
    stamp="$(date +%Y%m%d-%H%M%S)"
    dest="$BACKUP_DIR/fasting-bot-$stamp.sql.gz"

    find "$BACKUP_DIR" -type f -name 'fasting-bot-*.sql.gz' -mtime +"$RETENTION_DAYS" -delete 2>/dev/null || true

    pg_dump "$DB_URL" | gzip -c > "$dest"
    chmod 600 "$dest"

    log "Backup done: $dest"
    ;;
  restore)
    require_db_url
    backup="${2:-$(find "$BACKUP_DIR" -type f -name 'fasting-bot-*.sql.gz' -print | sort | tail -n 1)}"
    if [ -z "$backup" ] || [ ! -f "$backup" ]; then
      log "ERROR - backup file not found: ${backup:-<none>}"
      echo "backup file not found: ${backup:-<none>}" >&2
      exit 1
    fi

    sudo systemctl stop fasting-bot
    gzip -dc "$backup" | psql "$DB_URL"
    sudo systemctl start fasting-bot

    log "Restore done from $backup"
    ;;
  reset-session)
    sudo systemctl stop fasting-bot
    rm -f "$SESSION_DB" "$SESSION_DB-wal" "$SESSION_DB-shm"
    sudo systemctl start fasting-bot
    log "WhatsApp session reset only: $SESSION_DB"
    ;;
  *)
    echo "Usage: $0 {healthcheck|backup|restore [backup.sql.gz]|reset-session}"
    exit 1
    ;;
esac
