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
	if !isAuthorized(chat.String(), isGroup) {
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

	_, sendErr := h.client.SendMessage(context.Background(), chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(response),
		},
	})
	if sendErr != nil {
		log.Printf("[ERROR] SendMessage to %s (%d chars): %v", chat.String(), len(response), sendErr)
	} else {
		log.Printf("📤 Sent to %s (%d chars)", chat.String(), len(response))
	}
}

func isAuthorized(chatJID string, isGroup bool) bool {
	return isGroup && config.AllowedGroupJID != "" && chatJID == config.AllowedGroupJID
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
	case "/daftar":
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

	case "/status":
		return h.callUsecase(phone, "GetStatus", func() (string, error) {
			return h.usecase.GetStatus(phone)
		})

	case "/buka":
		if len(args) > 0 {
			return h.handleBuka(phone, args)
		}
		return h.callUsecase(phone, "CancelToday", func() (string, error) {
			return h.usecase.CancelToday(phone)
		})

	case "/batalkan":
		return h.callUsecase(phone, "DeleteSchedule", func() (string, error) {
			return h.usecase.DeleteSchedule(phone)
		})

	case "/hapus":
		// Deprecated alias — /hapus diganti jadi /batalkan. Tetap dijalankan supaya user yang sudah terbiasa
		// tidak kehilangan akses; tampilkan hint agar pindah ke nama baru.
		resp, err := h.callUsecase(phone, "DeleteSchedule", func() (string, error) {
			return h.usecase.DeleteSchedule(phone)
		})
		if err != nil {
			return resp, err
		}
		return "ℹ️ Catatan: */hapus* sekarang sudah berubah jadi */batalkan*. Tetap berfungsi, tapi yuk pakai */batalkan* mulai sekarang.\n\n" + resp, nil

	case "/stats":
		return h.callUsecase(phone, "GetStats", func() (string, error) {
			return h.usecase.GetStats(phone)
		})

	case "/leaderboard":
		return h.callUsecase(phone, "GetLeaderboard", func() (string, error) {
			return h.usecase.GetLeaderboard()
		})

	case "/bantuan":
		return getHelpText(), nil

	case "/help":
		// Deprecated alias — /help diganti jadi /bantuan. Tetap dijalankan supaya user yang sudah terbiasa
		// tidak kehilangan akses; tampilkan hint agar pindah ke nama baru.
		return "ℹ️ Catatan: */help* sekarang sudah berubah jadi */bantuan*. Tetap berfungsi, tapi yuk pakai */bantuan* mulai sekarang.\n\n" + getHelpText(), nil

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
	"BreakFastingAt": "membuka puasa",
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
		return "❌ /jadwalkan harus pakai nomor 1-10, bukan WF/DF.\nWater Fasting pakai nomor 8, Dry Fasting pakai nomor 9.\nContoh: /jadwalkan 8 23-05-2026 16:00 48", nil
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

func (h *CommandHandler) handleBuka(phone string, args []string) (string, error) {
	if len(args) != 2 {
		return "❌ Format salah.\nGunakan: /buka DD-MM-YYYY HH:MM\nContoh: /buka 23-05-2026 18:30", nil
	}

	return h.callUsecase(phone, "BreakFastingAt", func() (string, error) {
		return h.usecase.BreakFastingAt(phone, args[0], args[1])
	})
}

func getHelpText() string {
	return `🤖 *Fasting Bot — Teman Puasa Kamu*

✨ *4 Perintah Utama:*
1️⃣ */set-puasa <nomor> <jam> [durasi]* — Pilih jenis puasa & mulai
2️⃣ */jadwalkan <nomor> <tanggal> <jam> [durasi]* — Jadwalkan untuk tanggal tertentu
3️⃣ */buka* — Catat buka puasa sekarang
4️⃣ */buka <tanggal> <jam>* — Catat buka puasa di waktu yang lalu (kalau lupa)

📋 *Perintah Pendukung:*
/daftar <nama> — Daftar sebagai user
/setname <nama> — Ubah nama
/list-puasa — Lihat jenis-jenis puasa
/status — Cek status puasa kamu sekarang
/batalkan — Batalkan jadwal puasa aktif
/stats — Statistik puasa pribadi
/leaderboard — Klasemen puasa grup
/bantuan — Tampilkan bantuan ini
/info — Info bot

💡 *Contoh praktis:*
/daftar kyomel
/set-puasa 3 05:00          (IF 16:8 mulai 05:00)
/set-puasa 8 05:00 48       (Water Fasting 48 jam)
/jadwalkan 3 23-05-2026 16:00
/buka                       (buka sekarang)
/buka 23-05-2026 18:30      (buka jam 18:30 tadi)

Konsisten dikit-dikit, hasilnya luar biasa. Yuk mulai! 💪`
}
