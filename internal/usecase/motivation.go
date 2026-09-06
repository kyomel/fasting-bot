package usecase

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
)

var (
	motivationRandomMu sync.Mutex
	motivationRandom   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func (u *fastingUsecase) GetMotivation(phone string) (string, error) {
	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}
	if user == nil {
		return msgNotRegistered, nil
	}

	name := displayUserName(user)
	schedule, err := u.lookupActiveSchedule(user.ID)
	if err != nil {
		return "", err
	}
	if schedule == nil {
		return fmt.Sprintf("✨ *Motivasi Puasa untuk %s*\n\n%s", name, pickMotivation(domain.MotivationNoSchedule())), nil
	}

	now := time.Now().In(config.Location)
	startTime, _ := parseScheduleTime(schedule.FastStart, now)
	endTime, _ := parseScheduleTime(schedule.FastEnd, now)
	if !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	switch {
	case now.Before(startTime):
		return fmt.Sprintf(
			"✨ *Motivasi Puasa untuk %s*\n\n%s\n\n⏳ Mulai dalam: *%s*\n⏱ Jadwal mulai: %s\n\nSiapkan ritme, bukan tekanan. Kamu bisa. 💪",
			name,
			pickMotivation(domain.MotivationPreStart()),
			formatDuration(startTime.Sub(now)),
			formatDisplayTime(startTime),
		), nil
	case !now.Before(endTime):
		elapsed := now.Sub(startTime)
		body := pickMotivation(domain.MotivationTargetMet())
		if nudges := fastingSafetyNudges(schedule.FastingTypeName, elapsed); nudges != "" {
			body += "\n\n" + nudges
		}

		return fmt.Sprintf(
			"✨ *Motivasi Puasa untuk %s*\n\n%s\n\n🎯 Target: *%s*\n⏱ Durasi berjalan: *%s*\n\nKalau sudah buka, catat dengan */buka* ya — progress-mu layak terekam. 🏆",
			name,
			body,
			displayFastingTypeName(schedule.FastingTypeName),
			formatDuration(elapsed),
		), nil
	}

	elapsed := now.Sub(startTime)
	remaining := endTime.Sub(now)
	if remaining <= 2*time.Hour {
		body := pickMotivation(domain.MotivationNearTarget())
		if nudges := fastingSafetyNudges(schedule.FastingTypeName, elapsed); nudges != "" {
			body += "\n\n" + nudges
		}

		return fmt.Sprintf(
			"✨ *Motivasi Puasa untuk %s*\n\n%s\n\n⌛ Sudah berjalan: *%s*\n🏁 Sisa target: *%s*\n\nFinish strong — sebentar lagi kamu sampai. 🔥",
			name,
			body,
			formatDuration(elapsed),
			formatDuration(remaining),
		), nil
	}

	phase := domain.PhaseForElapsedHours(elapsed.Hours())
	body := pickMotivation(domain.MotivationForPhase(phase.Key))
	if nudges := fastingSafetyNudges(schedule.FastingTypeName, elapsed); nudges != "" {
		body += "\n\n" + nudges
	}

	return composeFastingMotivation(name, schedule, phase, elapsed, remaining, body), nil
}

func composeFastingMotivation(name string, schedule *domain.FastingSchedule, phase domain.MetabolicPhase, elapsed, remaining time.Duration, body string) string {
	message := fmt.Sprintf(
		"✨ *Motivasi Puasa untuk %s*\n\n"+
			"%s *%s*\n"+
			"Jenis: *%s*\n"+
			"⌛ Sudah berjalan: *%s*\n"+
			"🏁 Sisa target: *%s*\n\n"+
			"%s",
		name,
		phase.Emoji,
		phase.Name,
		displayFastingTypeName(schedule.FastingTypeName),
		formatDuration(elapsed),
		formatDuration(remaining),
		body,
	)

	if nextPhase, ok := phase.NextPhase(); ok {
		message += fmt.Sprintf("\n\n➡️ Berikutnya: *%s %s* sekitar jam ke-%.0f. Satu langkah lagi.", nextPhase.Emoji, nextPhase.Name, nextPhase.MinHours)
	}

	return message
}

func pickMotivation(pool []string) string {
	if len(pool) == 0 {
		return "💪 Kamu sedang membangun konsistensi. Lanjutkan pelan-pelan, satu jam demi satu jam."
	}

	motivationRandomMu.Lock()
	defer motivationRandomMu.Unlock()
	return pool[motivationRandom.Intn(len(pool))]
}

func isDryFasting(name string) bool {
	return strings.Contains(strings.ToLower(name), "dry fasting")
}

func fastingSafetyNudges(name string, elapsed time.Duration) string {
	if isDryFasting(name) {
		return ""
	}

	nudges := pickMotivation(domain.HydrationNudges())
	if isLongWaterFast(name, elapsed) {
		nudges += "\n\n" + pickMotivation(domain.ElectrolyteNudges())
	}
	return nudges
}

func isLongWaterFast(name string, elapsed time.Duration) bool {
	if elapsed < 24*time.Hour {
		return false
	}
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "water fasting") || strings.Contains(lowerName, "prolonged")
}
