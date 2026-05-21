package whatsapp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"fasting-bot/internal/config"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
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
		fmt.Println("\n📱 No session found. Starting pairing process...")
		fmt.Println("   Bot Number: " + config.BotNumber)
		fmt.Println()

		qrChan, _ := client.GetQRChannel(context.Background())

		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		phoneNumber := strings.TrimPrefix(config.BotNumber, "+")
		
		for attempt := 1; attempt <= 3; attempt++ {
			if attempt > 1 {
				waitTime := time.Duration(attempt*10) * time.Second
				fmt.Printf("   Waiting %v before retry...\n", waitTime)
				time.Sleep(waitTime)
			}
			
			fmt.Printf("🔄 Method 1: Phone Number Pairing (attempt %d/3)\n", attempt)
			fmt.Println("   Requesting pairing code untuk nomor:", config.BotNumber)
			
			pairCode, err := client.PairPhone(context.Background(), phoneNumber, true, whatsmeow.PairClientChrome, "Fasting Bot")
			if err == nil {
				fmt.Println("✅ Pairing code generated!")
				fmt.Println()
				fmt.Println("📱 CARA MENGGUNAKAN PHONE PAIRING:")
				fmt.Println("   1. Buka WhatsApp di HP")
				fmt.Println("   2. Settings → Linked Devices → Link a Device")
				fmt.Println("   3. Pilih 'Link with phone number instead'")
				fmt.Println("   4. Masukkan kode:", pairCode)
				fmt.Println()
				fmt.Println("⏱️  Kode expired dalam 60 detik!")
				
				codeFile, _ := os.Create("/opt/fasting-bot/data/pairing-code.txt")
				if codeFile != nil {
					fmt.Fprintf(codeFile, "Fasting Bot WhatsApp Pairing Code\n")
					fmt.Fprintf(codeFile, "Phone: %s\n", config.BotNumber)
					fmt.Fprintf(codeFile, "Code: %s\n\n", pairCode)
					fmt.Fprintf(codeFile, "Cara menggunakan:\n")
					fmt.Fprintf(codeFile, "1. Buka WhatsApp di HP\n")
					fmt.Fprintf(codeFile, "2. Settings → Linked Devices → Link a Device\n")
					fmt.Fprintf(codeFile, "3. Pilih 'Link with phone number instead'\n")
					fmt.Fprintf(codeFile, "4. Masukkan kode: %s\n", pairCode)
					codeFile.Close()
					fmt.Println("💾 Pairing code disimpan di: /opt/fasting-bot/data/pairing-code.txt")
				}
				
				fmt.Println("\n⏳ Menunggu pairing selesai...")
				select {}
			}

			fmt.Println("⚠️  Phone pairing failed:", err)
			if strings.Contains(err.Error(), "429") {
				fmt.Println("   Rate limited by WhatsApp. Waiting longer...")
				fmt.Println("   Ini normal kalau terlalu banyak percobaan.")
			}
		}

		fmt.Println()
		fmt.Println("🔄 Method 2: QR Code (fallback)")
		fmt.Println("   WhatsApp → Settings → Linked Devices → Link a Device")
		fmt.Println()

		if qrChan == nil {
			return nil, fmt.Errorf("QR channel not available and phone pairing failed")
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("📲 QR Code generated! Scan sekarang:")
				fmt.Println()
				
				err := qrcode.WriteFile(evt.Code, qrcode.Medium, 512, "/opt/fasting-bot/data/qr-code.png")
				if err == nil {
					fmt.Println("💾 QR code PNG disimpan di: /opt/fasting-bot/data/qr-code.png")
					fmt.Println("   Download: scp root@103.169.206.19:/opt/fasting-bot/data/qr-code.png ./qr-code.png")
				}

				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
				fmt.Println("⏱️  QR code akan expired dalam 60 detik...")
			} else if evt.Event == "timeout" {
				fmt.Println("⏱️  QR code expired. Generating new one...")
			} else if evt.Event == "success" {
				fmt.Println("✅ QR Login successful!")
				break
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