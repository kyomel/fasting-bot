package domain

import (
	"fmt"
	"sort"
	"strings"
)

const (
	NotificationTypePhaseFatBurning = "phase_fat_burning"
	NotificationTypePhaseDeepFast   = "phase_deep_fast"
	NotificationTypeNearTarget      = "near_target"
)

type ProactivePhaseNotification struct {
	NotificationType  string
	TriggerAfterHours int
	PhaseKey          string
}

// ProactivePhaseNotifications fires metabolic-phase nudges during an active fast.
// We skip the 18h (Ketosis) entry on purpose: for most IF schedules (12-23h) it
// would fire too close to the 12h (Fat Burning) nudge to add value, and for
// longer water/prolonged fasts the 24h (Deep Fast) nudge is the meaningful
// signal. Net effect: IF users see 1 phase nudge; water/prolonged users see 2.
var ProactivePhaseNotifications = []ProactivePhaseNotification{
	{NotificationType: NotificationTypePhaseFatBurning, TriggerAfterHours: 12, PhaseKey: "fat_burning"},
	{NotificationType: NotificationTypePhaseDeepFast, TriggerAfterHours: 24, PhaseKey: "deep_fast"},
}

type SmartNotificationPlan struct {
	HydrationReminderHours []int
	DrySafetyReminderHours []int
	PreBreakLeadHours      int
}

// HydrationReminderHours returns the union of elapsed-hour hydration checks the
// scheduler needs to query. Each target is still filtered through its own smart
// plan before sending, so short IF schedules do not receive long-fast nudges.
func HydrationReminderHours() []int {
	return []int{8, 12, 16, 18, 24, 30, 32, 48, 72, 96, 120, 144, 168}
}

func DrySafetyReminderHours() []int {
	return []int{4, 8, 12, 16, 24}
}

func PreBreakLeadHours() []int {
	return []int{1, 2, 3, 4, 6, 8}
}

func HydrationNotificationType(hours int) string {
	return fmt.Sprintf("hydration_%dh", hours)
}

func DrySafetyNotificationType(hours int) string {
	return fmt.Sprintf("dry_safety_%dh", hours)
}

func PreBreakNotificationType(leadHours int) string {
	return fmt.Sprintf("pre_break_%dh", leadHours)
}

func SmartNotificationPlanFor(fastingTypeName string, plannedHours int) SmartNotificationPlan {
	if plannedHours < 0 {
		plannedHours = 0
	}

	leadHours := preBreakLeadHoursFor(plannedHours)
	if isDryFastingType(fastingTypeName) {
		return SmartNotificationPlan{
			DrySafetyReminderHours: filteredReminderHours(drySafetyReminderHoursFor(plannedHours), plannedHours, leadHours),
			PreBreakLeadHours:      leadHours,
		}
	}

	return SmartNotificationPlan{
		HydrationReminderHours: filteredReminderHours(hydrationReminderHoursFor(plannedHours), plannedHours, leadHours),
		PreBreakLeadHours:      leadHours,
	}
}

func preBreakLeadHoursFor(plannedHours int) int {
	switch {
	case plannedHours <= 14:
		return 1
	case plannedHours <= 23:
		return 2
	case plannedHours <= 36:
		return 3
	case plannedHours <= 48:
		return 4
	case plannedHours <= 72:
		return 6
	default:
		return 8
	}
}

func hydrationReminderHoursFor(plannedHours int) []int {
	switch {
	case plannedHours <= 14:
		return nil
	case plannedHours <= 18:
		return []int{8}
	case plannedHours <= 23:
		return []int{12}
	case plannedHours <= 24:
		return []int{12, 18}
	case plannedHours <= 36:
		return []int{16, 24}
	case plannedHours <= 48:
		return []int{16, 30}
	case plannedHours <= 72:
		return []int{16, 32, 48}
	default:
		hours := []int{16, 32, 48}
		for hour := 72; hour < plannedHours; hour += 24 {
			hours = append(hours, hour)
		}
		return hours
	}
}

func drySafetyReminderHoursFor(plannedHours int) []int {
	switch {
	case plannedHours <= 6:
		return nil
	case plannedHours <= 12:
		return []int{4}
	case plannedHours <= 18:
		return []int{8}
	case plannedHours <= 24:
		return []int{8, 16}
	default:
		return []int{8, 16, 24}
	}
}

func filteredReminderHours(hours []int, plannedHours, preBreakLeadHours int) []int {
	if len(hours) == 0 || plannedHours <= 0 {
		return nil
	}

	latestUsefulHour := plannedHours - preBreakLeadHours - 1
	seen := map[int]struct{}{}
	filtered := make([]int, 0, len(hours))
	for _, hour := range hours {
		if hour <= 0 || hour > latestUsefulHour {
			continue
		}
		if _, ok := seen[hour]; ok {
			continue
		}
		seen[hour] = struct{}{}
		filtered = append(filtered, hour)
	}
	sort.Ints(filtered)
	return filtered
}

func isDryFastingType(name string) bool {
	return strings.Contains(strings.ToLower(name), "dry fasting")
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
	"🌱 Belum ada jadwal puasa aktif. Mulai dari langkah kecil saja: baca /pemula, lalu coba /puasa 12. Konsistensi menang dari sempurna.",
	"🚀 Tubuh suka ritme. Yuk buat satu jadwal puasa dulu dengan /puasa 16 atau /puasa 16 20-06-2026 05:00 — nanti aku bantu pantau dan kasih semangat di tengah jalan.",
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
