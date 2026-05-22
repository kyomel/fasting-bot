package config

import (
	"os"
)

var (
	BotNumber    string
	AdminNumber  string
	GroupName    string
	DatabasePath string
	SessionPath  string
	QRCodePath   string
	QRCodeHost   string
)

func init() {
	BotNumber = getEnv("BOT_NUMBER", "")
	AdminNumber = getEnv("ADMIN_NUMBER", "")
	GroupName = getEnv("GROUP_NAME", "Fasting Group")
	DatabasePath = getEnv("DATABASE_PATH", "fasting-bot.db")
	SessionPath = getEnv("SESSION_PATH", "whatsapp-session.db")
	QRCodePath = getEnv("QR_CODE_PATH", "")
	QRCodeHost = getEnv("QR_CODE_HOST", "")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
