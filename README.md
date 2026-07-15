# 🤖 Fasting Bot

Bot WhatsApp untuk reminder fasting/IF (Intermittent Fasting) dengan notifikasi otomatis.

## Fitur

- ⏰ Notifikasi otomatis saat fasting mulai dan berakhir
- 📱 Bisa digunakan di grup
- 🗄️ Database SQLite (ringan, tanpa server)
- 📋 3 perintah utama: `/puasa`, `/buka`, `/batalkan`
- 📋 Pendukung: `/daftar`, `/panduan`, `/pemula`, `/status`, `/motivasi`, `/stats`, `/badge`, `/leaderboard`

## Struktur Project (Clean Architecture)

```text
fasting-bot/
├── cmd/fasting-bot/              # Entry point (dependency injection)
│   └── main.go
├── internal/
│   ├── config/                   # Konfigurasi aplikasi
│   │   └── config.go
│   ├── domain/                   # Entities / business objects
│   │   └── entities.go           # User, FastingSchedule, NotificationLog
│   ├── repository/               # Data access interfaces (contracts)
│   │   └── interfaces.go         # UserRepository, ScheduleRepository, NotificationRepository
│   ├── usecase/                  # Business logic
│   │   └── fasting_usecase.go    # FastingUsecase interface + implementation
│   ├── infrastructure/           # External implementations
│   │   ├── database/
│   │   │   └── sqlite.go         # SQLite connection + migrations
│   │   ├── persistence/          # Repository implementations
│   │   │   ├── user_repository_sqlite.go
│   │   │   ├── schedule_repository_sqlite.go
│   │   │   └── notification_repository_sqlite.go
│   │   └── whatsapp/
│   │       ├── client.go         # WhatsApp client wrapper
│   │       └── notifier.go       # WhatsApp message sender
│   └── delivery/                 # Interface adapters (handlers)
│       └── whatsapp/
│           ├── command_handler.go  # Command parser + handler
│           └── scheduler.go        # Cron job notifikasi
├── go.mod
└── README.md
```

## Prinsip Clean Architecture

| Layer | Tujuan | Contoh |
|---|---|---|
| **Domain** | Pure business entities, no external deps | `User`, `FastingSchedule` structs |
| **Repository** | Interfaces/contracts for data access | `UserRepository`, `ScheduleRepository` |
| **Usecase** | Business logic, orchestrates repositories | `RegisterUser`, `SetSchedule`, `GetStatus` |
| **Infrastructure** | Implements repositories + external services | SQLite repos, WhatsApp client |
| **Delivery** | Handles incoming messages/events | WhatsApp command handler, scheduler |

Dependency direction: **Delivery → Usecase → Repository → Domain**

Domain tidak bergantung pada layer lainnya.

## Setup Lokal

### Prasyarat

- Go 1.22+
- SQLite3
- Nomor WhatsApp untuk bot (isi di `.env`)

### 1. Install Dependencies

```bash
cd fasting-bot
go mod tidy
```

### 2. Jalankan Bot

```bash
go run ./cmd/fasting-bot
```

Atau build binary:
```bash
go build -o fasting-bot ./cmd/fasting-bot
./fasting-bot
```

### 3. Scan QR Code

Saat pertama kali running, bot akan menampilkan **QR code di terminal**:

```
📱 No session found. Please scan the QR code below:
   (If QR code doesn't appear, make sure your terminal supports Unicode)

📲 Scan this QR code with WhatsApp:
   WhatsApp → Settings → Linked Devices → Link a Device

█████████████████████████████████████████████
█████████████████████████████████████████████
████ ▄▄▄▄▄ █▀▄▄▀▄▀▄▀▀▄▀▀▄▄▀▀▄▀▄▄▄▀▄▄▄▄▄ ████
... (QR code akan muncul di terminal)
█████████████████████████████████████████████
```

**Cara scan:**
1. Buka WhatsApp di HP (nomor bot: sesuai `.env` kamu)
2. Pergi ke: **Settings → Linked Devices → Link a Device**
3. Arahkan kamera HP ke QR code di terminal
4. Tunggu hingga muncul "✅ Login successful!"

Session akan tersimpan di path `SESSION_PATH`, jadi tidak perlu scan QR tiap kali run. Untuk production, arahkan `DATABASE_PATH` dan `SESSION_PATH` ke file berbeda di folder data yang permission-nya ketat, misalnya `/opt/fasting-bot/data`. Jika QR perlu direset, hapus hanya `SESSION_PATH`; jangan hapus `DATABASE_PATH` karena file itu menyimpan user, jadwal, `/stats`, dan `/leaderboard`.

> Security: isi `ALLOWED_GROUP_JID` supaya command hanya diproses dari grup yang dipercaya. Balasan command selalu dikirim ke grup; chat pribadi dari bot ke nomor user hanya dipakai untuk notifikasi otomatis mulai/selesai puasa.

### 4. Testing

**Test di grup yang JID-nya sesuai `ALLOWED_GROUP_JID`:**
1. Invite bot ke grup (dari HP pribadi)
2. Kirim command di grup:
```
/daftar kyomel
/panduan
/pemula
/puasa 16
/status
/batalkan
```

**Test /panduan dan /puasa:**
```
/panduan
/puasa 16
/status
/batalkan
```

**Test notifikasi otomatis:**
- Set jadwal 1-2 menit dari waktu sekarang
- Tunggu bot kirim notifikasi otomatis

## Daftar Command

| Command | Deskripsi | Contoh |
|---|---|---|
| `/daftar <nama>` | Daftar sebagai user | `/daftar kyomel` |
| `/setname <nama>` | Ubah nama user | `/setname kyomel baru` |
| `/panduan` | Panduan edukatif + daftar preset | `/panduan` |
| `/pemula` | Panduan IF bertahap dari 12 jam | `/pemula` |
| `/puasa [durasi] [jam] [tanggal jam]` | Mulai/jadwalkan puasa. 1 arg = sekarang, 2 = jam, 3 = tgl+jam | `/puasa 16`, `/puasa 16 05:00`, `/puasa 16 14-06-2026 19:30` |
| `/puasa-dry [durasi] [jam] [tanggal jam]` | Mulai/jadwalkan dry fasting, maks 48 jam | `/puasa-dry 18`, `/puasa-dry 18 14-06-2026 08:00` |
| `/water-24`, `/water-36`, `/water-48`, `/water-72` | Preset cepat water fasting | `/water-48` |
| `/dry-24` | Preset cepat dry fasting | `/dry-24` |
| `/if-1212`, `/if-1410`, `/if-168`, `/if-186`, `/if-204` | Preset IF | `/if-168` |
| `/omad` | Preset OMAD (22 jam) | `/omad` |
| `/status` | Cek status + fase metabolik saat ini | `/status` |
| `/fase` | Tahapan metabolik (fed → deep fast) + posisi sekarang | `/fase` |
| `/motivasi` | Suntikan semangat sesuai fase metabolik | `/motivasi` |
| `/buka [tanggal] [jam]` | Catat buka sekarang (tanpa arg) atau waktu lampau | `/buka`, `/buka 23-05-2026 18:30` |
| `/batalkan` | Batalkan jadwal puasa aktif | `/batalkan` |
| `/stats` | Statistik puasa pribadi | `/stats` |
| `/riwayat [n]` | Riwayat sesi terakhir (default 5, maks 10) | `/riwayat`, `/riwayat 7` |
| `/badge` | Koleksi badge & achievement | `/badge` |
| `/leaderboard` | Klasemen puasa grup | `/leaderboard` |
| `/bantuan` | Bantuan command | `/bantuan` |
| `/info` | Info bot | `/info` |

## Jenis-Jenis Puasa

Semua jenis via `/puasa <durasi>` atau preset cepat:

| Jenis | Durasi | Preset | Manual |
|---|---|---|---|
| IF 12:12 | 12 jam | `/if-1212` | `/puasa 12` |
| IF 14:10 | 14 jam | `/if-1410` | `/puasa 14` |
| IF 16:8 | 16 jam | `/if-168` | `/puasa 16` |
| IF 18:6 | 18 jam | `/if-186` | `/puasa 18` |
| IF 20:4 | 20 jam | `/if-204` | `/puasa 20` |
| OMAD | 22 jam | `/omad` | `/puasa 22` |
| Water 24-72 jam | 24/36/48/72 | `/water-24`…`/water-72` | `/puasa <durasi>` |
| Dry 1-48 jam | 1-48 | `/dry-24` | `/puasa-dry <durasi>` |

Mau >72 jam? `/puasa <durasi>` sampai 168 jam (7 hari). Hanya untuk berpengalaman.

### Cara Menggunakan

1. Daftar: `/daftar <nama>`
2. Lihat panduan: `/panduan` atau `/pemula` (bertahap)
3. Mulai puasa:
   - `/puasa 16` → 16 jam dari sekarang
   - `/puasa 16 05:00` → 16 jam mulai jam 5
   - `/puasa 16 14-06-2026 19:30` → jadwalkan
   - `/puasa-dry 18` → dry 18 jam
   - Preset: `/if-168`, `/omad`, `/water-48`, dll
   - Preset + waktu: `/if-168 19:30`
4. Cek status: `/status`
5. Buka puasa: `/buka` (sekarang) atau `/buka DD-MM-YYYY HH:MM` (lampau)
6. Statistik: `/stats`, `/leaderboard`
7. Batalkan: `/batalkan`

Catatan:
- Format jam: `HH:MM`, tanggal: `DD-MM-YYYY`
- `/puasa 1 arg` = sekarang. `/puasa 2 arg` = jam (lewat → besok). `/puasa 3 arg` = tgl+jam
- `/buka` tanpa arg = buka sekarang (v2)
- Notifikasi otomatis saat mulai & selesai
- Streak +1 setiap `/buka` tepat waktu; reset setelah 24 jam idle
- Riwayat mentah dibersihkan tiap 3 hari; ringkasan `/stats` permanen

## Menambah Fitur Baru

Dengan Clean Architecture, menambah fitur baru menjadi mudah:

1. **Tambah entity** di `internal/domain/entities.go`
2. **Tambah repository interface** di `internal/repository/interfaces.go`
3. **Implement repository** di `internal/infrastructure/persistence/`
4. **Tambah usecase method** di `internal/usecase/fasting_usecase.go`
5. **Tambah command handler** di `internal/delivery/whatsapp/command_handler.go`

Contoh: Menambah fitur riwayat fasting
- Tambah `FastingHistory` entity
- Buat `HistoryRepository` interface
- Implement `HistoryRepositorySQLite`
- Tambah `GetHistory()` di usecase
- Tambah `/riwayat` command di handler

## Troubleshooting

### Bot tidak bisa connect
- Pastikan nomor bot sudah terdaftar di WhatsApp
- Cek koneksi internet
- Hapus file di `SESSION_PATH` dan scan QR ulang

### QR code tidak muncul / tidak bisa di-scan
- Pastikan terminal support Unicode (gunakan Terminal bawaan Mac/Linux, iTerm2, atau Windows Terminal)
- Jika QR code muncul sebagai string acak, coba resize terminal window lebih besar
- Jika QR code expired (timeout), bot akan generate otomatis yang baru
- Pastikan kamera HP bersih dan cukup terang saat scan

### Database error
- Hapus file di `DATABASE_PATH` untuk reset database (hati-hati, data hilang!)
- Pastikan folder writable

## Reset Data

```bash
# Hapus database (semua data user & jadwal terhapus!)
rm /opt/fasting-bot/data/fasting-bot.db

# Reset session/QR saja (progress tetap aman)
sudo /opt/fasting-bot/monitor.sh reset-session
```

Untuk production, jangan reset QR dengan menghapus seluruh folder `/opt/fasting-bot/data`. Struktur aman:

```text
/opt/fasting-bot/data/
  fasting-bot.db          # data permanen: users, schedules, stats, leaderboard
  whatsapp-session.db     # session WhatsApp, boleh dihapus untuk scan QR ulang
  backups/                # backup fasting-bot.db
```

Backup harian bisa dijalankan dengan:

```bash
sudo /opt/fasting-bot/monitor.sh backup
```

Restore dari backup terbaru:

```bash
sudo /opt/fasting-bot/monitor.sh restore
```

Untuk skala kecil, backup lokal di folder `backups/` cukup dulu untuk mencegah progress hilang saat reset QR atau salah hapus DB. Sync ke S3/R2/Google Drive/rsync bisa ditambahkan nanti jika datanya sudah makin penting atau user bertambah.

## Catatan Penting

- Bot menggunakan **unofficial WhatsApp Web API** (whatsmeow)
- Jangan gunakan untuk spam atau bulk messaging
- Ideal untuk grup kecil (< 50 orang)
- Untuk production skala kecil, jalankan backup `fasting-bot.db` rutin ke folder lokal `backups/`; sinkronisasi ke storage di luar VPS bisa ditambahkan nanti
