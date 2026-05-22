package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"fasting-bot/internal/config"
	"fasting-bot/internal/delivery/whatsapp"
	"fasting-bot/internal/infrastructure/database"
	"fasting-bot/internal/infrastructure/persistence"
	waInfra "fasting-bot/internal/infrastructure/whatsapp"
	"fasting-bot/internal/usecase"
)

func main() {
	_ = godotenv.Load() // ignore error — production uses systemd EnvironmentFile

	fmt.Println("🤖 Fasting Bot Starting...")
	fmt.Printf("Bot Number: %s\n", config.BotNumber)
	fmt.Printf("Admin Number: %s\n", config.AdminNumber)
	fmt.Printf("Group Name: %s\n", config.GroupName)

	db, err := database.New()
	if err != nil {
		fmt.Printf("❌ Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println("✅ Database initialized")

	userRepo := persistence.NewUserRepository(db.Conn)
	scheduleRepo := persistence.NewScheduleRepository(db.Conn)
	notificationRepo := persistence.NewNotificationRepository(db.Conn)

	fastingUsecase := usecase.NewFastingUsecase(userRepo, scheduleRepo, notificationRepo)

	waClient, err := waInfra.NewClient()
	if err != nil {
		fmt.Printf("❌ Failed to initialize WhatsApp client: %v\n", err)
		os.Exit(1)
	}

	handler := whatsapp.NewCommandHandler(waClient.WA, fastingUsecase)
	waClient.WA.AddEventHandler(handler.HandleEvent)

	notifier := waInfra.NewNotifier(waClient.WA)
	scheduler := whatsapp.NewScheduler(scheduleRepo, notifier)
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
