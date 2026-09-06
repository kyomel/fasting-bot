package usecase

import (
	"fmt"
	"math"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
)

func (u *fastingUsecase) GetPhases(phone string) (string, error) {
	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}

	message := "🧬 *Tahapan Metabolik Puasa*\n\n"
	for _, phase := range domain.MetabolicPhases {
		upper := "∞"
		if !isInf(phase.MaxHours) {
			upper = fmt.Sprintf("%.0f", phase.MaxHours)
		}
		message += fmt.Sprintf(
			"%s *%s* — jam %.0f–%s\n%s\n\n",
			phase.Emoji,
			phase.Name,
			phase.MinHours,
			upper,
			phaseBlurb(phase.Key),
		)
	}

	if user == nil {
		message += "_Daftar dulu: */daftar <nama>*, lalu */puasa 16* untuk mulai._"
		return message, nil
	}

	schedule, err := u.lookupActiveSchedule(user.ID)
	if err != nil {
		return "", err
	}
	if schedule == nil {
		message += "_Belum ada puasa aktif. Mulai: */puasa 16*._"
		return message, nil
	}

	now := time.Now().In(config.Location)
	startTime, _ := parseScheduleTime(schedule.FastStart, now)
	endTime, _ := parseScheduleTime(schedule.FastEnd, now)
	if !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	var elapsedHours float64
	switch {
	case now.Before(startTime):
		message += fmt.Sprintf("_Jadwal aktif: *%s* mulai %s (belum jalan)._ ", displayFastingTypeName(schedule.FastingTypeName), formatDisplayTime(startTime))
		return message, nil
	case now.Before(endTime):
		elapsedHours = now.Sub(startTime).Hours()
	default:
		elapsedHours = now.Sub(startTime).Hours()
	}

	current := domain.PhaseForElapsedHours(elapsedHours)
	message += fmt.Sprintf(
		"📍 *Posisimu sekarang:* %s *%s* (jam ke-%.1f)\nJenis: *%s*",
		current.Emoji,
		current.Name,
		elapsedHours,
		displayFastingTypeName(schedule.FastingTypeName),
	)
	if next, ok := current.NextPhase(); ok {
		message += fmt.Sprintf("\nBerikutnya: %s %s sekitar jam ke-%.0f", next.Emoji, next.Name, next.MinHours)
	}
	return message, nil
}

func phaseBlurb(key string) string {
	switch key {
	case "fed":
		return "Insulin tinggi, tubuh pakai glukosa dari makan terakhir."
	case "post_absorptive":
		return "Glukosa darah turun; glikogen hati mulai dipakai."
	case "fat_burning":
		return "Metabolic switch: lemak jadi bahan bakar utama."
	case "ketosis":
		return "Ketone naik; fokus & autophagy mulai aktif."
	case "deep_fast":
		return "Deep repair: HGH naik, cellular cleanup dominan."
	default:
		return ""
	}
}

func isInf(v float64) bool {
	return math.IsInf(v, 1)
}
