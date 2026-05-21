package whatsapp

import (
	"context"
	"fmt"
	"os"

	"fasting-bot/internal/config"

	"github.com/mdp/qrterminal/v3"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

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
		fmt.Println("\n📱 No session found. Please scan the QR code below:")
		fmt.Println("   (If QR code doesn't appear, make sure your terminal supports Unicode)\n")

		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get QR channel: %w", err)
		}

		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\n📲 Scan this QR code with WhatsApp:")
				fmt.Println("   WhatsApp → Settings → Linked Devices → Link a Device")
				fmt.Println()
				
				// Save QR to file for easy viewing
				qrFile, err := os.Create("/opt/fasting-bot/data/qr-code.txt")
				if err == nil {
					fmt.Fprintf(qrFile, "Fasting Bot WhatsApp QR Code\n")
					fmt.Fprintf(qrFile, "Scan dengan: WhatsApp → Settings → Linked Devices → Link a Device\n\n")
					qrterminal.GenerateWithConfig(evt.Code, qrterminal.Config{
						Level:     qrterminal.L,
						Writer:    qrFile,
						BlackChar: qrterminal.WHITE,
						WhiteChar: qrterminal.BLACK,
						QuietZone: 2,
					})
					qrFile.Close()
					fmt.Println("💾 QR code juga disimpan di: /opt/fasting-bot/data/qr-code.txt")
					fmt.Println("   Download: scp root@103.169.206.19:/opt/fasting-bot/data/qr-code.txt ./qr-code.txt")
				}
				
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
			} else if evt.Event == "timeout" {
				fmt.Println("⏱️  QR code expired. Generating new one...")
			} else if evt.Event == "success" {
				fmt.Println("✅ Login successful!")
				break
			} else {
				fmt.Printf("QR Event: %s\n", evt.Event)
			}
		}
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
