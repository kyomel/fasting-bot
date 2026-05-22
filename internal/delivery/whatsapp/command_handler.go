package whatsapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/usecase"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type CommandHandler struct {
	client  *whatsmeow.Client
	usecase usecase.FastingUsecase
}

func NewCommandHandler(client *whatsmeow.Client, usecase usecase.FastingUsecase) *CommandHandler {
	return &CommandHandler{
		client:  client,
		usecase: usecase,
	}
}

func (h *CommandHandler) HandleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		h.handleMessage(v)
	case *events.Connected:
		fmt.Println("✅ Connected to WhatsApp!")
	case *events.Disconnected:
		fmt.Println("❌ Disconnected from WhatsApp")
	}
}

func (h *CommandHandler) handleMessage(msg *events.Message) {
	var text string
	if msg.Message.GetConversation() != "" {
		text = msg.Message.GetConversation()
	} else if msg.Message.GetExtendedTextMessage() != nil {
		text = msg.Message.GetExtendedTextMessage().GetText()
	}

	if text == "" {
		return
	}

	sender := msg.Info.Sender
	chat := msg.Info.Chat
	isGroup := msg.Info.IsGroup

	fmt.Printf("📩 Message from %s in %s (Group: %v): %s\n",
		sender.User, chat.User, isGroup, text)

	if msg.Info.IsFromMe {
		return
	}

	phone := "+" + sender.User
	response := h.processCommand(phone, sender.String(), text)
	if response != "" {
		replyTo := chat
		if !isGroup {
			replyTo = sender
		}

		_, err := h.client.SendMessage(context.Background(), replyTo, &waProto.Message{
			Conversation: proto.String(response),
		})
		if err != nil {
			fmt.Printf("❌ Failed to send message: %v\n", err)
		}
	}
}

func (h *CommandHandler) processCommand(phone, jid, text string) string {
	text = strings.TrimSpace(text)
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return ""
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "/daftar", "/register":
		resp, _ := h.usecase.RegisterUser(phone, jid, strings.Join(args, " "))
		return resp
	case "/list-puasa":
		return domain.GetFastingTypesList()
	case "/set-puasa":
		return h.handleSetPuasa(phone, args)
	case "/status":
		resp, _ := h.usecase.GetStatus(phone)
		return resp
	case "/buka", "/cancel":
		resp, _ := h.usecase.CancelToday(phone)
		return resp
	case "/help", "/bantuan":
		return getHelpText()
	case "/info":
		return fmt.Sprintf("🤖 *Fasting Bot*\nGrup: %s\nBot: %s", config.GroupName, config.BotNumber)
	default:
		return ""
	}
}

func (h *CommandHandler) handleSetPuasa(phone string, args []string) string {
	if len(args) < 2 {
		return "❌ Format salah.\n\nIF & OMAD (1-7): /set-puasa <nomor> <jam_mulai>\nContoh: /set-puasa 3 05:00\n\nWater/Dry Fasting (8-10): /set-puasa <nomor> <jam_mulai> <durasi_jam>\nContoh: /set-puasa 8 05:00 48"
	}

	typeID, err := strconv.Atoi(args[0])
	if err != nil || typeID < 1 || typeID > 10 {
		return "❌ Nomor puasa tidak valid. Pilih 1-10. Kirim /list-puasa untuk melihat daftar."
	}

	startTime := args[1]
	durationHours := 0

	if typeID >= 8 && len(args) >= 3 {
		durationHours, err = strconv.Atoi(args[2])
		if err != nil {
			return "❌ Durasi jam harus angka."
		}
	}

	resp, _ := h.usecase.SetFastingType(phone, typeID, startTime, durationHours)
	return resp
}

func getHelpText() string {
	return `🤖 *Fasting Bot - Daftar Perintah*

/daftar - Daftar sebagai user
/list-puasa - Lihat jenis-jenis puasa
/set-puasa <nomor> <jam> [durasi] - Pilih jenis puasa
/status - Cek status fasting hari ini
/buka - Batalkan fasting hari ini
/help - Tampilkan bantuan ini

Contoh:
/set-puasa 3 05:00
/set-puasa 6 05:00
/set-puasa 8 05:00 48
/set-puasa 10 05:00 18`
}
