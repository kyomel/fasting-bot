package whatsapp

import (
	"fmt"
	"time"

	"fasting-bot/internal/infrastructure/whatsapp"
	"fasting-bot/internal/repository"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	scheduleRepo repository.ScheduleRepository
	notifier     *whatsapp.Notifier
}

func NewScheduler(
	scheduleRepo repository.ScheduleRepository,
	notifier *whatsapp.Notifier,
) *Scheduler {
	return &Scheduler{
		scheduleRepo: scheduleRepo,
		notifier:     notifier,
	}
}

func (s *Scheduler) Start() {
	s.cron = cron.New(cron.WithLocation(time.Local))
	s.cron.AddFunc("* * * * *", s.checkAndNotify)
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

func (s *Scheduler) checkAndNotify() {
	now := time.Now()
	currentTime := now.Format("15:04")
	currentDate := now.Format("2006-01-02")

	targets, err := s.scheduleRepo.FindUsersToNotifyStart(currentTime, currentDate)
	if err != nil {
		fmt.Printf("❌ Scheduler error (start): %v\n", err)
		return
	}

	for _, t := range targets {
		msg := fmt.Sprintf("⏰ *Waktu Fasting Dimulai!*\nFasting sampai %s. Semangat! 💪", t.FastEnd)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send start notification: %v\n", err)
			continue
		}
		fmt.Printf("📨 Sent start notification to %s\n", t.JID)
	}

	targets, err = s.scheduleRepo.FindUsersToNotifyEnd(currentTime, currentDate)
	if err != nil {
		fmt.Printf("❌ Scheduler error (end): %v\n", err)
		return
	}

	for _, t := range targets {
		msg := fmt.Sprintf("✅ *Fasting Selesai!*\nSelamat berbuka! 🎉\nFasting dari %s - %s", t.FastStart, t.FastEnd)
		if err := s.notifier.Send(t.JID, msg); err != nil {
			fmt.Printf("❌ Failed to send end notification: %v\n", err)
			continue
		}
		fmt.Printf("📨 Sent end notification to %s\n", t.JID)
	}
}
