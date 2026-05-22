package whatsapp

import (
	"context"
	"fmt"
	"os"
	"time"

	"fasting-bot/internal/config"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const qrTimeout = 3 * time.Minute

type Client struct {
	WA *whatsmeow.Client
}

func NewClient() (*Client, error) {
	logger := waLog.Stdout("Client", "WARN", true)

	sessionPath, err := config.SecureFilePath(config.SessionPath)
	if err != nil {
		return nil, fmt.Errorf("invalid session path: %w", err)
	}

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+sessionPath+"?_foreign_keys=on", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, logger)

	if client.Store.ID == nil {
		if err := pairWithQRCode(client); err != nil {
			return nil, err
		}
		fmt.Println("✅ Connected to WhatsApp!")
	} else {
		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		fmt.Println("✅ Connected to WhatsApp (existing session)")
	}

	return &Client{WA: client}, nil
}

func pairWithQRCode(client *whatsmeow.Client) error {
	if !isInteractiveStdout() && config.QRCodePath == "" {
		return fmt.Errorf("no WhatsApp session found; pairing must be done interactively or with QR_CODE_PATH")
	}

	fmt.Println()
	fmt.Println("📱 No session found — QR Code pairing")
	fmt.Println()

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("📲 Tunggu QR code muncul...")
	fmt.Println("   WhatsApp → Settings → Linked Devices → Link a Device")
	fmt.Println()

	return waitForQRScan(qrChan)
}

func waitForQRScan(qrChan <-chan whatsmeow.QRChannelItem) error {
	defer removeQRCodeFile()

	deadline := time.After(qrTimeout)
	var qrCount int

	for {
		select {
		case <-deadline:
			fmt.Println()
			fmt.Println("⏱️  Waktu pairing habis (3 menit).")
			fmt.Println("   Jalankan ulang bot untuk mencoba lagi.")
			return fmt.Errorf("pairing timeout after %v", qrTimeout)

		case evt, ok := <-qrChan:
			if !ok {
				return fmt.Errorf("QR channel closed unexpectedly")
			}
			if evt.Event == "success" {
				fmt.Println()
				fmt.Println("✅ QR scanned! Authenticating...")
				return nil
			}
			handleQREvent(evt, &qrCount)
		}
	}
}

func handleQREvent(evt whatsmeow.QRChannelItem, qrCount *int) {
	switch evt.Event {
	case "code":
		(*qrCount)++
		fmt.Printf("📲 QR Code #%d — scan sekarang:\n", *qrCount)
		fmt.Println()

		if isInteractiveStdout() {
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Println()
		}

		if config.QRCodePath != "" {
			qrCodePath, err := config.SecureFilePath(config.QRCodePath)
			if err != nil {
				fmt.Printf("⚠️  Failed to prepare QR code file: %v\n", err)
				return
			}
			if err := qrcode.WriteFile(evt.Code, qrcode.Medium, 512, qrCodePath); err != nil {
				fmt.Printf("⚠️  Failed to save QR code: %v\n", err)
				return
			}
			fmt.Println("💾 Saved: " + qrCodePath)
			if config.QRCodeHost != "" {
				fmt.Printf("   scp %s:%s ./qr-code.png\n", config.QRCodeHost, qrCodePath)
			}
		} else if !isInteractiveStdout() {
			fmt.Println("⚠️  QR code not printed because stdout is not interactive.")
		}
		fmt.Println("⏱️  QR ini expired ~60 detik, tapi akan regenerate otomatis.")
		fmt.Println()

	case "timeout":
		fmt.Println("   ⏱️  QR expired — generating new one...")

	default:
		fmt.Printf("   Event: %s\n", evt.Event)
	}
}

func (c *Client) Disconnect() {
	c.WA.Disconnect()
}

func isInteractiveStdout() bool {
	info, err := os.Stdout.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}

func removeQRCodeFile() {
	if config.QRCodePath == "" {
		return
	}
	qrCodePath, err := config.SecureFilePath(config.QRCodePath)
	if err != nil {
		return
	}
	_ = os.Remove(qrCodePath)
}
