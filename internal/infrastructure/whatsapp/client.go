package whatsapp

import (
	"context"
	"fmt"
	"os"
	"time"

	"fasting-bot/internal/config"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const qrTimeout = 3 * time.Minute

type Client struct {
	WA *whatsmeow.Client
}

func NewClient() (*Client, error) {
	logger := waLog.Stdout("Client", "INFO", true)

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+config.SessionPath+"?_foreign_keys=on", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, logger)

	if client.Store.ID == nil {
		fmt.Println()
		fmt.Println("📱 No session found — QR Code pairing")
		fmt.Println("   Bot: " + config.BotNumber)
		fmt.Println()

		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get QR channel: %w", err)
		}

		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		fmt.Println("📲 Tunggu QR code muncul...")
		fmt.Println("   WhatsApp → Settings → Linked Devices → Link a Device")
		fmt.Println()

		deadline := time.After(qrTimeout)
		var qrCount int

		for {
			select {
			case <-deadline:
				fmt.Println()
				fmt.Println("⏱️  Waktu pairing habis (3 menit).")
				fmt.Println("   Jalankan ulang bot untuk mencoba lagi.")
				return nil, fmt.Errorf("pairing timeout after %v", qrTimeout)

			case evt, ok := <-qrChan:
				if !ok {
					return nil, fmt.Errorf("QR channel closed unexpectedly")
				}

				switch evt.Event {
				case "code":
					qrCount++
					fmt.Printf("📲 QR Code #%d — scan sekarang:\n", qrCount)
					fmt.Println()

					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					fmt.Println()

					if config.QRCodePath != "" {
						qrcode.WriteFile(evt.Code, qrcode.Medium, 512, config.QRCodePath)
						fmt.Println("💾 Saved: " + config.QRCodePath)
						if config.QRCodeHost != "" {
							fmt.Printf("   scp %s:%s ./qr-code.png\n", config.QRCodeHost, config.QRCodePath)
						}
					}
					fmt.Println("⏱️  QR ini expired ~60 detik, tapi akan regenerate otomatis.")
					fmt.Println()

				case "timeout":
					fmt.Println("   ⏱️  QR expired — generating new one...")

				case "success":
					fmt.Println()
					fmt.Println("✅ QR scanned! Authenticating...")

				default:
					fmt.Printf("   Event: %s\n", evt.Event)
				}

				if evt.Event == "success" {
					goto done
				}
			}
		}

	done:
		fmt.Println("✅ Connected to WhatsApp!")
	} else {
		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		fmt.Println("✅ Connected to WhatsApp (existing session)")
	}

	return &Client{WA: client}, nil
}

func (c *Client) Disconnect() {
	c.WA.Disconnect()
}
