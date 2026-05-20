package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"fasting-bot/internal/config"
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
	text = strings.TrimSpace(strings.ToLower(text))
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return ""
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "/daftar", "/register":
		resp, _ := h.usecase.RegisterUser(phone, jid)
		return resp
	case "/jadwal", "/schedule":
		if len(args) < 2 {
			return "❌ Format salah. Gunakan: /jadwal HH:MM HH:MM\nContoh: /jadwal 05:00 18:00"
		}
		resp, _ := h.usecase.SetSchedule(phone, args[0], args[1])
		return resp
	case "/status":
		resp, _ := h.usecase.GetStatus(phone)
		return resp
	case "/batal", "/cancel":
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

func getHelpText() string {
	return `🤖 *Fasting Bot - Daftar Perintah*

/daftar - Daftar sebagai user
/jadwal HH:MM HH:MM - Atur jadwal fasting (mulai selesai)
/status - Cek status fasting hari ini
/batal - Batalkan fasting hari ini
/help - Tampilkan bantuan ini

Contoh:
/jadwal 05:00 18:00`
}
