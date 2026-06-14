package usecase

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

const (
	clockLayout       = "15:04"
	inputDateLayout   = "02-01-2006"
	storeLayout       = "2006-01-02 15:04"
	displayDateLayout = "02-01-2006 15:04"
)

const errCheckDataFormat = "gagal memeriksa data: %w"
const msgNotRegistered = "❌ Kamu belum terdaftar. Kirim /daftar <nama> dulu."
const errSaveScheduleFormat = "gagal menyimpan jadwal: %w"

type FastingUsecase interface {
	RegisterUser(phone, jid, name string) (string, error)
	SetName(phone, name string) (string, error)
	SetSchedule(phone, start, end string) (string, error)
	GetStatus(phone string) (string, error)
	CancelToday(phone string) (string, error)
	BreakFastingAt(phone, dateInput, openTime string) (string, error)
	DeleteSchedule(phone string) (string, error)
	GetStats(phone string) (string, error)
	GetBadges(phone string) (string, error)
	GetLeaderboard() (string, error)
	GetMotivation(phone string) (string, error)
	SetFastingType(phone string, typeID int, startTime string, durationHours int) (string, error)
	ScheduleFastingType(phone string, typeID int, dateInput, startTime string, durationHours int) (string, error)
	SetFastingByDuration(phone string, durationHours int, isDry bool, startTime string) (string, error)
	ScheduleFastingByDuration(phone string, durationHours int, isDry bool, dateInput, startTime string) (string, error)
}

var (
	motivationRandomMu sync.Mutex
	motivationRandom   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type fastingUsecase struct {
	userRepo         repository.UserRepository
	scheduleRepo     repository.ScheduleRepository
	notificationRepo repository.NotificationRepository
	badgeRepo        repository.BadgeRepository
}

func NewFastingUsecase(
	userRepo repository.UserRepository,
	scheduleRepo repository.ScheduleRepository,
	notificationRepo repository.NotificationRepository,
	badgeRepo repository.BadgeRepository,
) FastingUsecase {
	return &fastingUsecase{
		userRepo:         userRepo,
		scheduleRepo:     scheduleRepo,
		notificationRepo: notificationRepo,
		badgeRepo:        badgeRepo,
	}
}

func (u *fastingUsecase) RegisterUser(phone, jid, name string) (string, error) {
	if name == "" {
		return "❌ Nama harus diisi. Gunakan: /daftar <nama>\nContoh: /daftar kyomel", nil
	}

	existingUser, err := u.userRepo.FindByPhone(phone)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if existingUser != nil && existingUser.ID > 0 {
		registeredName := existingUser.Name
		if registeredName == "" {
			registeredName = existingUser.Phone
		}
		return fmt.Sprintf("✅ Akun sudah terdaftar!\nID: %d\nNama: %s\nNomor: %s\n\nGunakan /setname <nama> untuk mengubah nama.", existingUser.ID, registeredName, existingUser.Phone), nil
	}

	user := &domain.User{
		Phone: phone,
		Name:  name,
		JID:   jid,
	}
	if err := u.userRepo.Create(user); err != nil {
		return "", fmt.Errorf("gagal mendaftar: %w", err)
	}

	return fmt.Sprintf("🎉 *Selamat datang, %s!*\n"+
		"ID: %d\nNomor: %s\n\n"+
		"Bot ini bakal nemenin kamu tracking puasa — IF, OMAD, Water/Dry/Prolonged Fasting — plus ngirim notifikasi otomatis pas mulai & waktunya buka.\n\n"+
		"🚀 *Mulai dalam 30 detik:*\n"+
		"1️⃣ /panduan — baca ringkas jenis puasa & cara pakai bot\n"+
		"2️⃣ /puasa 14 atau /puasa 16 — mulai dari sekarang\n"+
		"3️⃣ /puasa 16 05:00 — mulai jam 5, durasi 16 jam\n"+
		"4️⃣ /puasa 16 23-05-2026 05:00 — jadwalkan ke tanggal tertentu\n\n"+
		"Saran buat pemula: mulai dari IF 14:10 atau IF 16:8. Body adaptation lebih halus. 💪", name, user.ID, phone), nil
}

func (u *fastingUsecase) SetName(phone, name string) (string, error) {
	if name == "" {
		return "❌ Nama harus diisi. Gunakan: /setname <nama baru>\nContoh: /setname kyomel baru", nil
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if err := u.userRepo.UpdateName(user.ID, name); err != nil {
		return "", fmt.Errorf("gagal mengubah nama: %w", err)
	}

	return fmt.Sprintf("✅ Nama berhasil diubah menjadi: %s", name), nil
}

func (u *fastingUsecase) SetSchedule(phone, start, end string) (string, error) {
	startTime, err := nextStartFromClock(start)
	if err != nil {
		return "❌ Format waktu mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
	}
	endClock, err := parseClock(end)
	if err != nil {
		return "❌ Format waktu selesai salah. Gunakan HH:MM (contoh: 18:00)", nil
	}
	endTime := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), endClock.Hour(), endClock.Minute(), 0, 0, config.Location)
	if !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menonaktifkan jadwal lama: %w", err)
	}

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startTime),
		FastEnd:         formatStoredTime(endTime),
		FastingTypeName: "Manual",
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf(errSaveScheduleFormat, err)
	}

	return fmt.Sprintf("✅ *Jadwal Fasting Tersimpan!*\nMulai: %s\nSelesai: %s\n\nKamu akan menerima notifikasi otomatis.", formatDisplayTime(startTime), formatDisplayTime(endTime)), nil
}

func (u *fastingUsecase) GetStatus(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	name := user.Name
	if name == "" {
		name = user.Phone
	}

	schedule, err := u.scheduleRepo.FindActiveByUserID(user.ID)
	if err != nil {
		return fmt.Sprintf("📋 *Status Akun*\nID: %d\nNama: %s\nNomor: %s\n\nBelum ada jadwal puasa aktif.\n\n"+
			"Atur sekarang:\n"+
			"• */panduan* — baca panduan ringkas\n"+
			"• */puasa 16* — mulai 16 jam dari sekarang\n"+
			"• */puasa 16 23-05-2026 05:00* — jadwalkan ke tanggal tertentu", user.ID, name, user.Phone), nil
	}
	fastingTypeName := schedule.FastingTypeName
	if fastingTypeName == "" {
		fastingTypeName = "Belum diketahui"
	}

	now := time.Now().In(config.Location)
	startTime, startHasDate := parseScheduleTime(schedule.FastStart, now)
	endTime, endHasDate := parseScheduleTime(schedule.FastEnd, now)
	if !startHasDate && !endHasDate && !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	var status string
	if now.Before(startTime) {
		status = fmt.Sprintf("⏳ Fasting dimulai dalam %s", formatDuration(startTime.Sub(now)))
	} else if now.Before(endTime) {
		status = fmt.Sprintf("🍽️ Sedang fasting!\nSudah berjalan: %s\nSisa: %s", formatDuration(now.Sub(startTime)), formatDuration(endTime.Sub(now)))
	} else {
		status = "✅ Fasting hari ini sudah selesai!"
	}

	return fmt.Sprintf("📋 *Status Fasting*\nID: %d\nNama: %s\nNomor: %s\nJenis Puasa: %s\nMulai: %s\nSelesai: %s\n\n%s", user.ID, name, user.Phone, fastingTypeName, formatScheduleDisplay(schedule.FastStart), formatScheduleDisplay(schedule.FastEnd), status), nil
}

func (u *fastingUsecase) CancelToday(phone string) (string, error) {
	now := time.Now().In(config.Location)
	return u.finishFastAt(phone, now, true)
}

func (u *fastingUsecase) BreakFastingAt(phone, dateInput, openTime string) (string, error) {
	openedAt, err := time.ParseInLocation(inputDateLayout+" "+clockLayout, dateInput+" "+openTime, config.Location)
	if err != nil {
		return "❌ Format /buka salah. Gunakan: /buka DD-MM-YYYY HH:MM\nContoh: /buka 23-05-2026 18:30", nil
	}

	now := time.Now().In(config.Location).Truncate(time.Minute)
	if openedAt.After(now) {
		return "❌ Waktu buka tidak boleh di masa depan.", nil
	}

	return u.finishFastAt(phone, openedAt, false)
}

func (u *fastingUsecase) finishFastAt(phone string, openedAt time.Time, allowCancelBeforeStart bool) (string, error) {
	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}
	if user == nil {
		return msgNotRegistered, nil
	}

	schedule, err := u.lookupActiveSchedule(user.ID)
	if err != nil {
		return "", err
	}
	if schedule == nil {
		return "ℹ️ Belum ada jadwal fasting aktif untuk dibuka.", nil
	}

	startTime, _ := parseScheduleTime(schedule.FastStart, openedAt)
	plannedEndTime, _ := parseScheduleTime(schedule.FastEnd, openedAt)
	if !plannedEndTime.After(startTime) {
		plannedEndTime = plannedEndTime.AddDate(0, 0, 1)
	}
	if openedAt.Before(startTime) {
		if !allowCancelBeforeStart {
			return fmt.Sprintf("❌ Waktu buka tidak boleh sebelum puasa mulai.\nMulai: %s\nBuka yang kamu input: %s", formatDisplayTime(startTime), formatDisplayTime(openedAt)), nil
		}
		return u.cancelBeforeStart(user.ID, startTime)
	}
	return u.breakFasting(user, schedule, startTime, plannedEndTime, openedAt)
}

func (u *fastingUsecase) lookupUser(phone string) (*domain.User, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf(errCheckDataFormat, err)
	}
	return user, nil
}

func (u *fastingUsecase) lookupActiveSchedule(userID int64) (*domain.FastingSchedule, error) {
	schedule, err := u.scheduleRepo.FindActiveByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal memeriksa jadwal: %w", err)
	}
	return schedule, nil
}

func (u *fastingUsecase) cancelBeforeStart(userID int64, startTime time.Time) (string, error) {
	if err := u.scheduleRepo.DeactivateByUserID(userID); err != nil {
		return "", fmt.Errorf("gagal membatalkan jadwal: %w", err)
	}
	if err := u.notificationRepo.LogNotification(userID, "cancelled"); err != nil {
		return "", fmt.Errorf("gagal mencatat pembatalan: %w", err)
	}
	return fmt.Sprintf("ℹ️ Jadwal fasting dibatalkan.\nMulai: %s\n\nKarena /buka dilakukan sebelum jam puasa mulai, durasi tidak dihitung ke /stats.", formatDisplayTime(startTime)), nil
}

func (u *fastingUsecase) breakFasting(user *domain.User, schedule *domain.FastingSchedule, startTime, plannedEndTime, openedAt time.Time) (string, error) {
	durationMinutes := int(openedAt.Sub(startTime).Minutes())
	if durationMinutes < 0 {
		durationMinutes = 0
	}
	streakQualified := !openedAt.Before(plannedEndTime)

	record := &domain.FastingRecord{
		UserID:          user.ID,
		ScheduleID:      schedule.ID,
		FastingTypeName: schedule.FastingTypeName,
		FastStart:       schedule.FastStart,
		PlannedFastEnd:  schedule.FastEnd,
		OpenedAt:        formatStoredTime(openedAt),
		DurationMinutes: durationMinutes,
		CompletedDate:   openedAt.Format("2006-01-02"),
		StreakQualified: streakQualified,
	}
	if err := u.scheduleRepo.CreateFastingRecord(record); err != nil {
		return "", fmt.Errorf("gagal menyimpan hasil buka puasa: %w", err)
	}
	if err := u.scheduleRepo.UpsertFastingStats(record); err != nil {
		return "", fmt.Errorf("gagal memperbarui stats puasa: %w", err)
	}
	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menutup jadwal: %w", err)
	}
	if err := u.notificationRepo.LogNotification(user.ID, "opened"); err != nil {
		return "", fmt.Errorf("gagal membatalkan: %w", err)
	}
	newBadges, err := u.evaluateAndAwardBadges(user.ID, record)
	if err != nil {
		log.Printf("[WARN] evaluateAndAwardBadges failed for user %d: %v", user.ID, err)
	}
	badgeUnlockMessage := formatBadgeUnlock(newBadges)

	durationHours := durationMinutes / 60
	extendedFastMessage := extendedFastCompletionMessage(durationHours, schedule.FastingTypeName)
	if streakQualified {
		message := fmt.Sprintf(
			"🎊 *Selesai — keren banget!*\n"+
				"Jenis: *%s*\n"+
				"⏱ Mulai: %s\n"+
				"🏁 Target: %s\n"+
				"🍽 Buka: %s\n"+
				"⌛ Total: *%s*\n\n"+
				"🔥 Streak bertambah! Adaptasi metabolik kamu makin solid setiap sesi.\n\n"+
				"%s"+
				"🥗 *Cara buka yang ramah tubuh:*\n%s\n\n"+
				"Cek progress: */stats* (pribadi) atau */leaderboard* (grup) 🏆",
			displayFastingTypeName(schedule.FastingTypeName),
			formatDisplayTime(startTime),
			formatDisplayTime(plannedEndTime),
			formatDisplayTime(openedAt),
			formatDurationWithDays(durationMinutes),
			extendedFastMessage,
			breakFastTipForDuration(durationHours),
		)
		if badgeUnlockMessage != "" {
			message += "\n\n" + badgeUnlockMessage
		}
		return message, nil
	}

	message := fmt.Sprintf(
		"✅ *Buka puasa tercatat!*\n"+
			"Jenis: *%s*\n"+
			"⏱ Mulai: %s\n"+
			"🏁 Target: %s\n"+
			"🍽 Buka: %s\n"+
			"⌛ Durasi: *%s*\n\n"+
			"Buka sebelum target — streak belum naik, tapi durasi tetap masuk stats. Semua jam puasa = waktu insulin rendah = manfaat tetap.\n\n"+
			"%s"+
			"🥗 *Cara buka yang ramah tubuh:*\n%s\n\n"+
			"Konsistensi > kesempurnaan. Set jadwal berikutnya: /puasa atau /puasa-dry 🌱",
		displayFastingTypeName(schedule.FastingTypeName),
		formatDisplayTime(startTime),
		formatDisplayTime(plannedEndTime),
		formatDisplayTime(openedAt),
		formatDurationWithDays(durationMinutes),
		extendedFastMessage,
		breakFastTipForDuration(durationHours),
	)
	if badgeUnlockMessage != "" {
		message += "\n\n" + badgeUnlockMessage
	}
	return message, nil
}

func extendedFastCompletionMessage(durationHours int, fastingTypeName string) string {
	if durationHours < 24 {
		return ""
	}
	if isDryFasting(fastingTypeName) {
		return "🧬 *Extended fast selesai!*\nKamu sudah melewati zona 24+ jam: insulin rendah, autophagy tinggi, dan mental endurance terlatih. Pulihkan tubuh bertahap dan jangan langsung makan besar.\n\n"
	}
	return "🧬 *Extended fast selesai!*\nKamu sudah masuk zona 24+ jam: autophagy tinggi, keton dominan, dan tubuh bekerja dalam mode repair. Refeed pelan-pelan; perhatikan elektrolit Na/K/Mg agar transisinya aman.\n\n"
}

// breakFastTipForDuration returns short, secular refeeding advice scaled to
// the actual fast length (insulin-spike avoidance for short fasts up to
// refeeding-syndrome warnings for prolonged fasts).
func breakFastTipForDuration(durationHours int) string {
	switch {
	case durationHours < 12:
		return "• Mulai dengan protein + lemak sebelum karbo — kurva insulin lebih landai.\n• Makan pelan, kasih waktu hormon kenyang (GLP-1, CCK) bekerja."
	case durationHours < 18:
		return "• Protein & lemak dulu (telur, alpukat, kacang), baru sayur, baru karbo.\n• Hindari gula/nasi sebagai gigitan pertama — bisa bikin reactive hypoglycemia.\n• 1-2 gelas air dulu sebelum makan."
	case durationHours < 24:
		return "• Mulai dengan kaldu hangat atau sup ringan — bangunkan sistem cerna pelan.\n• Lalu protein (telur/yogurt), baru karbo kompleks.\n• Tambahkan sejumput garam — natrium banyak terbuang saat puasa panjang."
	case durationHours < 48:
		return "• *Cairan dulu 30-60 menit:* kaldu tulang, air kelapa, sup sayur — bukan langsung makanan padat.\n• Lalu makanan lunak: telur rebus, pisang, alpukat.\n• Hindari: gorengan, gula, alkohol, sayur mentah.\n• Elektrolit penting: garam + kalium duluan."
	default:
		return "⚠️ *Refeed pelan-pelan — refeeding syndrome itu nyata di puasa >48 jam:*\n" +
			"• 4-6 jam pertama: cairan saja (kaldu + elektrolit Na/K/Mg).\n" +
			"• Lalu jus encer / bubur tipis / pisang lembut.\n" +
			"• Makanan utuh baru setelah 24 jam refeed bertahap.\n" +
			"• Hindari porsi besar — fosfat & magnesium bisa drop drastis kalau langsung makan banyak."
	}
}

func (u *fastingUsecase) DeleteSchedule(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if _, err := u.scheduleRepo.FindActiveByUserID(user.ID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("gagal memeriksa jadwal: %w", err)
		}
		return "ℹ️ Belum ada jadwal fasting yang aktif untuk dihapus.", nil
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menghapus jadwal: %w", err)
	}

	return "✅ Jadwal fasting berhasil dihapus. Jika cek /status, jadwal tidak akan tampil lagi.", nil
}

func (u *fastingUsecase) GetStats(phone string) (string, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}
	if err := u.refreshStaleCurrentStreaks(); err != nil {
		return "", fmt.Errorf("gagal memperbarui streak puasa: %w", err)
	}

	stats, err := u.scheduleRepo.FindFastingStatsByUserID(user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "📊 *Stats Puasa*\nBelum ada hasil puasa yang tercatat.\n\nGunakan /buka setelah puasa dimulai supaya durasi masuk ke stats.", nil
		}
		return "", fmt.Errorf("gagal mengambil stats: %w", err)
	}
	if stats.TotalSessions == 0 {
		return "📊 *Stats Puasa*\nBelum ada hasil puasa yang tercatat.\n\nGunakan /buka setelah puasa dimulai supaya durasi masuk ke stats.", nil
	}

	if _, err := u.evaluateAndAwardBadges(user.ID, nil); err != nil {
		log.Printf("[WARN] lazy badge backfill failed for user %d: %v", user.ID, err)
	}

	message := fmt.Sprintf("📊 *Stats Puasa %s*\nTotal sesi: %d\nStreak puasa saat ini: %d hari\nStreak puasa terpanjang: %d hari\nTotal waktu puasa: %s\n\nTerakhir buka: %s\nDurasi terakhir: %s", stats.Name, stats.TotalSessions, stats.CurrentStreakDays, stats.LongestStreakDays, formatDurationWithDays(stats.TotalMinutes), formatScheduleDisplay(stats.LastOpenedAt), formatDurationWithDays(stats.LastDurationMinutes))
	if shelf := u.badgeShelf(user.ID); shelf != "" {
		message += "\n\n🏅 *Badge:* " + shelf + "\nCek koleksi lengkap: /badge"
	}
	return message, nil
}

func (u *fastingUsecase) GetBadges(phone string) (string, error) {
	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}
	if user == nil {
		return msgNotRegistered, nil
	}

	var stats *domain.FastingStats
	if err := u.refreshStaleCurrentStreaks(); err != nil {
		return "", fmt.Errorf("gagal memperbarui streak puasa: %w", err)
	}
	stats, err = u.scheduleRepo.FindFastingStatsByUserID(user.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("gagal mengambil stats badge: %w", err)
	}
	if stats != nil {
		if _, err := u.evaluateAndAwardBadges(user.ID, nil); err != nil {
			log.Printf("[WARN] lazy badge backfill failed for user %d: %v", user.ID, err)
		}
	}

	earned, err := u.earnedBadges(user.ID)
	if err != nil {
		return "", fmt.Errorf("gagal mengambil badge: %w", err)
	}
	return formatBadgeCollection(stats, earned), nil
}

func (u *fastingUsecase) GetLeaderboard() (string, error) {
	if err := u.refreshStaleCurrentStreaks(); err != nil {
		return "", fmt.Errorf("gagal memperbarui streak puasa: %w", err)
	}

	entries, err := u.scheduleRepo.FindFastingLeaderboard()
	if err != nil {
		return "", fmt.Errorf("gagal mengambil leaderboard: %w", err)
	}
	if len(entries) == 0 {
		return "🏆 *Leaderboard Puasa*\nBelum ada data puasa.\n\nLeaderboard akan terisi setelah user menjalankan /buka setelah puasa dimulai.", nil
	}
	if err := u.awardBadges(entries[0].UserID, []domain.BadgeKey{domain.BadgeGroupChampion}); err != nil {
		log.Printf("[WARN] award group champion badge failed for user %d: %v", entries[0].UserID, err)
	}

	limit := len(entries)
	if limit > 10 {
		limit = 10
	}

	result := "🏆 *Leaderboard Puasa*\nPatokan ranking: total waktu puasa\n\n"
	for i := 0; i < limit; i++ {
		entry := entries[i]
		result += fmt.Sprintf("%d. %s\n   Streak puasa: %d hari\n   Total: %s\n", i+1, entry.Name, entry.CurrentStreakDays, formatDurationWithDays(entry.TotalMinutes))
		if i < limit-1 {
			result += "\n"
		}
	}

	return result, nil
}

func (u *fastingUsecase) GetMotivation(phone string) (string, error) {
	user, err := u.lookupUser(phone)
	if err != nil {
		return "", err
	}
	if user == nil {
		return msgNotRegistered, nil
	}

	name := displayUserName(user)
	schedule, err := u.lookupActiveSchedule(user.ID)
	if err != nil {
		return "", err
	}
	if schedule == nil {
		return fmt.Sprintf("✨ *Motivasi Puasa untuk %s*\n\n%s", name, pickMotivation(domain.MotivationNoSchedule())), nil
	}

	now := time.Now().In(config.Location)
	startTime, _ := parseScheduleTime(schedule.FastStart, now)
	endTime, _ := parseScheduleTime(schedule.FastEnd, now)
	if !endTime.After(startTime) {
		endTime = endTime.AddDate(0, 0, 1)
	}

	switch {
	case now.Before(startTime):
		return fmt.Sprintf(
			"✨ *Motivasi Puasa untuk %s*\n\n%s\n\n⏳ Mulai dalam: *%s*\n⏱ Jadwal mulai: %s\n\nSiapkan ritme, bukan tekanan. Kamu bisa. 💪",
			name,
			pickMotivation(domain.MotivationPreStart()),
			formatDuration(startTime.Sub(now)),
			formatDisplayTime(startTime),
		), nil
	case !now.Before(endTime):
		elapsed := now.Sub(startTime)
		body := pickMotivation(domain.MotivationTargetMet())
		if nudges := fastingSafetyNudges(schedule.FastingTypeName, elapsed); nudges != "" {
			body += "\n\n" + nudges
		}

		return fmt.Sprintf(
			"✨ *Motivasi Puasa untuk %s*\n\n%s\n\n🎯 Target: *%s*\n⏱ Durasi berjalan: *%s*\n\nKalau sudah buka, catat dengan */buka* ya — progress-mu layak terekam. 🏆",
			name,
			body,
			displayFastingTypeName(schedule.FastingTypeName),
			formatDuration(elapsed),
		), nil
	}

	elapsed := now.Sub(startTime)
	remaining := endTime.Sub(now)
	if remaining <= 2*time.Hour {
		body := pickMotivation(domain.MotivationNearTarget())
		if nudges := fastingSafetyNudges(schedule.FastingTypeName, elapsed); nudges != "" {
			body += "\n\n" + nudges
		}

		return fmt.Sprintf(
			"✨ *Motivasi Puasa untuk %s*\n\n%s\n\n⌛ Sudah berjalan: *%s*\n🏁 Sisa target: *%s*\n\nFinish strong — sebentar lagi kamu sampai. 🔥",
			name,
			body,
			formatDuration(elapsed),
			formatDuration(remaining),
		), nil
	}

	phase := domain.PhaseForElapsedHours(elapsed.Hours())
	body := pickMotivation(domain.MotivationForPhase(phase.Key))
	if nudges := fastingSafetyNudges(schedule.FastingTypeName, elapsed); nudges != "" {
		body += "\n\n" + nudges
	}

	return composeFastingMotivation(name, schedule, phase, elapsed, remaining, body), nil
}

func (u *fastingUsecase) SetFastingType(phone string, typeID int, startTime string, durationHours int) (string, error) {
	startDateTime, err := nextStartFromClock(startTime)
	if err != nil {
		return "❌ Format jam mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
	}
	return u.saveFastingTypeSchedule(phone, typeID, startDateTime, durationHours, false)
}

func (u *fastingUsecase) ScheduleFastingType(phone string, typeID int, dateInput, startTime string, durationHours int) (string, error) {
	startDateTime, err := time.ParseInLocation(inputDateLayout+" "+clockLayout, dateInput+" "+startTime, config.Location)
	if err != nil {
		return "❌ Format jadwal salah. Gunakan: /puasa <durasi> DD-MM-YYYY HH:MM\nContoh: /puasa 16 23-05-2026 16:00", nil
	}
	return u.saveFastingTypeSchedule(phone, typeID, startDateTime, durationHours, true)
}

func (u *fastingUsecase) SetFastingByDuration(phone string, durationHours int, isDry bool, startTime string) (string, error) {
	if durationHours < 1 || durationHours > 168 {
		return "❌ Durasi harus 1-168 jam.", nil
	}
	if isDry && durationHours > 48 {
		return "❌ Dry fasting maksimal 48 jam dulu karena terlalu ekstrem.", nil
	}

	var startDateTime time.Time
	var err error
	if startTime == "" {
		startDateTime = time.Now().In(config.Location).Truncate(time.Minute)
	} else {
		startDateTime, err = nextStartFromClock(startTime)
		if err != nil {
			return "❌ Format jam mulai salah. Gunakan HH:MM (contoh: 05:00)", nil
		}
	}

	fastingTypeName := buildFastingTypeName(durationHours, isDry)
	return u.saveFastingScheduleCore(phone, startDateTime, durationHours, fastingTypeName, false)
}

func (u *fastingUsecase) ScheduleFastingByDuration(phone string, durationHours int, isDry bool, dateInput, startTime string) (string, error) {
	if durationHours < 1 || durationHours > 168 {
		return "❌ Durasi harus 1-168 jam.", nil
	}
	if isDry && durationHours > 48 {
		return "❌ Dry fasting maksimal 48 jam dulu karena terlalu ekstrem.", nil
	}

	startDateTime, err := time.ParseInLocation(inputDateLayout+" "+clockLayout, dateInput+" "+startTime, config.Location)
	if err != nil {
		return "❌ Format jadwal salah. Gunakan: /jadwal <durasi> DD-MM-YYYY HH:MM\nContoh: /jadwal 16 20-06-2026 05:00", nil
	}

	fastingTypeName := buildFastingTypeName(durationHours, isDry)
	return u.saveFastingScheduleCore(phone, startDateTime, durationHours, fastingTypeName, true)
}

func buildFastingTypeName(durationHours int, isDry bool) string {
	if isDry {
		return fmt.Sprintf("Dry Fasting %d jam", durationHours)
	}
	if durationHours >= 24 {
		return fmt.Sprintf("Water Fasting %d jam", durationHours)
	}
	if durationHours >= 22 {
		return fmt.Sprintf("OMAD %d jam", durationHours)
	}
	return fmt.Sprintf("IF %d jam", durationHours)
}

func (u *fastingUsecase) saveFastingScheduleCore(phone string, startDateTime time.Time, durationHours int, fastingTypeName string, markElapsedNotifications bool) (string, error) {
	endDateTime := startDateTime.Add(time.Duration(durationHours) * time.Hour)

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if err := u.scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		return "", fmt.Errorf("gagal menonaktifkan jadwal lama: %w", err)
	}

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       formatStoredTime(startDateTime),
		FastEnd:         formatStoredTime(endDateTime),
		FastingTypeName: fastingTypeName,
	}
	if err := u.scheduleRepo.Create(schedule); err != nil {
		return "", fmt.Errorf(errSaveScheduleFormat, err)
	}
	if markElapsedNotifications {
		u.markElapsedNotifications(user.ID, startDateTime, endDateTime)
	}

	return fmt.Sprintf(
		"🎯 *Jadwal %s tersimpan!*\n"+
			"⏱ Mulai: *%s*\n"+
			"🏁 Buka: *%s* (%d jam)\n\n"+
			"%s\n\n"+
			"Kamu akan dapat notifikasi otomatis saat mulai & saat waktunya buka.\n"+
			"_Mau ganti? Jalankan /puasa lagi — jadwal lama otomatis dimatikan._",
		fastingTypeName, formatDisplayTime(startDateTime), formatDisplayTime(endDateTime),
		durationHours, scheduleTeaserForDuration(durationHours, fastingTypeName),
	), nil
}

func (u *fastingUsecase) saveFastingTypeSchedule(phone string, typeID int, startDateTime time.Time, durationHours int, markElapsedNotifications bool) (string, error) {
	fastHours, fastingTypeName, validationMessage := fastingTypeScheduleDetails(typeID, durationHours)
	if validationMessage != "" {
		return validationMessage, nil
	}
	return u.saveFastingScheduleCore(phone, startDateTime, fastHours, fastingTypeName, markElapsedNotifications)
}

// scheduleTeaserForDuration shows a 1-line, secular preview of what the body
// will do during the planned fast — appears in the schedule-saved confirmation.
func scheduleTeaserForDuration(durationHours int, fastingTypeName string) string {
	if isDryFasting(fastingTypeName) {
		return "💡 Dry fasting butuh ekstra sadar sinyal tubuh: pantau pusing, lemas berlebihan, dan suhu tubuh. Kalau tubuh memberi sinyal keras, batalkan dengan bijak."
	}

	switch {
	case durationHours <= 0:
		return "Tubuhmu akan mulai turunkan insulin & masuk mode repair."
	case durationHours <= 14:
		return "💡 Di rentang ini: insulin turun, glikogen mulai dipakai, sel mulai shift ke mode \"akses\" lemak."
	case durationHours <= 20:
		return "💡 Di rentang ini: *metabolic switch* aktif, ketogenesis dimulai, autophagy ramp-up — Yoshinori Ohsumi dapat Nobel buat proses ini."
	case durationHours <= 36:
		return "💡 Di rentang ini: HGH spike (proteksi otot), IGF-1 turun, autophagy puncak. Wajib hidrasi + sedikit garam."
	case durationHours <= 72:
		return "💡 Di rentang ini: sinyal regenerasi stem cell aktif (Valter Longo, Cell 2014). ⚠️ Elektrolit wajib: Na, K, Mg tiap hari."
	default:
		return "💡 Zona prolonged fasting (>72 jam): imun reset & deep cellular cleanup. ⚠️ Wajib hidrasi + elektrolit, refeed pelan-pelan, konsultasi dokter kalau ada kondisi medis."
	}
}

func fastingTypeScheduleDetails(typeID int, durationHours int) (int, string, string) {
	fastingType, err := domain.GetFastingTypeByID(typeID)
	if err != nil {
		return 0, "", "❌ Jenis puasa tidak ditemukan. Gunakan /panduan atau command baru /puasa."
	}

	switch fastingType.ID {
	case 1, 2, 3, 4, 5, 6, 7:
		return fastingType.FastHours, fastingType.Name, ""
	case 8:
		if durationHours != 24 && durationHours != 36 && durationHours != 48 && durationHours != 56 && durationHours != 64 && durationHours != 72 {
			return 0, "", "❌ Durasi Water Fasting harus 24, 36, 48, 56, 64, atau 72 jam."
		}
		return durationHours, fmt.Sprintf("Water Fasting %d jam", durationHours), ""
	case 9:
		if durationHours < 1 || durationHours > 48 {
			return 0, "", "❌ Durasi Dry Fasting harus 1-48 jam dulu karena terlalu ekstrem."
		}
		return durationHours, fmt.Sprintf("Dry Fasting %d jam", durationHours), ""
	case 10:
		if durationHours < 24 || durationHours > 168 {
			return 0, "", "❌ Prolonged Fasting metode water fasting harus 24-168 jam."
		}
		return durationHours, fmt.Sprintf("Prolonged Fasting (Water) %d jam", durationHours), ""
	}

	return 0, "", "❌ Jenis puasa tidak ditemukan. Gunakan /panduan atau command baru /puasa."
}

func (u *fastingUsecase) markElapsedNotifications(userID int64, startDateTime, endDateTime time.Time) {
	now := time.Now().In(config.Location).Truncate(time.Minute)
	if !startDateTime.After(now) {
		_ = u.notificationRepo.LogNotification(userID, "start")
	}
	if !endDateTime.After(now) {
		_ = u.notificationRepo.LogNotification(userID, "end")
	}
}

func parseClock(value string) (time.Time, error) {
	return time.ParseInLocation(clockLayout, value, config.Location)
}

func nextStartFromClock(value string) (time.Time, error) {
	clock, err := parseClock(value)
	if err != nil {
		return time.Time{}, err
	}

	now := time.Now().In(config.Location)
	nowMinute := now.Truncate(time.Minute)
	candidate := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, config.Location)
	if candidate.Before(nowMinute) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate, nil
}

func calculateEndDateTime(start time.Time, hours int) time.Time {
	return start.Add(time.Duration(hours) * time.Hour)
}

func (u *fastingUsecase) refreshStaleCurrentStreaks() error {
	now := time.Now().In(config.Location)
	return u.scheduleRepo.ResetStaleCurrentStreaks(now.Format("2006-01-02"), formatStoredTime(now))
}

func (u *fastingUsecase) evaluateAndAwardBadges(userID int64, record *domain.FastingRecord) ([]domain.Badge, error) {
	if u.badgeRepo == nil {
		return nil, nil
	}

	stats, err := u.scheduleRepo.FindFastingStatsByUserID(userID)
	if err != nil {
		return nil, err
	}
	earned, err := u.badgeRepo.EarnedBadges(userID)
	if err != nil {
		return nil, err
	}

	newKeys := domain.EvaluateBadges(stats, record, earned)
	if len(newKeys) == 0 {
		return nil, nil
	}
	if err := u.badgeRepo.AwardBadges(userID, newKeys); err != nil {
		return nil, err
	}

	badges := make([]domain.Badge, 0, len(newKeys))
	for _, key := range newKeys {
		if badge, ok := domain.GetBadge(key); ok {
			badges = append(badges, badge)
		}
	}
	return badges, nil
}

func (u *fastingUsecase) awardBadges(userID int64, keys []domain.BadgeKey) error {
	if u.badgeRepo == nil || len(keys) == 0 {
		return nil
	}
	return u.badgeRepo.AwardBadges(userID, keys)
}

func (u *fastingUsecase) earnedBadges(userID int64) (map[domain.BadgeKey]struct{}, error) {
	if u.badgeRepo == nil {
		return map[domain.BadgeKey]struct{}{}, nil
	}
	return u.badgeRepo.EarnedBadges(userID)
}

func (u *fastingUsecase) badgeShelf(userID int64) string {
	earned, err := u.earnedBadges(userID)
	if err != nil || len(earned) == 0 {
		return ""
	}

	parts := make([]string, 0, len(earned))
	for _, badge := range domain.Badges() {
		if _, ok := earned[badge.Key]; ok {
			parts = append(parts, badge.Emoji)
		}
	}
	return strings.Join(parts, " ")
}

func formatBadgeUnlock(badges []domain.Badge) string {
	if len(badges) == 0 {
		return ""
	}
	if len(badges) == 1 {
		badge := badges[0]
		return fmt.Sprintf("🎖️ *Badge baru terbuka!*\n%s *%s* — %s\n\nCek semua badge: /badge", badge.Emoji, badge.Name, badge.Description)
	}

	lines := []string{"🎖️ *Badge baru terbuka!*"}
	for _, badge := range badges {
		lines = append(lines, fmt.Sprintf("• %s *%s*", badge.Emoji, badge.Name))
	}
	lines = append(lines, "", "Cek semua badge: /badge")
	return strings.Join(lines, "\n")
}

func formatBadgeCollection(stats *domain.FastingStats, earned map[domain.BadgeKey]struct{}) string {
	progresses := domain.BadgeProgresses(stats, earned)
	earnedLines := make([]string, 0, len(progresses))
	lockedLines := make([]string, 0, len(progresses))

	for _, progress := range progresses {
		line := fmt.Sprintf("%s *%s*", progress.Badge.Emoji, progress.Badge.Name)
		if progress.Earned {
			earnedLines = append(earnedLines, "✅ "+line)
			continue
		}

		lockedLine := "🔒 " + line
		if progress.Target > 0 {
			lockedLine += fmt.Sprintf(" (Progress: %d/%d)", progress.Current, progress.Target)
		}
		lockedLines = append(lockedLines, lockedLine)
	}

	message := "🏆 *Koleksi Badge*\n\n"
	if len(earnedLines) == 0 {
		message += "Belum ada badge terbuka. Selesaikan sesi puasa pertama untuk mulai mengumpulkan.\n"
	} else {
		message += strings.Join(earnedLines, "\n") + "\n"
	}
	if len(lockedLines) > 0 {
		message += "\n🔒 *Terkunci:*\n" + strings.Join(lockedLines, "\n")
	}
	return message
}

func formatStoredTime(t time.Time) string {
	return t.In(config.Location).Format(storeLayout)
}

func formatDisplayTime(t time.Time) string {
	return t.In(config.Location).Format(displayDateLayout)
}

func formatScheduleDisplay(value string) string {
	t, err := time.ParseInLocation(storeLayout, value, config.Location)
	if err != nil {
		return value
	}
	return formatDisplayTime(t)
}

func parseScheduleTime(value string, now time.Time) (time.Time, bool) {
	if t, err := time.ParseInLocation(storeLayout, value, config.Location); err == nil {
		return t, true
	}

	clock, err := parseClock(value)
	if err != nil {
		return now, false
	}
	return time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, config.Location), false
}

func formatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	totalHours := totalMinutes / 60
	minutes := totalMinutes % 60
	days := totalHours / 24
	hours := totalHours % 24

	if days > 0 {
		return fmt.Sprintf("%d hari %d jam %d menit (total: %d jam %d menit)", days, hours, minutes, totalHours, minutes)
	}
	if totalHours > 0 {
		return fmt.Sprintf("%d jam %d menit", totalHours, minutes)
	}
	return fmt.Sprintf("%d menit", minutes)
}

func formatDurationWithDays(totalMinutes int) string {
	if totalMinutes < 0 {
		totalMinutes = 0
	}

	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60
	totalHours := totalMinutes / 60

	if days > 0 {
		return fmt.Sprintf("%d hari %d jam %d menit (total: %d jam %d menit)", days, hours, minutes, totalHours, minutes)
	}
	return fmt.Sprintf("%d jam %d menit", hours, minutes)
}

func displayFastingTypeName(name string) string {
	if name == "" {
		return "Belum diketahui"
	}
	return name
}

func displayUserName(user *domain.User) string {
	if user.Name != "" {
		return user.Name
	}
	return user.Phone
}

func composeFastingMotivation(name string, schedule *domain.FastingSchedule, phase domain.MetabolicPhase, elapsed, remaining time.Duration, body string) string {
	message := fmt.Sprintf(
		"✨ *Motivasi Puasa untuk %s*\n\n"+
			"%s *%s*\n"+
			"Jenis: *%s*\n"+
			"⌛ Sudah berjalan: *%s*\n"+
			"🏁 Sisa target: *%s*\n\n"+
			"%s",
		name,
		phase.Emoji,
		phase.Name,
		displayFastingTypeName(schedule.FastingTypeName),
		formatDuration(elapsed),
		formatDuration(remaining),
		body,
	)

	if nextPhase, ok := phase.NextPhase(); ok {
		message += fmt.Sprintf("\n\n➡️ Berikutnya: *%s %s* sekitar jam ke-%.0f. Satu langkah lagi.", nextPhase.Emoji, nextPhase.Name, nextPhase.MinHours)
	}

	return message
}

func pickMotivation(pool []string) string {
	if len(pool) == 0 {
		return "💪 Kamu sedang membangun konsistensi. Lanjutkan pelan-pelan, satu jam demi satu jam."
	}

	motivationRandomMu.Lock()
	defer motivationRandomMu.Unlock()
	return pool[motivationRandom.Intn(len(pool))]
}

func isDryFasting(name string) bool {
	return strings.Contains(strings.ToLower(name), "dry fasting")
}

func fastingSafetyNudges(name string, elapsed time.Duration) string {
	if isDryFasting(name) {
		return ""
	}

	nudges := pickMotivation(domain.HydrationNudges())
	if isLongWaterFast(name, elapsed) {
		nudges += "\n\n" + pickMotivation(domain.ElectrolyteNudges())
	}
	return nudges
}

func isLongWaterFast(name string, elapsed time.Duration) bool {
	if elapsed < 24*time.Hour {
		return false
	}
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "water fasting") || strings.Contains(lowerName, "prolonged")
}
