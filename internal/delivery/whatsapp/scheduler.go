package whatsapp

import (
	"fmt"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/infrastructure/whatsapp"
	"fasting-bot/internal/repository"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	scheduleRepo repository.ScheduleRepository
	notifRepo    repository.NotificationRepository
	notifier     *whatsapp.Notifier
}

func NewScheduler(
	scheduleRepo repository.ScheduleRepository,
	notifRepo repository.NotificationRepository,
	notifier *whatsapp.Notifier,
) *Scheduler {
	return &Scheduler{
		scheduleRepo: scheduleRepo,
		notifRepo:    notifRepo,
		notifier:     notifier,
	}
}

func (s *Scheduler) Start() {
	s.cron = cron.New(
		cron.WithLocation(config.Location),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)
	s.cron.AddFunc("* * * * *", s.checkAndNotify)
	s.cron.AddFunc("0 3 */3 * *", s.cleanupFastingHistory)
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

func (s *Scheduler) checkAndNotify() {
	now := time.Now().In(config.Location)
	currentTime := now.Format("15:04")
	currentDate := now.Format("2006-01-02")
	currentDateTime := now.Format("2006-01-02 15:04")

	targets, err := s.scheduleRepo.FindUsersToNotifyStart(currentTime, currentDate, currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (start): %v\n", err)
		return
	}

	for _, t := range targets {
		msg := fmt.Sprintf("⏰ *Yuk mulai, %s!*\nPuasa kamu resmi dimulai sekarang.\n🏁 Target buka: *%s*\n\nWaktu yang paling berat biasanya jam pertama — tapi kamu udah buktiin bisa lewatin itu berkali-kali. 🔥\nAir putih boleh, nafas dalam boleh, istirahat boleh. Sisanya tahan ya! Semangat! 💪", t.Name, formatScheduleForMessage(t.FastEnd))
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send start notification: %v\n", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "start"); err != nil {
			fmt.Printf("❌ Failed to log start notification: %v\n", err)
			continue
		}
		fmt.Println("📨 Sent start notification")
	}

	targets, err = s.scheduleRepo.FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (end): %v\n", err)
		return
	}

	todayDate := now.Format("02-01-2006")
	for _, t := range targets {
		msg := fmt.Sprintf(
			"🎉 *Mantap, %s — kamu berhasil!*\nKamu udah tahan dari *%s* sampai *%s* — luar biasa! 💪\n\n"+
				"Sekarang catat buka puasa kamu biar masuk ke /stats & streak:\n"+
				"• Kirim */buka* → kalau kamu berbuka *sekarang*\n"+
				"• Kirim */buka %s HH:MM* → kalau udah buka tadi (contoh: */buka %s 18:30*)\n\n"+
				"_Tanpa /buka, durasi puasa kamu nggak ke-record lho!_ ⚠️\nKeep going — istirahat cukup, besok lanjut lagi 🌿",
			t.Name,
			formatScheduleForMessage(t.FastStart),
			formatScheduleForMessage(t.FastEnd),
			todayDate, todayDate,
		)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send end notification: %v\n", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "end"); err != nil {
			fmt.Printf("❌ Failed to log end notification: %v\n", err)
			continue
		}
		fmt.Println("📨 Sent end notification")
	}
}

func formatScheduleForMessage(value string) string {
	t, err := time.ParseInLocation("2006-01-02 15:04", value, config.Location)
	if err != nil {
		return value
	}
	return t.Format("02-01-2006 15:04")
}

func (s *Scheduler) cleanupFastingHistory() {
	cutoff := time.Now().In(config.Location).AddDate(0, 0, -3).Format("2006-01-02 15:04:05")
	deleted, err := s.scheduleRepo.CleanupOldFastingRecords(cutoff)
	if err != nil {
		fmt.Printf("❌ Failed to cleanup fasting history: %v\n", err)
		return
	}
	if deleted > 0 {
		fmt.Printf("🧹 Cleaned up %d old fasting history records\n", deleted)
	}
}
