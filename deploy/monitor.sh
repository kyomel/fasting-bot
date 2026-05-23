#!/bin/bash
set -euo pipefail

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
APP_DB="${DATABASE_PATH:-$(read_env_value DATABASE_PATH)}"
APP_DB="${APP_DB:-$DATA_DIR/fasting-bot.db}"
SESSION_DB="${SESSION_PATH:-$(read_env_value SESSION_PATH)}"
SESSION_DB="${SESSION_DB:-$DATA_DIR/whatsapp-session.db}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-$(read_env_value BACKUP_RETENTION_DAYS)}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"

mkdir -p "$BACKUP_DIR"

log() {
  echo "[$(date '+%F %T')] $*" >> "$LOG"
}

require_sqlite3() {
  if ! command -v sqlite3 >/dev/null 2>&1; then
    log "ERROR - sqlite3 is required for backup/restore"
    echo "sqlite3 is required" >&2
    exit 1
  fi
}

latest_backup() {
  find "$BACKUP_DIR" -type f -name 'fasting-bot-*.db' -print | sort | tail -n 1
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
    require_sqlite3
    if [ ! -f "$APP_DB" ]; then
      log "ERROR - app database not found: $APP_DB"
      exit 1
    fi

    stamp="$(date +%Y%m%d-%H%M%S)"
    tmp="$BACKUP_DIR/.fasting-bot-$stamp.db.tmp"
    dest="$BACKUP_DIR/fasting-bot-$stamp.db"
    dump="$BACKUP_DIR/fasting-bot-$stamp.sql.gz"

    find "$BACKUP_DIR" -type f \( -name 'fasting-bot-*.db' -o -name 'fasting-bot-*.sql.gz' \) -mtime +"$RETENTION_DAYS" -delete 2>/dev/null || true

    sqlite3 "$APP_DB" "PRAGMA wal_checkpoint(PASSIVE);" >/dev/null
    sqlite3 "$APP_DB" ".backup '$tmp'"
    if [ "$(sqlite3 "$tmp" "PRAGMA quick_check;")" != "ok" ]; then
      rm -f "$tmp"
      log "ERROR - backup quick_check failed"
      exit 1
    fi

    mv "$tmp" "$dest"
    sqlite3 "$dest" ".dump" | gzip -c > "$dump"
    chmod 600 "$dest" "$dump"

    log "Backup done: $dest and $dump"
    ;;
  restore)
    require_sqlite3
    backup="${2:-$(latest_backup)}"
    if [ -z "$backup" ] || [ ! -f "$backup" ]; then
      log "ERROR - backup file not found: ${backup:-<none>}"
      echo "backup file not found: ${backup:-<none>}" >&2
      exit 1
    fi

    stamp="$(date +%Y%m%d-%H%M%S)"
    pre_restore="$BACKUP_DIR/pre-restore-$stamp.db"

    sudo systemctl stop fasting-bot
    if [ -f "$APP_DB" ]; then
      sqlite3 "$APP_DB" ".backup '$pre_restore'" || cp "$APP_DB" "$pre_restore"
      chmod 600 "$pre_restore" || true
    fi

    rm -f "$APP_DB" "$APP_DB-wal" "$APP_DB-shm"
    if [[ "$backup" == *.sql.gz ]]; then
      gzip -dc "$backup" | sqlite3 "$APP_DB"
    else
      cp "$backup" "$APP_DB"
    fi
    chown fastingbot:fastingbot "$APP_DB" 2>/dev/null || true
    chmod 600 "$APP_DB"
    if [ "$(sqlite3 "$APP_DB" "PRAGMA quick_check;")" != "ok" ]; then
      log "ERROR - restored database quick_check failed from $backup"
      exit 1
    fi
    sudo systemctl start fasting-bot

    log "Restore done from $backup; previous DB saved at $pre_restore"
    ;;
  reset-session)
    sudo systemctl stop fasting-bot
    rm -f "$SESSION_DB" "$SESSION_DB-wal" "$SESSION_DB-shm"
    sudo systemctl start fasting-bot
    log "WhatsApp session reset only: $SESSION_DB"
    ;;
  *)
    echo "Usage: $0 {healthcheck|backup|restore [backup.db|backup.sql.gz]|reset-session}"
    exit 1
    ;;
esac
