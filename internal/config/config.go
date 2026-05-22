package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "time/tzdata"
)

var (
	BotNumber       string
	AdminNumber     string
	AllowedGroupJID string
	GroupName       string
	DatabasePath    string
	SessionPath     string
	QRCodePath      string
	QRCodeHost      string
	AppTimezone     string
	Location        *time.Location
)

func init() {
	Load()
}

func Load() {
	BotNumber = getEnv("BOT_NUMBER", "")
	AdminNumber = getEnv("ADMIN_NUMBER", "")
	AllowedGroupJID = getEnv("ALLOWED_GROUP_JID", "")
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

func SecureFilePath(value string) (string, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	lowerPath := strings.ToLower(path)
	if strings.HasPrefix(lowerPath, "file:") || strings.ContainsAny(path, "?#\x00\r\n") {
		return "", fmt.Errorf("path must be a plain filesystem path")
	}

	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0700); err != nil {
		return "", err
	}

	return absPath, nil
}
