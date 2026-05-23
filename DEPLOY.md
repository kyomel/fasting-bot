# 🚀 CI/CD Deployment Guide

## Overview

This project uses **GitHub Actions** for automated deployment to a DomaiNesia VPS.

**Trigger**: Push ke branch `main` → Build binary → Deploy ke VPS → Restart service

## 🏗️ Arsitektur Deployment

```
Git Push (main)
    ↓
GitHub Actions
    ├── Build: Go binary (CGO enabled, Linux amd64)
    ├── Deploy: SCP ke VPS
    └── Restart: systemd service
    ↓
VPS DomaiNesia (Ubuntu 22.04)
    ├── fasting-bot binary
    ├── systemd service (auto-restart)
    ├── SQLite app database (users, stats, leaderboard)
    ├── WhatsApp session database
    └── backup/restore helper
```

## 📋 Prerequisites

### 1. VPS DomaiNesia Setup

Sebelum CI/CD jalan, setup VPS terlebih dahulu:

```bash
# SSH ke VPS
ssh root@<VPS_IP>

# Update system
apt update && apt upgrade -y

# Install utilities
apt install -y curl wget git htop nano sqlite3

# Buat user untuk bot
useradd --system --home /opt/fasting-bot --shell /bin/bash fastingbot

# Buat direktori aplikasi
mkdir -p /opt/fasting-bot/data
chown -R fastingbot:fastingbot /opt/fasting-bot

# Setup firewall
apt install -y ufw
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw --force enable
```

### 2. GitHub Secrets

Tambahkan secrets di **Settings → Secrets and variables → Actions**:

| Secret | Description | Contoh |
|--------|-------------|--------|
| `VPS_IP` | IP address VPS DomaiNesia | `103.123.456.78` |
| `VPS_USER` | SSH user (bukan root!) | `fastingbot` atau `deploy` |
| `VPS_SSH_KEY` | Private key SSH | `-----BEGIN OPENSSH PRIVATE KEY-----...` |

**Cara generate SSH key**:

```bash
# Di local machine (Mac/Linux)
ssh-keygen -t ed25519 -f ~/.ssh/fasting-bot-deploy

# Copy public key ke VPS
ssh-copy-id -i ~/.ssh/fasting-bot-deploy.pub fastingbot@<VPS_IP>

# Copy private key ke GitHub secrets (isi penuh)
cat ~/.ssh/fasting-bot-deploy
# → Copy isi file ke secret VPS_SSH_KEY
```

### 3. Environment File

Buat `.env` file di VPS (manual setup pertama kali):

```bash
# SSH ke VPS
cat > /opt/fasting-bot/.env << 'EOF'
BOT_NUMBER=628xxxxxxxxxx
ADMIN_NUMBER=628xxxxxxxxxx
ALLOWED_GROUP_JID=120xxxxxxxxxx@g.us
GROUP_NAME=Fasting Group
DATABASE_PATH=/opt/fasting-bot/data/fasting-bot.db
SESSION_PATH=/opt/fasting-bot/data/whatsapp-session.db
QR_CODE_PATH=
QR_CODE_HOST=
APP_TIMEZONE=Asia/Jakarta
EOF

chown fastingbot:fastingbot /opt/fasting-bot/.env
chmod 600 /opt/fasting-bot/.env
```

## 🔄 Deployment Flow

### Automatic (Push to main)

1. Push code ke `main` branch
2. GitHub Actions otomatis:
   - Build binary dengan CGO untuk Linux
   - Upload ke VPS via SCP
   - Restart systemd service
   - Verify service running

### Manual Trigger

Bisa juga trigger manual via GitHub UI:
- Go to **Actions → Deploy to DomaiNesia VPS → Run workflow**
- Pilih environment: `production`

## 📁 File Deployment

File yang dideploy otomatis:

| File | Lokasi VPS | Purpose |
|------|-----------|---------|
| `fasting-bot-linux` | `/opt/fasting-bot/fasting-bot` | Binary utama |
| `.env.example` | `/opt/fasting-bot/.env.example` | Template config |
| `deploy/fasting-bot.service` | `/etc/systemd/system/fasting-bot.service` | Systemd config |
| `deploy/monitor.sh` | `/opt/fasting-bot/monitor.sh` | Healthcheck, backup, restore, reset session |

## 🔧 Monitoring & Maintenance

### Cek Status Bot

```bash
# Di VPS
sudo systemctl status fasting-bot
sudo journalctl -u fasting-bot -f
```

### Healthcheck (setiap 5 menit via cron)

```bash
# Bot auto-heal jika crash
crontab -l
# → */5 * * * * /opt/fasting-bot/monitor.sh healthcheck
```

### Backup App DB Harian (jam 3 pagi)

```bash
# Backup user, stats, jadwal aktif, dan leaderboard otomatis
crontab -l
# → 0 3 * * * /opt/fasting-bot/monitor.sh backup
```

Backup memakai SQLite Online Backup API lewat `sqlite3 .backup`, jadi tidak perlu menghentikan bot. File yang dibuat:

```bash
/opt/fasting-bot/data/backups/fasting-bot-YYYYMMDD-HHMMSS.db
/opt/fasting-bot/data/backups/fasting-bot-YYYYMMDD-HHMMSS.sql.gz
```

Yang dibackup hanya `DATABASE_PATH` (`fasting-bot.db`), karena berisi data permanen:

- `users`
- `fasting_schedules`
- `fasting_records`
- `user_fasting_stats` untuk `/stats` dan `/leaderboard`
- `notification_logs`

`SESSION_PATH` (`whatsapp-session.db`) sengaja tidak dibackup rutin. Session WhatsApp boleh dihapus untuk scan QR ulang, sedangkan data puasa tidak boleh ikut terhapus.

Untuk skala kecil, backup lokal di `/opt/fasting-bot/data/backups` sudah cukup sebagai proteksi dari salah hapus DB saat reset QR atau deploy. Nanti jika user dan data sudah makin banyak, baru pertimbangkan sync folder backup ini ke storage lain seperti S3/R2/Google Drive atau server kedua.

### Restore Jika Database Hilang

```bash
# Restore dari backup terbaru
sudo /opt/fasting-bot/monitor.sh restore

# Atau restore dari file tertentu
sudo /opt/fasting-bot/monitor.sh restore /opt/fasting-bot/data/backups/fasting-bot-YYYYMMDD-HHMMSS.db
sudo /opt/fasting-bot/monitor.sh restore /opt/fasting-bot/data/backups/fasting-bot-YYYYMMDD-HHMMSS.sql.gz
```

Restore akan menghentikan service, menyimpan salinan DB lama sebagai `pre-restore-*.db`, menghapus `fasting-bot.db-wal`/`fasting-bot.db-shm`, lalu menyalakan service lagi.

### Reset QR / Session WhatsApp yang Aman

Jika hanya ingin scan QR ulang, jangan hapus folder data dan jangan hapus `fasting-bot.db`.

```bash
sudo /opt/fasting-bot/monitor.sh reset-session
```

Command ini hanya menghapus:

```bash
/opt/fasting-bot/data/whatsapp-session.db
/opt/fasting-bot/data/whatsapp-session.db-wal
/opt/fasting-bot/data/whatsapp-session.db-shm
```

Progress user, `/stats`, dan `/leaderboard` tetap aman di `fasting-bot.db`.

## 🛠️ Troubleshooting

### Deployment Failed

```bash
# Cek GitHub Actions logs
# → Actions tab → Workflow run → View logs

# Common issues:
# 1. SSH key tidak valid → regenerate dan update secret
# 2. VPS tidak reachable → cek IP dan firewall
# 3. Permission denied → pastikan user bisa sudo tanpa password
```

### Bot Tidak Berjalan Setelah Deploy

```bash
# SSH ke VPS dan cek
sudo systemctl status fasting-bot
sudo journalctl -u fasting-bot -n 50

# Pastikan .env file ada dan permission aman
ls -la /opt/fasting-bot/.env
stat /opt/fasting-bot/.env

# Pastikan binary executable
ls -la /opt/fasting-bot/fasting-bot
file /opt/fasting-bot/fasting-bot
```

## 📊 Workflow File

Workflow lengkap ada di `.github/workflows/deploy.yml`.

**Features**:
- ✅ Cross-compile dengan CGO (sqlite3 support)
- ✅ Artifact upload/download (secure)
- ✅ SSH deployment dengan known_hosts
- ✅ Automatic systemd reload dan restart
- ✅ Post-deployment verification
- ✅ Artifact cleanup
