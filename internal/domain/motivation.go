package domain

import "fmt"

const (
	NotificationTypePhaseFatBurning = "phase_fat_burning"
	NotificationTypePhaseKetosis    = "phase_ketosis"
	NotificationTypePhaseDeepFast   = "phase_deep_fast"
	NotificationTypeNearTarget      = "near_target"
)

type ProactivePhaseNotification struct {
	NotificationType  string
	TriggerAfterHours int
	PhaseKey          string
}

var ProactivePhaseNotifications = []ProactivePhaseNotification{
	{NotificationType: NotificationTypePhaseFatBurning, TriggerAfterHours: 12, PhaseKey: "fat_burning"},
	{NotificationType: NotificationTypePhaseKetosis, TriggerAfterHours: 18, PhaseKey: "ketosis"},
	{NotificationType: NotificationTypePhaseDeepFast, TriggerAfterHours: 24, PhaseKey: "deep_fast"},
}

var HydrationReminderHours = []int{8, 16, 20, 28, 32, 36, 40, 44, 48}

func HydrationNotificationType(hours int) string {
	return fmt.Sprintf("hydration_%dh", hours)
}

var motivationByPhase = map[string][]string{
	"fed": {
		"🍽️ Tubuhmu masih memproses energi dari makan terakhir. Ini fase pemanasan: insulin mulai turun pelan, dan kamu sedang membangun momentum.",
		"🌱 Awal yang bagus. Di fase ini pencernaan masih aktif, jadi fokusmu cukup satu: jaga ritme dan jangan buru-buru menyerah.",
	},
	"post_absorptive": {
		"⏳ Glikogen mulai dipakai, insulin makin rendah, dan tubuh mulai membuka akses ke cadangan energi. Progress kecil ini nyata.",
		"🧭 Kamu masuk fase transisi. Rasa lapar biasanya datang bergelombang — tunggu beberapa menit, gelombangnya sering turun sendiri.",
	},
	"fat_burning": {
		"🔥 Metabolic switch mulai aktif: tubuh makin nyaman memakai lemak sebagai bahan bakar. Lapar hari ini = sinyal adaptasi, bukan kegagalan.",
		"🔥 Ini zona pembakaran lemak. Setiap jam yang kamu tahan membantu tubuh belajar lebih fleksibel memakai energi.",
	},
	"ketosis": {
		"⚡ Produksi keton mulai naik. Otak mendapat bahan bakar bersih seperti BHB, dan banyak orang mulai merasa fokusnya lebih stabil.",
		"⚡ Kamu sudah jauh. Di rentang ini insulin rendah, ketogenesis naik, dan tubuh makin efisien mengelola energi.",
	},
	"deep_fast": {
		"🧬 Deep fast aktif. Sinyal AMPK naik, mTOR turun, dan autophagy membantu sel membersihkan komponen yang sudah aus.",
		"🧬 Ini fase serius: cellular cleanup makin dominan. Tetap tenang, pantau tubuh, dan buka pelan saat waktunya tiba.",
	},
}

var motivationNoSchedule = []string{
	"🌱 Belum ada jadwal puasa aktif. Mulai dari langkah kecil saja: pilih jenis puasa dengan /list-puasa, lalu set jadwal pertama. Konsistensi menang dari sempurna.",
	"🚀 Tubuh suka ritme. Yuk buat satu jadwal puasa dulu dengan /set-puasa atau /jadwalkan — nanti aku bantu pantau dan kasih semangat di tengah jalan.",
}

var motivationPreStart = []string{
	"🧭 Puasamu belum mulai. Ini waktu bagus buat menyiapkan mental: makan secukupnya, tutup jendela makan dengan tenang, lalu biarkan tubuh bekerja.",
	"⏳ Countdown dimulai. Saat puasa mulai nanti, targetmu bukan heroik — targetmu konsisten dan sadar sinyal tubuh.",
}

var motivationNearTarget = []string{
	"🏁 Tinggal sedikit lagi. Bagian akhir memang sering terasa paling panjang, tapi justru di sini kamu membangun kepercayaan diri.",
	"💪 Kamu hampir sampai target. Tarik napas, alihkan fokus sebentar, dan biarkan menit terakhir ini jadi bukti kamu bisa konsisten.",
}

var motivationTargetMet = []string{
	"🎊 Target puasamu sudah tercapai! Catat dengan /buka supaya durasi dan streak masuk ke statistik.",
	"✅ Kamu sudah melewati garis finish. Buka pelan-pelan, dengarkan tubuh, lalu jalankan /buka agar progress hari ini terekam.",
}

var hydrationNudges = []string{
	"💧 Kalau metode puasamu mengizinkan, jaga hidrasi. Kadang rasa lapar cuma sinyal haus yang salah dibaca tubuh.",
	"💧 Tetap cukup cairan bila metode puasamu water/IF. Hidrasi membantu fokus dan bikin fase puasa lebih nyaman.",
}

var electrolyteNudges = []string{
	"🧂 Puasa 24+ jam butuh perhatian elektrolit: natrium, kalium, dan magnesium membantu tubuh tetap stabil.",
	"⚡ Untuk water/prolonged fasting 24+ jam, jangan cuma air: perhatikan elektrolit Na/K/Mg agar badan tetap aman.",
}

func MotivationForPhase(key string) []string {
	return motivationByPhase[key]
}

func MotivationNoSchedule() []string {
	return motivationNoSchedule
}

func MotivationPreStart() []string {
	return motivationPreStart
}

func MotivationNearTarget() []string {
	return motivationNearTarget
}

func MotivationTargetMet() []string {
	return motivationTargetMet
}

func HydrationNudges() []string {
	return hydrationNudges
}

func ElectrolyteNudges() []string {
	return electrolyteNudges
}
