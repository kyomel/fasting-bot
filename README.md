# 🤖 Fasting Bot

Bot WhatsApp untuk reminder fasting/IF (Intermittent Fasting) dengan notifikasi otomatis.

## Fitur

- ⏰ Notifikasi otomatis saat fasting mulai dan berakhir
- 📱 Bisa digunakan di grup maupun DM personal
- 🗄️ Database SQLite (ringan, tanpa server)
- 📋 Command sederhana: /daftar, /jadwal, /status, /batal

## Nomor Bot

- **Bot:** +6285111334509
- **Admin:** +6282227026082
- **Grup:** Fasting Group

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
- Nomor WhatsApp untuk bot (+6285111334509)

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
1. Buka WhatsApp di HP (nomor bot: +6285111334509)
2. Pergi ke: **Settings → Linked Devices → Link a Device**
3. Arahkan kamera HP ke QR code di terminal
4. Tunggu hingga muncul "✅ Login successful!"

Session akan tersimpan di `whatsapp-session.db`, jadi tidak perlu scan QR tiap kali run.

### 4. Testing

**Test di DM (nomor pribadi kamu):**
```
/daftar
/jadwal 05:00 18:00
/status
```

**Test di grup "Fasting Group":**
1. Invite bot ke grup (dari HP pribadi)
2. Kirim command di grup:
```
/daftar
/jadwal 05:00 18:00
/status
```

**Test notifikasi otomatis:**
- Set jadwal 1-2 menit dari waktu sekarang
- Tunggu bot kirim notifikasi otomatis

## Daftar Command

| Command | Deskripsi | Contoh |
|---|---|---|
| `/daftar` | Daftar sebagai user | `/daftar` |
| `/jadwal HH:MM HH:MM` | Atur jadwal fasting | `/jadwal 05:00 18:00` |
| `/status` | Cek status fasting | `/status` |
| `/batal` | Batalkan fasting hari ini | `/batal` |
| `/help` | Tampilkan bantuan | `/help` |
| `/info` | Info bot | `/info` |

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
- Hapus `whatsapp-session.db` dan scan QR ulang

### QR code tidak muncul / tidak bisa di-scan
- Pastikan terminal support Unicode (gunakan Terminal bawaan Mac/Linux, iTerm2, atau Windows Terminal)
- Jika QR code muncul sebagai string acak, coba resize terminal window lebih besar
- Jika QR code expired (timeout), bot akan generate otomatis yang baru
- Pastikan kamera HP bersih dan cukup terang saat scan

### Database error
- Hapus `fasting-bot.db` untuk reset database (hati-hati, data hilang!)
- Pastikan folder writable

## Reset Data

```bash
# Hapus database (semua data user & jadwal terhapus!)
rm fasting-bot.db

# Hapus session (perlu scan QR ulang)
rm whatsapp-session.db
```

## Catatan Penting

- Bot menggunakan **unofficial WhatsApp Web API** (whatsmeow)
- Jangan gunakan untuk spam atau bulk messaging
- Ideal untuk grup kecil (< 50 orang)
- Untuk production, pertimbangkan backup database secara rutin
