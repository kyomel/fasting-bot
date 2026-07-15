package whatsapp

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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

	case "/panduan":
		return domain.GetPanduan(), nil

	case "/pemula":
		return domain.GetPemulaGuide(), nil

	case "/puasa":
		return h.handlePuasa(phone, args)
	case "/puasa-dry":
		return h.handlePuasaDry(phone, args)

	case "/if-1212":
		return h.handlePuasa(phone, append([]string{"12"}, args...))
	case "/if-1410":
		return h.handlePuasa(phone, append([]string{"14"}, args...))
	case "/if-168":
		return h.handlePuasa(phone, append([]string{"16"}, args...))
	case "/if-186":
		return h.handlePuasa(phone, append([]string{"18"}, args...))
	case "/if-204":
		return h.handlePuasa(phone, append([]string{"20"}, args...))
	case "/omad":
		return h.handlePuasa(phone, append([]string{"22"}, args...))
	case "/water-24":
		return h.handlePuasa(phone, append([]string{"24"}, args...))
	case "/water-36":
		return h.handlePuasa(phone, append([]string{"36"}, args...))
	case "/water-48":
		return h.handlePuasa(phone, append([]string{"48"}, args...))
	case "/water-72":
		return h.handlePuasa(phone, append([]string{"72"}, args...))
	case "/dry-24":
		return h.handlePuasaDry(phone, append([]string{"24"}, args...))

	case "/status":
		return h.callUsecase(phone, "GetStatus", func() (string, error) {
			return h.usecase.GetStatus(phone)
		})

	case "/fase", "/tahap":
		return h.callUsecase(phone, "GetPhases", func() (string, error) {
			return h.usecase.GetPhases(phone)
		})

	case "/motivasi":
		return h.callUsecase(phone, "GetMotivation", func() (string, error) {
			return h.usecase.GetMotivation(phone)
		})

	case "/buka":
		if len(args) > 0 {
			return h.handleBuka(phone, args)
		}
		return h.callUsecase(phone, "BreakFastingAt", func() (string, error) {
			now := time.Now().In(config.Location)
			return h.usecase.BreakFastingAt(phone, now.Format("02-01-2006"), now.Format("15:04"))
		})

	case "/batalkan":
		return h.callUsecase(phone, "DeleteSchedule", func() (string, error) {
			return h.usecase.DeleteSchedule(phone)
		})

	case "/stats":
		return h.callUsecase(phone, "GetStats", func() (string, error) {
			return h.usecase.GetStats(phone)
		})

	case "/riwayat", "/history":
		return h.handleRiwayat(phone, args)

	case "/badge":
		return h.callUsecase(phone, "GetBadges", func() (string, error) {
			return h.usecase.GetBadges(phone)
		})

	case "/leaderboard":
		return h.callUsecase(phone, "GetLeaderboard", func() (string, error) {
			return h.usecase.GetLeaderboard()
		})

	case "/bantuan":
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
	"RegisterUser":              "mendaftar",
	"SetName":                   "mengubah nama",
	"GetStatus":                 "mengambil status",
	"GetMotivation":             "mengambil pesan motivasi",
	"CancelToday":               "membatalkan",
	"BreakFastingAt":            "membuka puasa",
	"DeleteSchedule":            "menghapus jadwal",
	"GetStats":                  "mengambil stats",
	"GetHistory":                "mengambil riwayat",
	"GetBadges":                 "mengambil badge",
	"GetLeaderboard":            "mengambil leaderboard",
	"GetPhases":                 "mengambil tahap metabolik",
	"SetFastingByDuration":      "menyimpan jadwal puasa",
	"ScheduleFastingByDuration": "menjadwalkan puasa",
}

func errorLabel(method string) string {
	if label, ok := errorLabels[method]; ok {
		return label
	}
	return method
}

func (h *CommandHandler) handleBuka(phone string, args []string) (string, error) {
	if len(args) != 2 {
		return "❌ Format salah.\nGunakan: /buka DD-MM-YYYY HH:MM\nContoh: /buka 23-05-2026 18:30", nil
	}

	return h.callUsecase(phone, "BreakFastingAt", func() (string, error) {
		return h.usecase.BreakFastingAt(phone, args[0], args[1])
	})
}

func (h *CommandHandler) handleRiwayat(phone string, args []string) (string, error) {
	limit := 5
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return "❌ Format salah.\nGunakan: /riwayat [jumlah]\nContoh: /riwayat 5", nil
		}
		limit = n
	}
	return h.callUsecase(phone, "GetHistory", func() (string, error) {
		return h.usecase.GetHistory(phone, limit)
	})
}

func (h *CommandHandler) handlePuasa(phone string, args []string) (string, error) {
	if len(args) == 0 {
		return "❌ Durasi harus diisi. Gunakan: /puasa [durasi] [jam] [tanggal jam]\nContoh:\n  /puasa 16\n  /puasa 16 05:00\n  /puasa 16 14-06-2026 19:30", nil
	}
	durationHours, err := strconv.Atoi(args[0])
	if err != nil {
		return "❌ Durasi harus angka. Contoh: /puasa 16", nil
	}

	switch len(args) {
	case 1:
		return h.callUsecase(phone, "SetFastingByDuration", func() (string, error) {
			return h.usecase.SetFastingByDuration(phone, durationHours, false, "")
		})
	case 2:
		return h.callUsecase(phone, "SetFastingByDuration", func() (string, error) {
			return h.usecase.SetFastingByDuration(phone, durationHours, false, args[1])
		})
	case 3:
		return h.callUsecase(phone, "ScheduleFastingByDuration", func() (string, error) {
			return h.usecase.ScheduleFastingByDuration(phone, durationHours, false, args[1], args[2])
		})
	default:
		return "❌ Terlalu banyak argumen. Maksimal format: /puasa <durasi> <tanggal> <jam>\nContoh: /puasa 16 14-06-2026 19:30", nil
	}
}

func (h *CommandHandler) handlePuasaDry(phone string, args []string) (string, error) {
	if len(args) == 0 {
		return "❌ Durasi harus diisi. Gunakan: /puasa-dry [durasi] [jam] [tanggal jam]\nContoh:\n  /puasa-dry 18\n  /puasa-dry 18 05:00\n  /puasa-dry 18 14-06-2026 05:00", nil
	}
	durationHours, err := strconv.Atoi(args[0])
	if err != nil {
		return "❌ Durasi harus angka. Contoh: /puasa-dry 18", nil
	}

	switch len(args) {
	case 1:
		return h.callUsecase(phone, "SetFastingByDuration", func() (string, error) {
			return h.usecase.SetFastingByDuration(phone, durationHours, true, "")
		})
	case 2:
		return h.callUsecase(phone, "SetFastingByDuration", func() (string, error) {
			return h.usecase.SetFastingByDuration(phone, durationHours, true, args[1])
		})
	case 3:
		return h.callUsecase(phone, "ScheduleFastingByDuration", func() (string, error) {
			return h.usecase.ScheduleFastingByDuration(phone, durationHours, true, args[1], args[2])
		})
	default:
		return "❌ Terlalu banyak argumen. Maksimal format: /puasa-dry <durasi> <tanggal> <jam>\nContoh: /puasa-dry 18 14-06-2026 05:00", nil
	}
}

func getHelpText() string {
	return `🤖 *Fasting Bot — Teman Puasa Kamu*

✨ *4 Perintah Utama:*
1️⃣ */puasa [durasi] [jam]* — Mulai puasa, default 16 jam dari sekarang
2️⃣ */puasa [durasi] [tanggal] [jam]* — Jadwalkan untuk tanggal+j jam tertentu
3️⃣ */buka* — Catat buka puasa sekarang
4️⃣ */buka <tanggal> <jam>* — Catat buka puasa di waktu yang lalu (kalau lupa)

📋 *Perintah Pendukung:*
/daftar <nama> — Daftar sebagai user
/setname <nama> — Ubah nama
/panduan — Panduan edukatif tentang puasa
/pemula — Panduan IF bertahap dari 12 jam
/status — Cek status + fase metabolik saat ini
/fase — Lihat tahapan metabolik (fed → deep fast)
/motivasi — Suntikan semangat sesuai fase puasamu
/batalkan — Batalkan jadwal puasa aktif
/stats — Statistik puasa pribadi
/riwayat [n] — Riwayat sesi terakhir (default 5)
/badge — Koleksi badge & achievement
/leaderboard — Klasemen puasa grup
/bantuan — Tampilkan bantuan ini
/info — Info bot

⚡ *Preset Cepat — dari Pemula sampai Advanced:*
🌱 /if-1212, /if-1410   (IF ringan)
🔥 /if-168, /if-186, /if-204, /omad
💧 /water-24, /water-36, /water-48, /water-72
⚠️ /dry-24

💡 *Semua preset bisa dikasih waktu:*
/if-168 19:30   — mulai jam 7:30 malam
/water-48 14-06-2026 20:00   — jadwalkan 48 jam ke tanggal tertentu

💡 *Contoh praktis:*
/daftar kyomel
/puasa 16                        (IF 16 jam dari sekarang)
/puasa 16 05:00                  (IF 16 jam mulai jam 5)
/puasa 16 14-06-2026 19:30       (jadwalkan 16 jam, tgl 14 Juni jam 19:30)
/puasa-dry 18                    (Dry Fasting 18 jam dari sekarang)
/puasa-dry 18 14-06-2026 08:00   (dry dijadwalkan)
/buka                            (buka sekarang)
/buka 23-05-2026 18:30           (buka jam 18:30 tadi)

Konsisten dikit-dikit, hasilnya luar biasa. Yuk mulai! 💪`
}
