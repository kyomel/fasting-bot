package config

import (
	"os"
	"time"

	_ "time/tzdata"
)

var (
	BotNumber    string
	AdminNumber  string
	GroupName    string
	DatabasePath string
	SessionPath  string
	QRCodePath   string
	QRCodeHost   string
	AppTimezone  string
	Location     *time.Location
)

func init() {
	Load()
}

func Load() {
	BotNumber = getEnv("BOT_NUMBER", "")
	AdminNumber = getEnv("ADMIN_NUMBER", "")
	GroupName = getEnv("GROUP_NAME", "Fasting Group")
	DatabasePath = getEnv("DATABASE_PATH", "fasting-bot.db")
	SessionPath = getEnv("SESSION_PATH", "whatsapp-session.db")
	QRCodePath = getEnv("QR_CODE_PATH", "")
	QRCodeHost = getEnv("QR_CODE_HOST", "")
	AppTimezone = getEnv("APP_TIMEZONE", "Asia/Jakarta")
	Location = loadLocation(AppTimezone)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}
