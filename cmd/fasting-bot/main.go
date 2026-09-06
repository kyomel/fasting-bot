package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"database/sql"
	"fasting-bot/internal/config"
	deliveryHTTP "fasting-bot/internal/delivery/http"
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

	apiServer := startAPIServer(fastingUsecase)
	defer apiServer.Shutdown(context.Background())

	fmt.Println("\n🚀 Bot is running! Scan the QR code above to login.")
	fmt.Println("Press Ctrl+C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 Shutting down bot...")
	waClient.Disconnect()
}

// startAPIServer serves the REST API on API_ADDR (default :8080) in the
// background. It returns the server so main can shut it down gracefully.
func startAPIServer(fastingUsecase usecase.FastingUsecase) *http.Server {
	addr := config.APIAddr
	server := &http.Server{
		Addr:              addr,
		Handler:           deliveryHTTP.NewServer(fastingUsecase).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		fmt.Printf("✅ API listening on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ERROR] API server failed: %v", err)
		}
	}()
	return server
}

// newRepositories wires the PostgreSQL repository set. PostgreSQL is the
// only supported application database (SQLite remains only for the WhatsApp
// session store in infrastructure/whatsapp).
func newRepositories(db *sql.DB) (repository.UserRepository, repository.ScheduleRepository, repository.NotificationRepository, repository.BadgeRepository) {
	return persistence.NewUserRepositoryPostgres(db),
		persistence.NewScheduleRepositoryPostgres(db),
		persistence.NewNotificationRepositoryPostgres(db),
		persistence.NewBadgeRepositoryPostgres(db)
}
