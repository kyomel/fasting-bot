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
		msg := fmt.Sprintf("⏰ *Waktu Fasting Dimulai!*\nFasting sampai %s. Semangat! 💪", formatScheduleForMessage(t.FastEnd))
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send start notification: %v\n", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "start"); err != nil {
			fmt.Printf("❌ Failed to log start notification: %v\n", err)
			continue
		}
		fmt.Printf("📨 Sent start notification to %s\n", t.JID)
	}

	targets, err = s.scheduleRepo.FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime)
	if err != nil {
		fmt.Printf("❌ Scheduler error (end): %v\n", err)
		return
	}

	for _, t := range targets {
		msg := fmt.Sprintf("✅ *Fasting Selesai!*\nPuasa kamu sudah selesai. Saatnya berbuka! 🎉\nFasting dari %s - %s", formatScheduleForMessage(t.FastStart), formatScheduleForMessage(t.FastEnd))
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send end notification: %v\n", err)
			continue
		}
		if err := s.notifRepo.LogNotification(t.UserID, "end"); err != nil {
			fmt.Printf("❌ Failed to log end notification: %v\n", err)
			continue
		}
		fmt.Printf("📨 Sent end notification to %s\n", t.JID)
	}
}

func formatScheduleForMessage(value string) string {
	t, err := time.ParseInLocation("2006-01-02 15:04", value, config.Location)
	if err != nil {
		return value
	}
	return t.Format("02-01-2006 15:04")
}
