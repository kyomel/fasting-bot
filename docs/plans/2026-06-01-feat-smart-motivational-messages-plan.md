---
title: "Smart Motivational Messages (/motivasi)"
type: feat
date: 2026-06-01
status: draft
---

# ✨ Smart Motivational Messages — `/motivasi`

> Bot mengirim pesan motivasi yang relevan dengan kondisi puasa user — berbasis
> fase metabolisme tubuh dan event puasa. **Template-based, tanpa AI** (integrasi
> AI di-_hold_ sampai ada provider gratis). Bahasa universal, ilmiah, tanpa embel-embel agama.

## Overview

Tambahkan command `/motivasi` yang, saat dipanggil, membaca jadwal puasa aktif user,
menghitung berapa lama ia sudah berpuasa, memetakan ke **fase metabolisme**, lalu
membalas pesan motivasi yang kontekstual dan menyemangati. Tujuannya: dukungan
emosional + edukasi ringan supaya user makin semangat menjalani Intermittent Fasting
untuk kesehatan.

Plan ini dibagi dua fase:

- **Fase 1 (MVP — kerjakan dulu):** Command `/motivasi` on-demand + katalog pesan +
  model fase metabolisme reusable di `domain/`.
- **Fase 2 (follow-up, opsional):** Pesan **proaktif** via scheduler — saat user masuk
  fase Fat Burning, saat hampir putus streak, saat selesai extended fast, reminder hidrasi.

## Problem Statement / Motivation

Bot sekarang mengirim notifikasi otomatis hanya di dua titik: **mulai** dan **waktunya
buka**. Di antara dua titik itu — saat user sedang berjuang melewati rasa lapar jam ke-14,
atau saat ia ragu apakah puasanya "ada gunanya" — bot diam. Padahal di situlah dukungan
paling dibutuhkan.

Riset adherence puasa (Mattson, Panda, Longo) konsisten menunjukkan: yang membuat orang
bertahan bukan sekadar tahu *apa* yang harus dilakukan, tapi *merasa progress-nya nyata*.
Memberi tahu user "tubuhmu sekarang sedang membakar lemak / sel-selmu sedang autophagy"
mengubah rasa lapar dari "penderitaan tanpa makna" jadi "bukti tubuhku sedang bekerja".

`/motivasi` memberi user tombol untuk memanggil dukungan itu kapan pun ia butuh.

## Proposed Solution

Mengikuti Clean Architecture yang sudah ada:

1. **Domain** — `MetabolicPhase` sebagai komponen reusable: mapping `jam berpuasa → fase`
   (Fed → Post-Absorptive → Fat Burning → Ketosis → Deep Fast). Fitur `/fase` (ide #1)
   nanti reuse ini tanpa refactor.
2. **Domain** — katalog pesan motivasi (pool per fase & per event) sebagai data murni,
   gampang ditambah/diedit tanpa menyentuh logika.
3. **Usecase** — method baru `GetMotivation(phone)` di `FastingUsecase`: lookup user →
   cek jadwal aktif → hitung elapsed → tentukan konteks (pre-start / fasting+fase /
   near-target / target-tercapai / belum ada jadwal) → pilih pesan.
4. **Delivery** — route `/motivasi` di `command_handler.go` + tambah ke `/bantuan`.

Logika penentuan fase & konteks **pure & unit-testable**; pemilihan pesan acak ditipiskan
agar tidak mengganggu test.

### Pemetaan Fase Metabolisme

Mengikuti tabel di `.omo/drafts/fasting-bot-feature-ideas.md` (ide #1):

| Fase | Rentang | Emoji | Inti pesan |
|------|---------|-------|------------|
| **Fed State** | 0–4 jam | 🍽️ | Pencernaan aktif, tubuh memproses makanan |
| **Post-Absorptive** | 4–12 jam | ⏳ | Glikogen mulai dipakai, insulin turun |
| **Fat Burning** | 12–18 jam | 🔥 | Metabolic switch — lemak jadi bahan bakar utama |
| **Ketosis** | 18–24 jam | ⚡ | Produksi keton naik, otak dapat bahan bakar bersih |
| **Deep Fast** | 24+ jam | 🧬 | Autophagy — sel membersihkan & mendaur ulang dirinya |

> **Catatan tipe puasa:** Untuk **Dry Fasting**, pesan **tidak boleh** menyarankan minum
> air. Untuk **Water/Prolonged Fasting** durasi panjang (≥24 jam), sisipkan pengingat
> elektrolit (Na/K/Mg), konsisten dengan pesan yang sudah ada di `scheduler.go`.

## Technical Considerations

- **Arsitektur:** Logika fase di `domain/` (reusable, pure). Logika konteks di `usecase/`
  (mirror `GetStatus`). String route di `delivery/`. Konsisten dengan struktur saat ini.
- **Reuse:** `GetMotivation` reuse helper yang sudah ada — `lookupUser`,
  `lookupActiveSchedule`, `parseScheduleTime`, `formatDuration`. Tidak menduplikasi parsing waktu.
- **Variasi pesan:** Saat dipanggil berkali-kali, jangan balas pesan yang sama terus.
  Pakai `math/rand` (aman di runtime Go — beda dengan batasan Workflow). Seed sekali di
  package init. Agar tetap testable, fungsi penentu fase/konteks dipisah dari pemilihan acak.
- **Tidak ada perubahan skema DB di Fase 1.** Fase 2 cukup memakai tabel `notification_log`
  yang sudah ada (kolom `NotificationType`) untuk dedup notifikasi proaktif.
- **Bahasa & tone:** Indonesian, ilmiah-tapi-hangat, sekuler. Mirror referensi yang sudah
  dipakai (Mattson, Ohsumi/Nobel 2016, Panda, Longo, BDNF, HGH, keton/BHB, mTOR/AMPK).
- **Keamanan:** Tidak ada input bebas yang dieksekusi; hanya membaca state user sendiri.
  Otorisasi grup tetap lewat `isAuthorized` yang sudah ada.

---

## Fase 1 — MVP: Command `/motivasi` (kerjakan dulu)

### File yang dibuat / diubah

```
internal/domain/metabolic_phase.go        (BARU) — model fase reusable + mapping
internal/domain/motivation.go             (BARU) — katalog pesan (pool per fase & event)
internal/domain/metabolic_phase_test.go   (BARU) — unit test boundary fase
internal/usecase/fasting_usecase.go        (EDIT) — tambah GetMotivation + ke interface
internal/usecase/motivation_test.go        (BARU) — unit test konteks GetMotivation
internal/delivery/whatsapp/command_handler.go (EDIT) — route /motivasi + masuk /bantuan
```

### Pseudocode

#### `internal/domain/metabolic_phase.go`

```go
package domain

import "math"

type MetabolicPhase struct {
    Key      string  // "fat_burning"
    Name     string  // "Fat Burning"
    Emoji    string  // "🔥"
    MinHours float64 // inklusif
    MaxHours float64 // eksklusif; math.Inf(1) untuk fase terakhir
}

var MetabolicPhases = []MetabolicPhase{
    {Key: "fed",             Name: "Fed State",       Emoji: "🍽️", MinHours: 0,  MaxHours: 4},
    {Key: "post_absorptive", Name: "Post-Absorptive", Emoji: "⏳", MinHours: 4,  MaxHours: 12},
    {Key: "fat_burning",     Name: "Fat Burning",     Emoji: "🔥", MinHours: 12, MaxHours: 18},
    {Key: "ketosis",         Name: "Ketosis",         Emoji: "⚡", MinHours: 18, MaxHours: 24},
    {Key: "deep_fast",       Name: "Deep Fast",       Emoji: "🧬", MinHours: 24, MaxHours: math.Inf(1)},
}

// PhaseForElapsedHours memetakan jam berpuasa ke fase metabolisme.
func PhaseForElapsedHours(h float64) MetabolicPhase {
    if h < 0 { h = 0 }
    for _, p := range MetabolicPhases {
        if h >= p.MinHours && h < p.MaxHours {
            return p
        }
    }
    return MetabolicPhases[len(MetabolicPhases)-1]
}

// NextPhase mengembalikan fase berikutnya (untuk teaser "menuju ..."), false jika sudah terakhir.
func (p MetabolicPhase) NextPhase() (MetabolicPhase, bool) { /* cari index+1 */ }
```

#### `internal/domain/motivation.go`

```go
package domain

// Pool pesan per fase. Placeholder %s = nama user (opsional di sebagian pesan).
// Lihat Lampiran A untuk teks lengkap.
var motivationByPhase = map[string][]string{
    "fed":             { /* ... */ },
    "post_absorptive": { /* ... */ },
    "fat_burning":     { /* ... */ },
    "ketosis":         { /* ... */ },
    "deep_fast":       { /* ... */ },
}

// Pool pesan per event/konteks.
var motivationNoSchedule  []string // belum ada jadwal aktif
var motivationPreStart    []string // jadwal ada, belum mulai
var motivationNearTarget  []string // sisa <= 2 jam
var motivationTargetMet   []string // sudah lewat target, belum /buka

// Reminder hidrasi disisipkan terpisah agar bisa di-skip untuk Dry Fasting.
var hydrationNudges []string

func MotivationForPhase(key string) []string { return motivationByPhase[key] }
```

#### `internal/usecase/fasting_usecase.go` (tambah ke interface + impl)

```go
// di interface FastingUsecase:
GetMotivation(phone string) (string, error)

func (u *fastingUsecase) GetMotivation(phone string) (string, error) {
    user, err := u.lookupUser(phone)
    if err != nil { return "", err }
    if user == nil { return msgNotRegistered, nil }
    name := displayName(user) // helper kecil: Name kalau ada, else Phone

    schedule, err := u.lookupActiveSchedule(user.ID)
    if err != nil { return "", err }
    if schedule == nil {
        return decorate(pick(domain.MotivationNoSchedule()), name), nil
    }

    now := time.Now().In(config.Location)
    start, _ := parseScheduleTime(schedule.FastStart, now)
    end, _   := parseScheduleTime(schedule.FastEnd, now)
    if !end.After(start) { end = end.AddDate(0, 0, 1) }

    switch {
    case now.Before(start):
        return preStartMessage(name, start.Sub(now)), nil
    case !now.Before(end): // target tercapai, belum /buka
        return targetMetMessage(name, schedule, start, end), nil
    default: // sedang puasa
        elapsed := now.Sub(start)
        remaining := end.Sub(now)
        if remaining <= 2*time.Hour {
            return nearTargetMessage(name, remaining), nil
        }
        phase := domain.PhaseForElapsedHours(elapsed.Hours())
        body := pick(domain.MotivationForPhase(phase.Key))
        // sisipkan reminder hidrasi HANYA jika bukan Dry Fasting
        if !isDryFasting(schedule.FastingTypeName) {
            body += "\n\n" + pick(domain.HydrationNudges())
        }
        return composeFastingMessage(name, phase, elapsed, remaining, body), nil
    }
}
```

> `isDryFasting`: cek `strings.Contains(strings.ToLower(name), "dry fasting")`.
> `pick`: pilih elemen acak dari pool (`math/rand`), aman untuk pool kosong (fallback string default).

#### `internal/delivery/whatsapp/command_handler.go`

```go
case "/motivasi":
    return h.callUsecase(phone, "GetMotivation", func() (string, error) {
        return h.usecase.GetMotivation(phone)
    })
```

Plus tambahkan baris di `getHelpText()`:
`/motivasi — Minta suntikan semangat sesuai fase puasamu`
dan label error di `errorLabels`: `"GetMotivation": "mengambil pesan motivasi"`.

### Acceptance Criteria — Fase 1

- [ ] Command `/motivasi` terdaftar di `processCommand` dan muncul di `/bantuan`.
- [ ] User belum terdaftar → balasan arahkan `/daftar` (reuse `msgNotRegistered`).
- [ ] User terdaftar tanpa jadwal aktif → pesan ajakan mulai puasa (pool `NoSchedule`).
- [ ] Jadwal ada tapi belum mulai → pesan antisipasi + hitung mundur ke jam mulai.
- [ ] Sedang puasa → pesan sesuai fase yang benar (uji boundary 0/4/12/18/24 jam).
- [ ] Sisa ≤ 2 jam → pesan "near-target" ("tinggal sedikit lagi").
- [ ] Sudah lewat target, belum `/buka` → pesan selamat + ingatkan catat dengan `/buka`.
- [ ] **Dry Fasting tidak pernah menyarankan minum air.**
- [ ] Water/Prolonged ≥24 jam → menyertakan pengingat elektrolit.
- [ ] Panggilan berulang tidak selalu mengembalikan pesan identik (variasi pool).
- [ ] `PhaseForElapsedHours` punya unit test untuk tiap boundary.
- [ ] `go build ./...`, `go vet ./...`, dan `go test ./...` hijau.

---

## Fase 2 — Follow-up (opsional): Notifikasi Proaktif

> Dikerjakan **setelah** Fase 1 stabil. Tidak memblokir MVP.

Mengirim pesan motivasi otomatis lewat `scheduler.go` (cron `* * * * *` yang sudah ada,
atau cron baru) dengan **dedup** memakai `notification_log`.

### Trigger yang diusulkan

| Trigger | Kapan | Dedup key (`NotificationType`) |
|---------|-------|--------------------------------|
| Masuk fase Fat Burning | elapsed lewat 12 jam | `phase_fat_burning` |
| Masuk fase Ketosis | elapsed lewat 18 jam | `phase_ketosis` |
| Masuk fase Deep Fast | elapsed lewat 24 jam | `phase_deep_fast` |
| Hampir putus streak / dekat target | sisa ≤ 2 jam | `near_target` |
| Selesai extended fast | `/buka` durasi ≥ 24 jam (hook di `breakFasting`) | — (kirim langsung) |
| Reminder hidrasi | tiap N jam saat puasa, **skip Dry Fasting** | `hydration_<slot>` |

### File yang diubah — Fase 2

```
internal/repository/interfaces.go                         (EDIT) — query cari user per fase + cek log
internal/infrastructure/persistence/schedule_repository_sqlite.go (EDIT) — implementasi query
internal/delivery/whatsapp/scheduler.go                   (EDIT) — job baru + kirim + LogNotification
```

### Acceptance Criteria — Fase 2

- [ ] Tiap user menerima notifikasi fase **maksimal sekali** per sesi puasa (dedup via log).
- [ ] Notifikasi hidrasi **tidak** dikirim untuk Dry Fasting.
- [ ] Notifikasi "selesai extended fast" hanya untuk durasi ≥ 24 jam.
- [ ] Tidak ada notifikasi dobel saat cron berjalan tiap menit.

---

## Lampiran A — Draft Pesan (Fase 1)

> Catatan gaya: hangat, ilmiah, sekuler. `*...*` = bold WhatsApp. `%s` = nama user.
> Tiap pool punya beberapa varian untuk variasi. Ini draft awal — boleh diperkaya.

### 🍽️ Fed State (0–4 jam)

```
🍽️ *Baru mulai, %s — pondasi sedang dibangun.*

Tubuhmu lagi memproses makanan terakhir. Belum terasa apa-apa? Wajar. Tapi jam-jam
inilah yang menentukan: setiap menit insulinmu mulai turun pelan-pelan.

Tetap tenang, sibukkan diri. Rasa lapar belum tentu datang — dan kalaupun datang,
ia cuma sinyal hormon, bukan keadaan darurat. 💪
```

```
🍽️ *Langkah pertama sudah diambil, %s.*

Pencernaan masih bekerja, tapi jam biologismu sudah mulai berhitung. Yang berat
itu memulai — dan itu sudah kamu lewati. Sisanya tinggal membiarkan tubuh bekerja.
```

### ⏳ Post-Absorptive (4–12 jam)

```
⏳ *%s, mesin pembakar mulai panas.*

Glikogen (gula simpanan) di hatimu mulai dipakai sebagai energi. Insulin turun,
dan tubuhmu pelan-pelan membuka akses ke cadangan lemak.

Ini fase transisi — sebentar lagi *metabolic switch* yang bikin Mark Mattson (NIH)
terkenal itu akan menyala. Kamu di jalur yang benar. 🔥
```

```
⏳ *Sabar, %s — yang seru sebentar lagi.*

Tubuhmu lagi menghabiskan bahan bakar "mudah" (glukosa) sebelum beralih ke lemak.
Lapar yang muncul sekarang itu gelombang ghrelin — puncaknya cuma ~20 menit, lalu
reda sendiri. Tarik napas 4-4-4, lewati gelombangnya.
```

### 🔥 Fat Burning (12–18 jam)

```
🔥 *Selamat, %s — tubuhmu sekarang membakar lemak!*

Glikogen sudah menipis, *metabolic switch* aktif: tubuhmu beralih dari gula ke lemak
sebagai bahan bakar utama. Ini momen yang nggak bisa dibeli di apotek mana pun —
kamu menciptakannya sendiri, jam demi jam.

Pertahankan. Kamu sudah lebih jauh dari yang banyak orang berani coba. 💪
```

```
🔥 *Ini dia, %s — zona pembakaran lemak.*

Setiap menit dari sekarang, lipolysis (pemecahan lemak) makin tinggi. Tubuhmu
mengakses cadangan yang biasanya terkunci. Rasa lapar mungkin datang-pergi —
itu normal, dan itu tanda mesinnya bekerja.
```

### ⚡ Ketosis (18–24 jam)

```
⚡ *Luar biasa, %s — kamu masuk ketosis.*

Hatimu mulai memproduksi keton (BHB) — bahan bakar bersih yang bahkan otak suka.
Banyak orang justru merasa *lebih fokus* di jam-jam ini. Keton juga sinyal
anti-inflamasi yang menenangkan tubuh.

Kamu nggak cuma menahan lapar — kamu sedang meng-upgrade cara tubuhmu beroperasi. ⚡
```

```
⚡ *%s, otakmu lagi dapat bahan bakar premium.*

Keton mulai menggantikan glukosa. Selain bikin fokus, proses ini menurunkan
inflamasi dan menyiapkan tubuh untuk fase pembersihan sel. Tinggal sedikit lagi
menuju autophagy. Kamu kuat. 🔥
```

### 🧬 Deep Fast (24+ jam)

```
🧬 *Hormat, %s — kamu di zona autophagy.*

Lewat 24 jam, tubuhmu menjalankan *autophagy* — proses daur ulang sel rusak yang
bikin Yoshinori Ohsumi dapat Nobel 2016. Hormon pertumbuhan (HGH) naik untuk
melindungi ototmu sementara lemak dibakar.

Ini bukan lagi soal disiplin — ini soal kualitas sel-selmu bertahun-tahun ke depan.
⚠️ Jaga elektrolit (garam, kalium, magnesium) dan dengarkan tubuhmu. 🧬
```

```
🧬 *%s, ini level yang sedikit orang capai.*

Di jam-jam ini sinyal regenerasi mulai menyala (Valter Longo, Cell 2014): sel imun
lama dibersihkan, IGF-1 turun ke zona longevity. Tubuhmu benar-benar "reset".
⚠️ Wajib hidrasi + elektrolit. Pantau pusing/lemas berat — kalau berlebihan, buka pelan-pelan.
```

### 🌱 Belum ada jadwal aktif (No Schedule)

```
🌱 *Siap kasih hadiah buat tubuhmu, %s?*

Belum ada puasa aktif. Padahal cukup 12-16 jam jeda makan untuk menyalakan
perbaikan sensitivitas insulin (riset Satchin Panda, Salk). Nggak perlu ekstrem —
mulai dari yang ringan.

• */list-puasa* — lihat 10 jenis puasa
• */set-puasa <nomor> <jam>* — mulai hari ini
Konsisten dikit-dikit, hasilnya luar biasa. 💪
```

### ⏳ Jadwal ada, belum mulai (Pre-Start)

```
⏳ *Bentar lagi mulai, %s!*

Puasamu mulai dalam %s. Pakai waktu ini buat makan yang bener: protein + lemak
biar kenyangnya tahan lama dan kurva insulinmu landai. Siapkan air putih juga.

Niat sudah dikunci — tinggal jalanin. Let's go! 🔥
```

### 🏁 Sisa ≤ 2 jam (Near-Target)

```
🏁 *Tinggal %s lagi, %s — JANGAN nyerah sekarang!*

Kamu sudah lewati bagian terberatnya. Dua jam terakhir ini justru saat tubuh paling
aktif membakar lemak dan membersihkan sel. Menyerah sekarang = berhenti tepat
sebelum garis finish.

Tarik napas. Minum air. Kamu sudah sejauh ini. Selesaikan! 💪🔥
```

### 🎉 Target tercapai, belum /buka (Target Met)

```
🎉 *Kamu BERHASIL, %s!*

Target puasamu sudah tercapai. Tubuhmu baru saja menyelesaikan kerja keras yang nyata —
banggalah. Sekarang catat biar masuk stats & streak:

• */buka* → buka sekarang
• */buka DD-MM-YYYY HH:MM* → kalau bukanya tadi
Buka pelan-pelan ya — protein & lemak dulu sebelum karbo. 🥗
```

### 💧 Reminder hidrasi (disisipkan, SKIP untuk Dry Fasting)

```
💧 Sudah minum air hari ini? Saat insulin turun, ginjal buang lebih banyak natrium —
tambah sejumput garam kalau puasamu >16 jam biar nggak lemas.
```

```
💧 Tetap terhidrasi: air putih, atau kopi/teh tanpa gula tetap aman. Dehidrasi sering
disangka lapar — segelas air dulu sebelum memutuskan.
```

---

## Success Metrics

- User memakai `/motivasi` lebih dari sekali (mengindikasikan terasa berguna).
- Tidak ada laporan pesan yang "salah fase" atau menyuruh Dry Faster minum air.
- Tone konsisten dengan notifikasi yang sudah ada (review manual).

## Dependencies & Risks

- **Tidak ada dependency baru** (tanpa AI/API eksternal). `math/rand` & `math` dari stdlib.
- **Risiko rendah:** fitur read-only terhadap state user; tidak mengubah skema di Fase 1.
- **Risiko tone:** pesan harus tetap sekuler & tidak menggurui — mitigasi: review draft
  Lampiran A bersama user sebelum implementasi.
- **Akurasi fase** bergantung pada `parseScheduleTime` (sudah dipakai `GetStatus`) — reuse,
  jangan bikin parser baru.

## References & Research

### Internal
- Ide fitur sumber: `.omo/drafts/fasting-bot-feature-ideas.md` (ide #5, dan #1 untuk mapping fase)
- Perhitungan elapsed/remaining existing: `internal/usecase/fasting_usecase.go:154` (`GetStatus`)
- Pola pesan ilmiah-sekuler existing: `internal/delivery/whatsapp/scheduler.go:168`
  (`buildNoFastersMessage`), `:328` (`fastingMilestonePreview`), `:395` (`buildStreakMessage`)
- Pola penambahan command: `internal/delivery/whatsapp/command_handler.go:136` (`processCommand`)
- Helper reuse: `lookupUser`, `lookupActiveSchedule`, `parseScheduleTime`, `formatDuration`
- Tabel tipe puasa (deteksi Dry/Water/Prolonged): `internal/domain/fasting_types.go:12`

### Konsep ilmiah yang dirujuk pesan (sudah dipakai di codebase)
- Metabolic switch — Mark Mattson (NIH)
- Autophagy — Yoshinori Ohsumi (Nobel 2016)
- Time-restricted eating — Satchin Panda (Salk Institute)
- IGF-1 / regenerasi stem cell — Valter Longo (Cell 2014)
- Keton/BHB, BDNF, HGH, mTOR/AMPK
