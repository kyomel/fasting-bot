package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"database/sql"
	"fasting-bot/internal/config"
	"fasting-bot/internal/delivery/whatsapp"
	"fasting-bot/internal/infrastructure/database"
	"fasting-bot/internal/infrastructure/persistence"
	waInfra "fasting-bot/internal/infrastructure/whatsapp"
	"fasting-bot/internal/usecase"

	"fasting-bot/internal/repository"
)

func main() {
	_ = godotenv.Load() // ignore error — production uses systemd EnvironmentFile
	config.Load()

	fmt.Println("🤖 Fasting Bot Starting...")
	fmt.Printf("Group Name: %s\n", config.GroupName)
	fmt.Printf("Timezone: %s\n", config.AppTimezone)

	db, err := database.New()
	if err != nil {
		fmt.Printf("❌ Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println("✅ Database initialized")

	userRepo, scheduleRepo, notificationRepo, badgeRepo := newRepositories(db.Conn)

	fastingUsecase := usecase.NewFastingUsecase(userRepo, scheduleRepo, notificationRepo, badgeRepo)

	waClient, err := waInfra.NewClient()
	if err != nil {
		fmt.Printf("❌ Failed to initialize WhatsApp client: %v\n", err)
		os.Exit(1)
	}

	handler := whatsapp.NewCommandHandler(waClient.WA, fastingUsecase)
	waClient.WA.AddEventHandler(handler.HandleEvent)

	notifier := waInfra.NewNotifier(waClient.WA)
	scheduler := whatsapp.NewScheduler(scheduleRepo, notificationRepo, notifier)
	scheduler.Start()
	defer scheduler.Stop()
	fmt.Println("✅ Scheduler started")

	fmt.Println("\n🚀 Bot is running! Scan the QR code above to login.")
	fmt.Println("Press Ctrl+C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 Shutting down bot...")
	waClient.Disconnect()
}

// newRepositories selects the PostgreSQL or SQLite repository set based on
// the configured DB_CONNECTION. PostgreSQL is the target for new deployments;
// SQLite remains the fallback when DB_CONNECTION is empty.
func newRepositories(db *sql.DB) (repository.UserRepository, repository.ScheduleRepository, repository.NotificationRepository, repository.BadgeRepository) {
	if config.DBConnection != "" {
		return persistence.NewUserRepositoryPostgres(db),
			persistence.NewScheduleRepositoryPostgres(db),
			persistence.NewNotificationRepositoryPostgres(db),
			persistence.NewBadgeRepositoryPostgres(db)
	}
	return persistence.NewUserRepository(db),
		persistence.NewScheduleRepository(db),
		persistence.NewNotificationRepository(db),
		persistence.NewBadgeRepository(db)
}
