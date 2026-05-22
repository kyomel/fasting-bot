# 🤖 Fasting Bot

Bot WhatsApp untuk reminder fasting/IF (Intermittent Fasting) dengan notifikasi otomatis.

## Fitur

- ⏰ Notifikasi otomatis saat fasting mulai dan berakhir
- 📱 Bisa digunakan di grup maupun DM personal
- 🗄️ Database SQLite (ringan, tanpa server)
- 📋 Command sederhana: /daftar, /list-puasa, /set-puasa, /jadwalkan, /jadwal-bebas, /status, /buka, /hapus

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

Session akan tersimpan di path `SESSION_PATH`, jadi tidak perlu scan QR tiap kali run. Untuk production, arahkan `DATABASE_PATH` dan `SESSION_PATH` ke folder data yang permission-nya ketat, misalnya `/opt/fasting-bot/data`.

> Security: isi `ALLOWED_GROUP_JID` supaya command grup hanya diproses dari grup yang dipercaya. Command personal seperti `/daftar`, `/set-puasa`, `/status`, `/stats`, `/buka`, dan `/hapus` akan dibalas via DM agar nomor dan jadwal tidak terbuka di grup.

### 4. Testing

**Test di DM admin (`ADMIN_NUMBER`):**
```
/daftar kyomel
/list-puasa
/set-puasa 3 05:00
/status
/hapus
```

**Test di grup yang JID-nya sesuai `ALLOWED_GROUP_JID`:**
1. Invite bot ke grup (dari HP pribadi)
2. Kirim command di grup:
```
/daftar kyomel
/list-puasa
/set-puasa 3 05:00
/status
/hapus
```

**Test /list-puasa dan /set-puasa:**
```
/list-puasa
/set-puasa 3 05:00
/status
/hapus
```

**Test notifikasi otomatis:**
- Set jadwal 1-2 menit dari waktu sekarang
- Tunggu bot kirim notifikasi otomatis

## Daftar Command

| Command | Deskripsi | Contoh |
|---|---|---|
| `/daftar <nama>` | Daftar sebagai user. Jika nomor WhatsApp sudah terdaftar, bot akan menolak pendaftaran ulang | `/daftar kyomel` |
| `/setname <nama>` | Ubah nama user yang sudah terdaftar | `/setname kyomel baru` |
| `/list-puasa` | Lihat jenis-jenis puasa | `/list-puasa` |
| `/set-puasa <nomor> <jam> [durasi]` | Pilih jenis puasa dari daftar | `/set-puasa 3 05:00` |
| `/jadwalkan <nomor> <tanggal> <jam> [durasi]` | Seperti `/set-puasa`, tetapi memakai tanggal eksplisit. Boleh memakai waktu lampau untuk restore progres yang ter-reset. Tidak memakai kode WF/DF | `/jadwalkan 3 23-05-2026 16:00` |
| `/jadwal-bebas <WF\|DF> <tanggal> <jam> <durasi>` | Khusus Water/Dry Fasting freestyle dengan kode WF/DF | `/jadwal-bebas WF 23-05-2026 16:00 12` |
| `/status` | Cek status fasting, nama, nomor, ID user, jenis puasa, tanggal/jam mulai, tanggal/jam selesai, dan durasi puasa yang sedang berjalan | `/status` |
| `/buka` | Buka puasa / batalkan fasting. Jika puasa sudah mulai, durasi dicatat ke stats | `/buka` |
| `/hapus` | Hapus jadwal puasa aktif. Setelah dihapus, `/status` akan menampilkan belum ada jadwal fasting | `/hapus` |
| `/stats` | Lihat statistik hasil buka puasa pribadi | `/stats` |
| `/leaderboard` | Lihat klasemen puasa berdasarkan total waktu puasa | `/leaderboard` |
| `/help` | Tampilkan bantuan | `/help` |
| `/info` | Info bot | `/info` |

## Jenis-Jenis Puasa

Bot mendukung 10 jenis puasa yang bisa dipilih:

| No | Jenis | Durasi Puasa | Cara Set |
|---|---|---|---|
| 1 | IF 12:12 | 12 jam | `/set-puasa 1 05:00` |
| 2 | IF 14:10 | 14 jam | `/set-puasa 2 05:00` |
| 3 | IF 16:8 | 16 jam | `/set-puasa 3 05:00` |
| 4 | IF 18:6 | 18 jam | `/set-puasa 4 05:00` |
| 5 | IF 20:4 | 20 jam | `/set-puasa 5 05:00` |
| 6 | OMAD-1 | 22 jam | `/set-puasa 6 05:00` |
| 7 | OMAD-2 | 23 jam | `/set-puasa 7 05:00` |
| 8 | Water Fasting | 24/36/48/72 jam | `/set-puasa 8 05:00 48` |
| 9 | Dry Fasting | Bebas tentukan | `/set-puasa 9 05:00 18` |
| 10 | Prolonged Fasting (Bebas) | Metode water fasting, minimal 24 jam | `/set-puasa 10 05:00 96` |

### Cara Menggunakan

1. Lihat daftar: `/list-puasa`
2. Pilih jenis IF & OMAD (1-7): `/set-puasa <nomor> <jam_mulai>`
   - Contoh: `/set-puasa 3 05:00` → Puasa jam 05:00 - 21:00 (16 jam)
   - Contoh: `/set-puasa 6 05:00` → Puasa jam 05:00 - 03:00 (22 jam)
3. Pilih Water/Dry/Prolonged Fasting (8-10): `/set-puasa <nomor> <jam_mulai> <durasi_jam>`
   - Contoh: `/set-puasa 8 05:00 48` → Water Fasting 48 jam dari jam 05:00
   - Contoh: `/set-puasa 9 05:00 18` → Dry Fasting 18 jam dari jam 05:00
   - Contoh: `/set-puasa 10 05:00 96` → Prolonged Fasting metode water fasting 96 jam dari jam 05:00
4. Jadwalkan puasa dari daftar dengan tanggal khusus: `/jadwalkan <nomor> <tanggal> <jam_mulai> [durasi_jam]`
   - Contoh: `/jadwalkan 3 23-05-2026 16:00` → IF 16:8 dari 23-05-2026 16:00 sampai 24-05-2026 08:00
   - Contoh: `/jadwalkan 8 23-05-2026 16:00 48` → Water Fasting 48 jam dari 23-05-2026 16:00 sampai 25-05-2026 16:00
   - `/jadwalkan` selalu memakai nomor seperti `/set-puasa`. Jangan pakai `WF`/`DF` di command ini; Water Fasting pakai nomor 8, Dry Fasting pakai nomor 9.
   - `/jadwalkan` boleh memakai tanggal/jam yang sudah lewat untuk memulihkan progres setelah data aktif ter-reset. Jika mulai sudah lewat, notifikasi mulai tidak dikirim ulang.
5. Jadwalkan WF/DF freestyle dengan tanggal dan durasi bebas: `/jadwal-bebas <WF|DF> <tanggal> <jam_mulai> <durasi_jam>`
   - Contoh: `/jadwal-bebas WF 23-05-2026 16:00 12` → Water Fasting 12 jam dari 23-05-2026 16:00 sampai 24-05-2026 04:00
   - Contoh: `/jadwal-bebas DF 23-05-2026 20:00 10` → Dry Fasting 10 jam dari 23-05-2026 20:00 sampai 24-05-2026 06:00
6. Cek status jadwal: `/status`
   - Status menampilkan jenis puasa, tanggal/jam mulai, tanggal/jam selesai, dan jika sedang berjalan akan menampilkan sudah berjalan berapa lama.
7. Buka puasa: `/buka`
   - Jika puasa sudah mulai, bot mencatat total waktu puasa ke `/stats` dalam format hari, jam, dan menit.
   - Jika `/buka` dilakukan sebelum jam mulai puasa, jadwal dibatalkan tetapi durasi tidak dihitung.
8. Cek statistik dan klasemen: `/stats` atau `/leaderboard`
   - `/leaderboard` diurutkan berdasarkan total waktu puasa terbesar.
9. Hapus jadwal aktif jika ingin mengosongkan status: `/hapus`
   - Setelah `/hapus`, `/status` akan kembali menampilkan belum ada jadwal fasting.

Catatan waktu:
- Format tanggal untuk `/jadwalkan` adalah `DD-MM-YYYY`.
- Format tanggal untuk `/jadwal-bebas` adalah `DD-MM-YYYY`.
- `/jadwalkan` mengikuti format nomor `/set-puasa`: nomor 1-7 tanpa durasi, nomor 8-10 wajib durasi. Kode `WF`/`DF` hanya untuk `/jadwal-bebas`.
- `/jadwalkan` dan `/jadwal-bebas` boleh memakai tanggal/jam mulai yang sudah lewat untuk restore progres. Setelah itu gunakan `/buka` saat benar-benar berbuka supaya durasi masuk ke `/stats`.
- Jika `/set-puasa` memakai jam mulai yang sudah lewat hari ini, bot otomatis menjadwalkannya untuk besok.
- Streak puasa dihitung dari tanggal kalender lokal saat puasa berjalan. Jika ada satu hari kalender tanpa puasa berjalan, streak saat ini otomatis kembali ke 0 saat `/stats` atau `/leaderboard` dibuka.
- `/stats` hanya menghitung hasil puasa dari `/buka` setelah puasa dimulai.
- Progres total `/stats` dan `/leaderboard` disimpan di ringkasan permanen, sehingga riwayat mentah lama bisa dibersihkan tanpa mengurangi total user.
- Bot membersihkan riwayat mentah lama dan jadwal nonaktif lama setiap 3 hari agar database tetap ringan. Cleanup akan melewati user yang masih punya jadwal puasa aktif.
- Balasan bot dan `/status` menampilkan jenis puasa, tanggal dan jam mulai, serta tanggal dan jam selesai agar jadwal lebih mudah dipahami.

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

# Hapus session (perlu scan QR ulang)
rm /opt/fasting-bot/data/whatsapp-session.db
```

## Catatan Penting

- Bot menggunakan **unofficial WhatsApp Web API** (whatsmeow)
- Jangan gunakan untuk spam atau bulk messaging
- Ideal untuk grup kecil (< 50 orang)
- Untuk production, pertimbangkan backup database secara rutin
