package domain

import "fmt"

type FastingType struct {
	ID          int
	Name        string
	Description string
	FastHours   int
}

var FastingTypes = []FastingType{
	{ID: 1, Name: "IF 12:12", Description: "Puasa 12 jam, makan 12 jam", FastHours: 12},
	{ID: 2, Name: "IF 14:10", Description: "Puasa 14 jam, makan 10 jam", FastHours: 14},
	{ID: 3, Name: "IF 16:8", Description: "Puasa 16 jam, makan 8 jam", FastHours: 16},
	{ID: 4, Name: "IF 18:6", Description: "Puasa 18 jam, makan 6 jam", FastHours: 18},
	{ID: 5, Name: "IF 20:4", Description: "Puasa 20 jam, makan 4 jam", FastHours: 20},
	{ID: 6, Name: "OMAD-1", Description: "One Meal A Day - Puasa 22 jam", FastHours: 22},
	{ID: 7, Name: "OMAD-2", Description: "One Meal A Day - Puasa 23 jam", FastHours: 23},
	{ID: 8, Name: "Water Fasting", Description: "Puasa air - 24, 36, 48, atau 72 jam", FastHours: 0},
	{ID: 9, Name: "Dry Fasting", Description: "Puasa kering - maksimal 48 jam", FastHours: 0},
	{ID: 10, Name: "Prolonged Fasting (Bebas)", Description: "Puasa panjang metode water fasting, minimal 24 jam", FastHours: 0},
}

func GetFastingTypeByID(id int) (*FastingType, error) {
	for i := range FastingTypes {
		if FastingTypes[i].ID == id {
			return &FastingTypes[i], nil
		}
	}
	return nil, fmt.Errorf("jenis puasa tidak ditemukan")
}

func GetPanduan() string {
	return "📚 *Panduan Puasa*\n\n" +
		"*1. Intermittent Fasting (IF)*\n" +
		"Pola makan dengan jendela makan terbatas. Tubuh masuk mode \"repair\" saat insulin rendah.\n" +
		"• 14:10 — 14 jam puasa, 10 jam makan (pemula)\n" +
		"• 16:8 — 16 jam puasa, 8 jam makan (populer)\n" +
		"• 18:6 — 18 jam puasa, 6 jam makan (intermediate)\n" +
		"• 20:4 — 20 jam puasa, 4 jam makan (advanced)\n\n" +
		"*2. OMAD (One Meal A Day)*\n" +
		"Satu kali makan dalam 24 jam. Puasa 22-23 jam.\n\n" +
		"*3. Water Fasting*\n" +
		"Hanya minum air (tanpa makanan) selama periode tertentu.\n" +
		"• 24 jam — autophagy mulai aktif\n" +
		"• 36-48 jam — regenerasi sel\n" +
		"• 72 jam — imun reset (Valter Longo)\n\n" +
		"*4. Dry Fasting*\n" +
		"Tanpa makanan & tanpa minum. Risiko lebih tinggi — hanya untuk yang berpengalaman.\n" +
		"⚠️ Maksimal 48 jam dulu karena ini lebih ekstrem. Pantau sinyal tubuh: pusing, lemas, suhu.\n\n" +
		"*5. Prolonged Fasting*\n" +
		"Puasa >24 jam. Perlu persiapan & refeed pelan-pelan.\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"💡 *Cara Pakai Bot*\n\n" +
		"*Mulai sekarang:*\n" +
		"• /puasa 16 — 16 jam dari sekarang\n" +
		"• /puasa 16 05:00 — mulai jam 5, durasi 16 jam\n" +
		"• /puasa-dry 18 — dry fasting 18 jam\n\n" +
		"*Jadwalkan ke depan:*\n" +
		"• /jadwal 16 20-06-2026 05:00\n\n" +
		"*Preset cepat:*\n" +
		"• /if-168 = /puasa 16\n" +
		"• /if-186 = /puasa 18\n" +
		"• /if-204 = /puasa 20\n" +
		"• /omad = /puasa 22\n" +
		"• /water-48 = /puasa 48\n" +
		"• /dry-24 = /puasa-dry 24\n\n" +
		"*Setelah puasa:*\n" +
		"• /buka — catat buka puasa\n\n" +
		"Konsisten dikit-dikit, hasilnya luar biasa. 💪"
}

func GetPemulaGuide() string {
	return "🌱 *Panduan IF untuk Pemula*\n\n" +
		"Tujuan awal bukan langsung kuat lama, tapi bikin tubuh adaptasi tanpa stres. Banyak panduan medis menjelaskan IF sebagai pola makan berbasis waktu; 12 jam adalah titik awal yang paling ringan karena sebagian besar terjadi saat tidur.\n\n" +
		"⚠️ *Jangan mulai tanpa konsultasi dokter* kalau sedang hamil/menyusui, punya riwayat eating disorder, diabetes/obat gula darah, penyakit ginjal, asam urat berat, tekanan darah rendah, atau sedang sakit.\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"🪜 *Naik Level Pelan-Pelan*\n\n" +
		"*Level 1 — 12:12 (7 hari)*\n" +
		"• Puasa 12 jam, makan 12 jam\n" +
		"• Contoh: makan terakhir 20:00, makan lagi 08:00\n" +
		"• Command: /puasa 12\n\n" +
		"*Level 2 — 14:10 (1-2 minggu)*\n" +
		"• Naik kalau 12 jam sudah terasa biasa\n" +
		"• Cocok untuk membangun konsistensi tanpa lapar ekstrem\n" +
		"• Command: /puasa 14\n\n" +
		"*Level 3 — 16:8 (2-4 minggu)*\n" +
		"• Pola IF paling populer\n" +
		"• Tubuh biasanya mulai lebih terbiasa dengan jeda makan panjang\n" +
		"• Command: /puasa 16\n\n" +
		"*Level 4 — 18:6 / 20:4 (opsional)*\n" +
		"• Naik hanya kalau tidur, energi, mood, dan olahraga tetap aman\n" +
		"• Jangan dipaksa setiap hari\n" +
		"• Command: /puasa 18 atau /puasa 20\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"✅ *Aturan Aman Saat IF*\n" +
		"• Saat puasa IF: air putih, teh tawar, atau kopi hitam tanpa gula biasanya aman\n" +
		"• Saat makan: prioritaskan protein, sayur/serat, lemak baik, dan karbo kompleks\n" +
		"• Hindari balas dendam makan; manfaat IF bisa hilang kalau overeating\n" +
		"• Tidur cukup — kurang tidur bikin lapar dan craving naik\n" +
		"• Kalau pusing berat, gemetar, hampir pingsan, bingung, atau lemas ekstrem: hentikan puasa\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"💧 *Setelah IF Terbiasa*\n" +
		"*Water fasting* jangan jadi langkah awal. Coba hanya kalau IF 16:8 sudah stabil beberapa minggu. Mulai dari 24 jam dulu, tetap hidrasi, dan perhatikan elektrolit.\n\n" +
		"*Dry fasting* lebih ekstrem karena tanpa air. Tidak untuk pemula. Di bot ini dibatasi maksimal 48 jam dulu, dan sebaiknya hanya setelah pengalaman cukup serta kondisi tubuh aman.\n\n" +
		"Kunci: tambah durasi sedikit-sedikit. Kalau tubuh protes, turun level dulu. Konsisten > ekstrem. 💪"
}

func GetFastingTypesList() string {
	return GetPanduan()
}
