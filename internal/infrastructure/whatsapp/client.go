package whatsapp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"fasting-bot/internal/config"

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
		fmt.Println("🔄 Using Phone Number Pairing (tanpa QR code)")
		fmt.Println()

		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		phoneNumber := strings.TrimPrefix(config.BotNumber, "+")
		fmt.Println("🔄 Requesting pairing code untuk nomor:", config.BotNumber)
		
		pairCode, err := client.PairPhone(context.Background(), phoneNumber, true, whatsmeow.PairClientChrome, "Fasting Bot")
		if err != nil {
			return nil, fmt.Errorf("failed to get pairing code: %w", err)
		}
		
		fmt.Println("✅ Pairing code generated!")
		fmt.Println()
		fmt.Println("📱 CARA MENGGUNAKAN:")
		fmt.Println("   1. Buka WhatsApp di HP")
		fmt.Println("   2. Settings → Linked Devices → Link a Device")
		fmt.Println("   3. Pilih 'Link with phone number instead'")
		fmt.Println("   4. Masukkan kode:", pairCode)
		fmt.Println()
		fmt.Println("⏱️  Kode expired dalam 60 detik!")
		fmt.Println("   Kalau gagal, restart bot: systemctl restart fasting-bot")
		
		codeFile, err := os.Create("/opt/fasting-bot/data/pairing-code.txt")
		if err == nil {
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
		fmt.Println("   (Tekan Ctrl+C kalau mau cancel)")
		
		select {}
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