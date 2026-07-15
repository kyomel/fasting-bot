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
	{ID: 8, Name: "Water Fasting", Description: "Puasa air - 24, 36, 48, 56, 64, atau 72 jam", FastHours: 0},
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
		"• 12:12 — 12 jam puasa, 12 jam makan (pemula absolut)\n" +
		"• 14:10 — 14 jam puasa, 10 jam makan (pemula)\n" +
		"• 16:8 — 16 jam puasa, 8 jam makan (populer)\n" +
		"• 18:6 — 18 jam puasa, 6 jam makan (intermediate)\n" +
		"• 20:4 — 20 jam puasa, 4 jam makan (advanced)\n\n" +
		"*2. OMAD (One Meal A Day)*\n" +
		"Satu kali makan dalam 24 jam. Puasa 22-23 jam.\n\n" +
		"*3. Water Fasting*\n" +
		"Hanya minum air (tanpa makanan) selama periode tertentu.\n" +
		"• 24 jam — autophagy mulai aktif\n" +
		"• 36 jam — masuk zona deep fast, elektrolit mulai penting\n" +
		"• 48 jam — regenerasi sel makin kuat\n" +
		"• 56 jam — naik bertahap setelah 48 jam, wajib pantau energi & tekanan tubuh\n" +
		"• 64 jam — step lanjutan sebelum 72 jam, refeed harus makin pelan\n" +
		"• 72 jam — batas atas bot untuk water fasting; refeed harus pelan\n\n" +
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
		"*Jadwalkan ke depan (tanggal + jam):*\n" +
		"• /puasa 16 14-06-2026 19:30\n" +
		"• /jadwal 16 20-06-2026 05:00\n\n" +
		"*Setelah puasa:*\n" +
		"• /buka — catat buka puasa\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		getPresetList()
}

func getPresetList() string {
	return "⚡ *Preset Cepat — dari Pemula sampai Advanced*\n\n" +
		"🌱 *IF Ringan (Pemula)*\n" +
		"/if-1212 — 12:12 (12 jam puasa)\n" +
		"   🧘 \"Tubuh mulai turunkan insulin & akses lemak perlahan — 12 jam sudah cukup perbaiki ritme sirkadian\"\n" +
		"/if-1410 — 14:10 (14 jam puasa)\n" +
		"   🌿 \"Sweet spot pemula: cukup tidur + sedikit ekstra = metabolic switch mulai aktif\"\n\n" +
		"🔥 *IF Klasik*\n" +
		"/if-168 — 16:8 (16 jam puasa)\n" +
		"   ⚡ \"Pola paling populer — autophagy mulai ramp-up, ketone (BHB) jadi bahan bakar otak\"\n" +
		"/if-186 — 18:6 (18 jam puasa)\n" +
		"   🧠 \"BDNF naik, fokus lebih tajam, insulin resistance mulai reset\"\n" +
		"/if-204 — 20:4 (20 jam puasa)\n" +
		"   💎 \"Warrior diet zone — HGH spike signifikan, fat-adaptation makin efisien\"\n\n" +
		"🍽️ *OMAD (One Meal A Day)*\n" +
		"/omad — 22 jam puasa\n" +
		"   🔋 \"Satu kali makan, 22 jam repair — tubuh dominan ketone sepanjang hari\"\n\n" +
		"💧 *Water Fasting*\n" +
		"/water-24 — 24 jam\n" +
		"   🧬 \"Glikogen 0%, autophagy puncak, tubuh full ketosis\"\n" +
		"/water-36 — 36 jam\n" +
		"   🔄 \"IGF-1 turun, sel masuk mode damage control (Valter Longo, USC)\"\n" +
		"/water-48 — 48 jam\n" +
		"/water-72 — 72 jam\n" +
		"   🏔️ \"Batas atas bot — imun reset menyeluruh, mitokondria diperbarui\"\n\n" +
		"⚠️ *Dry Fasting (berpengalaman)*\n" +
		"/dry-24 — 24 jam\n" +
		"   🔴 \"Tanpa air — maksimal 48 jam via /puasa-dry. Dengarkan tubuh, jangan paksa\"\n\n" +
		"💡 *Mau lebih dari 72 jam?*\n" +
		"Bot membatasi 168 jam (7 hari) via /puasa <durasi>.\n" +
		"Hanya untuk yang sudah terbiasa — konsultasi dokter dulu.\n" +
		"Format: /puasa 96 14-06-2026 06:00\n\n" +
		"✨ *Semua preset bisa diberi waktu:*\n" +
		"/if-168 19:30 — mulai jam 7:30 malam\n" +
		"/water-48 14-06-2026 20:00 — jadwalkan ke tanggal tertentu\n\n" +
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
		"• Command: /puasa 12 atau /if-1212\n\n" +
		"*Level 2 — 14:10 (1-2 minggu)*\n" +
		"• Naik kalau 12 jam sudah terasa biasa\n" +
		"• Cocok untuk membangun konsistensi tanpa lapar ekstrem\n" +
		"• Command: /puasa 14 atau /if-1410\n\n" +
		"*Level 3 — 16:8 (2-4 minggu)*\n" +
		"• Pola IF paling populer\n" +
		"• Tubuh biasanya mulai lebih terbiasa dengan jeda makan panjang\n" +
		"• Command: /puasa 16 atau /if-168\n\n" +
		"*Level 4 — 18:6 / 20:4 (opsional)*\n" +
		"• Naik hanya kalau tidur, energi, mood, dan olahraga tetap aman\n" +
		"• Jangan dipaksa setiap hari\n" +
		"• Command: /puasa 18, /if-186, /puasa 20, /if-204\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"✅ *Aturan Aman Saat IF*\n" +
		"• Saat puasa IF: air putih, teh tawar, atau kopi hitam tanpa gula biasanya aman\n" +
		"• Saat makan: prioritaskan protein, sayur/serat, lemak baik, dan karbo kompleks\n" +
		"• Hindari balas dendam makan; manfaat IF bisa hilang kalau overeating\n" +
		"• Tidur cukup — kurang tidur bikin lapar dan craving naik\n" +
		"• Kalau pusing berat, gemetar, hampir pingsan, bingung, atau lemas ekstrem: hentikan puasa\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"💡 *Semua command bisa dikasih waktu:*\n" +
		"• /puasa 16 19:30 — 16 jam mulai jam 7:30 malam\n" +
		"• /puasa 16 14-06-2026 05:00 — jadwalkan ke tanggal & jam tertentu\n" +
		"• /if-168 19:30 — preset /if-168 dengan jam mulai\n\n" +
		"━━━━━━━━━━━━━━━\n" +
		"💧 *Setelah IF Terbiasa*\n" +
		"*Water fasting* jangan jadi langkah awal. Coba hanya kalau IF 16:8 sudah stabil beberapa minggu. Mulai dari 24 jam dulu, tetap hidrasi, dan perhatikan elektrolit. Preset: */water-24*.\n\n" +
		"*Dry fasting* lebih ekstrem karena tanpa air. Tidak untuk pemula. Di bot ini dibatasi maksimal 48 jam dulu, dan sebaiknya hanya setelah pengalaman cukup serta kondisi tubuh aman.\n\n" +
		"Kunci: tambah durasi sedikit-sedikit. Kalau tubuh protes, turun level dulu. Konsisten > ekstrem. 💪"
}
