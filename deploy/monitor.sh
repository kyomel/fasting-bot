#!/bin/bash
set -euo pipefail

LOG="/opt/fasting-bot/data/monitor.log"
BACKUP_DIR="/opt/fasting-bot/data/backups"

mkdir -p "$BACKUP_DIR"

case "${1:-}" in
  healthcheck)
    if ! systemctl is-active --quiet fasting-bot; then
      echo "[$(date '+%F %T')] DOWN - restarting" >> "$LOG"
      sudo systemctl restart fasting-bot
      sleep 5
      systemctl is-active --quiet fasting-bot && echo "[$(date '+%F %T')] UP after restart" >> "$LOG"
    fi
    ;;
  backup)
    find "$BACKUP_DIR" -name "*.bak-*" -mtime +7 -delete 2>/dev/null || true
    sudo systemctl stop fasting-bot; sleep 1
    for db in fasting-bot.db whatsapp-session.db; do
      [ -f "/opt/fasting-bot/$db" ] && cp "/opt/fasting-bot/$db" "$BACKUP_DIR/$db.bak-$(date +%Y%m%d-%H%M%S)"
    done
    sudo systemctl start fasting-bot
    echo "[$(date '+%F %T')] Backup done" >> "$LOG"
    ;;
  *)
    echo "Usage: $0 {healthcheck|backup}"
    exit 1
    ;;
esac