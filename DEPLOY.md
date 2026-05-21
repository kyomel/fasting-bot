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
    ├── SQLite databases
    └── WhatsApp session
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
APP_NAME=fasting-bot
APP_ENV=production
HTTP_ADDR=:3000
SQLITE_DSN=file:data/fasting.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on
BOT_TICK_INTERVAL=1m
SHUTDOWN_TIMEOUT=10s
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
# → */5 * * * * /opt/fasting-bot/healthcheck.sh
```

### Backup Harian (jam 3 pagi)

```bash
# Backup database otomatis
crontab -l
# → 0 3 * * * /opt/fasting-bot/backup.sh
```

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

# Pastikan .env file ada dan benar
ls -la /opt/fasting-bot/.env
cat /opt/fasting-bot/.env

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