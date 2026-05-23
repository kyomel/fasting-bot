package whatsapp

import (
	"context"
	"fmt"
	"log"
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

	if msg.Info.IsFromMe {
		return
	}

	log.Printf("📩 Message from %s in %s (Group: %v): %s", sender.User, chat.String(), isGroup, text)

	phone := "+" + sender.User
	if !isAuthorized(phone, chat.String(), isGroup) {
		log.Printf("🚫 Blocked: sender=%s chat=%s group=%v (allowed group=%s)", phone, chat.String(), isGroup, config.AllowedGroupJID)
		return
	}

	response, err := h.processCommand(phone, sender.String(), text)
	if err != nil {
		log.Printf("[ERROR] processCommand failed: %v", err)
	}

	if response == "" {
		return
	}

	replyTo := chat
	if !isGroup {
		replyTo = sender
	}

	_, sendErr := h.client.SendMessage(context.Background(), replyTo, &waProto.Message{
		Conversation: proto.String(response),
	})
	if sendErr != nil {
		log.Printf("[ERROR] SendMessage to %s (%d chars): %v", replyTo.String(), len(response), sendErr)
	} else {
		log.Printf("📤 Sent to %s (%d chars)", replyTo.String(), len(response))
	}
}

func isAuthorized(phone, chatJID string, isGroup bool) bool {
	if isAdminPhone(phone) {
		if !isGroup {
			return true
		}
		return config.AllowedGroupJID != "" && chatJID == config.AllowedGroupJID
	}

	if isGroup {
		return config.AllowedGroupJID != "" && chatJID == config.AllowedGroupJID
	}
	return false
}

func isAdminPhone(phone string) bool {
	return normalizePhone(phone) == normalizePhone(config.AdminNumber)
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	if at := strings.IndexByte(phone, '@'); at >= 0 {
		phone = phone[:at]
	}
	if colon := strings.IndexByte(phone, ':'); colon >= 0 {
		phone = phone[:colon]
	}

	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}

	normalized := digits.String()
	if strings.HasPrefix(normalized, "0") {
		normalized = "62" + strings.TrimLeft(normalized, "0")
	}
	if normalized == "" {
		return ""
	}
	return "+" + normalized
}

func (h *CommandHandler) processCommand(phone, jid, text string) (string, error) {
	text = strings.TrimSpace(text)
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return "", nil
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "/daftar", "/register":
		name := strings.Join(args, " ")
		return h.callUsecase(phone, "RegisterUser", func() (string, error) {
			return h.usecase.RegisterUser(phone, jid, name)
		})

	case "/setname":
		name := strings.Join(args, " ")
		return h.callUsecase(phone, "SetName", func() (string, error) {
			return h.usecase.SetName(phone, name)
		})

	case "/list-puasa":
		return domain.GetFastingTypesList(), nil

	case "/set-puasa":
		return h.handleSetPuasa(phone, args)

	case "/jadwalkan":
		return h.handleJadwalkan(phone, args)

	case "/jadwal-bebas":
		return h.handleJadwalBebas(phone, args)

	case "/status":
		return h.callUsecase(phone, "GetStatus", func() (string, error) {
			return h.usecase.GetStatus(phone)
		})

	case "/buka", "/cancel":
		return h.callUsecase(phone, "CancelToday", func() (string, error) {
			return h.usecase.CancelToday(phone)
		})

	case "/hapus":
		return h.callUsecase(phone, "DeleteSchedule", func() (string, error) {
			return h.usecase.DeleteSchedule(phone)
		})

	case "/stats":
		return h.callUsecase(phone, "GetStats", func() (string, error) {
			return h.usecase.GetStats(phone)
		})

	case "/leaderboard":
		return h.callUsecase(phone, "GetLeaderboard", func() (string, error) {
			return h.usecase.GetLeaderboard()
		})

	case "/help", "/bantuan":
		return getHelpText(), nil

	case "/info":
		return fmt.Sprintf("🤖 *Fasting Bot*\nGrup: %s\nBot: %s", config.GroupName, config.BotNumber), nil

	default:
		return "", nil
	}
}

func (h *CommandHandler) callUsecase(phone, label string, fn func() (string, error)) (string, error) {
	resp, err := fn()
	if err != nil {
		log.Printf("[ERROR] %s failed for %s: %v", label, phone, err)
		return "❌ Terjadi kesalahan saat " + errorLabel(label) + ". Coba lagi nanti.", nil
	}
	return resp, nil
}

const (
	ErrMsgSaveSchedule = "❌ Terjadi kesalahan saat menyimpan jadwal. Coba lagi nanti."
)

var errorLabels = map[string]string{
	"RegisterUser":   "mendaftar",
	"SetName":        "mengubah nama",
	"GetStatus":      "mengambil status",
	"CancelToday":    "membatalkan",
	"DeleteSchedule": "menghapus jadwal",
	"GetStats":       "mengambil stats",
	"GetLeaderboard": "mengambil leaderboard",
}

func errorLabel(method string) string {
	if label, ok := errorLabels[method]; ok {
		return label
	}
	return method
}

func (h *CommandHandler) handleSetPuasa(phone string, args []string) (string, error) {
	if len(args) < 2 {
		return "❌ Format salah.\n\nIF & OMAD (1-7): /set-puasa <nomor> <jam_mulai>\nContoh: /set-puasa 3 05:00\n\nWater/Dry/Prolonged (8-10): /set-puasa <nomor> <jam_mulai> <durasi_jam>\nContoh: /set-puasa 8 05:00 48\n\nJadwal tanggal khusus: /jadwalkan <nomor> <tanggal> <jam_mulai> [durasi_jam]\nContoh: /jadwalkan 3 23-05-2026 16:00", nil
	}

	typeID, err := strconv.Atoi(args[0])
	if err != nil || typeID < 1 || typeID > 10 {
		return "❌ Nomor puasa tidak valid. Pilih 1-10. Kirim /list-puasa untuk melihat daftar.", nil
	}

	startTime := args[1]
	durationHours := 0

	if typeID >= 8 && len(args) >= 3 {
		durationHours, err = strconv.Atoi(args[2])
		if err != nil {
			return "❌ Durasi jam harus angka.", nil
		}
	}

	resp, err := h.usecase.SetFastingType(phone, typeID, startTime, durationHours)
	if err != nil {
		log.Printf("[ERROR] SetFastingType failed: %v", err)
		return ErrMsgSaveSchedule, nil
	}
	return resp, nil
}

func (h *CommandHandler) handleJadwalkan(phone string, args []string) (string, error) {
	if len(args) < 3 {
		return "❌ Format salah.\nGunakan nomor puasa seperti /set-puasa: /jadwalkan <nomor> <tanggal> <jam_mulai> [durasi_jam]\nContoh IF: /jadwalkan 3 23-05-2026 16:00\nContoh Water Fasting: /jadwalkan 8 23-05-2026 16:00 48", nil
	}

	if strings.EqualFold(args[0], "WF") || strings.EqualFold(args[0], "DF") {
		return "❌ /jadwalkan tidak memakai kode WF/DF.\nUntuk Water/Dry dari daftar, pakai nomor seperti /set-puasa: /jadwalkan 8 23-05-2026 16:00 48 atau /jadwalkan 9 23-05-2026 16:00 18\nUntuk freestyle WF/DF, pakai: /jadwal-bebas WF 23-05-2026 16:00 12", nil
	}

	typeID, err := strconv.Atoi(args[0])
	if err != nil || typeID < 1 || typeID > 10 {
		return "❌ Nomor puasa tidak valid. Pilih 1-10. Kirim /list-puasa untuk melihat daftar.", nil
	}

	durationHours := 0
	if typeID >= 8 {
		if len(args) < 4 {
			return "❌ Durasi jam wajib untuk Water/Dry/Prolonged Fasting.\nContoh: /jadwalkan 8 23-05-2026 16:00 48", nil
		}
		durationHours, err = strconv.Atoi(args[3])
		if err != nil {
			return "❌ Durasi jam harus angka.\nContoh: /jadwalkan 8 23-05-2026 16:00 48", nil
		}
	}

	resp, err := h.usecase.ScheduleFastingType(phone, typeID, args[1], args[2], durationHours)
	if err != nil {
		log.Printf("[ERROR] ScheduleFastingType failed: %v", err)
		return ErrMsgSaveSchedule, nil
	}
	return resp, nil
}

func (h *CommandHandler) handleJadwalBebas(phone string, args []string) (string, error) {
	if len(args) != 4 {
		return "❌ Format salah.\nGunakan: /jadwal-bebas <WF|DF> <tanggal> <jam_mulai> <durasi_jam>\nContoh: /jadwal-bebas WF 23-05-2026 16:00 12", nil
	}

	kind := strings.ToUpper(args[0])
	durationHours, err := strconv.Atoi(args[3])
	if err != nil {
		return "❌ Durasi jam harus angka.\nContoh: /jadwal-bebas WF 23-05-2026 16:00 12", nil
	}

	resp, err := h.usecase.ScheduleFreestyleFasting(phone, kind, args[1], args[2], durationHours)
	if err != nil {
		log.Printf("[ERROR] ScheduleFreestyleFasting failed: %v", err)
		return ErrMsgSaveSchedule, nil
	}
	return resp, nil
}

func getHelpText() string {
	return `🤖 *Fasting Bot - Daftar Perintah*

/daftar <nama> - Daftar sebagai user
/setname <nama> - Ubah nama user
/list-puasa - Lihat jenis-jenis puasa
/set-puasa <nomor> <jam> [durasi] - Pilih jenis puasa
/jadwalkan <nomor> <tanggal> <jam> [durasi] - Seperti /set-puasa tapi pakai tanggal, bisa rollback
/jadwal-bebas <WF|DF> <tanggal> <jam> <durasi> - Khusus freestyle WF/DF
/status - Cek status akun & fasting
/buka - Batalkan fasting hari ini
/hapus - Hapus jadwal fasting aktif
/stats - Lihat statistik hasil buka puasa
/leaderboard - Lihat klasemen puasa
/help - Tampilkan bantuan ini

Contoh:
/daftar kyomel
/setname kyomel baru
/set-puasa 3 05:00
/set-puasa 8 05:00 48
/jadwalkan 3 23-05-2026 16:00
/jadwal-bebas WF 23-05-2026 16:00 12
/stats
/leaderboard
/hapus`
}
