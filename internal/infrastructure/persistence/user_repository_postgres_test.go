package persistence

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/infrastructure/database"
	"fasting-bot/internal/repository"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestPostgresUserRepositoryAccountColumns verifies the PostgreSQL user
// repository persists the OAuth-ready account fields (username, password_hash,
// email) alongside phone/name/jid, and that lookup by phone and id roundtrips
// them correctly. Skips unless TEST_DATABASE_URL is set.
func TestPostgresUserRepositoryAccountColumns(t *testing.T) {
	db := newPostgresTestDB(t)
	resetPostgresTables(t, db)
	repo := NewUserRepositoryPostgres(db)

	user := &domain.User{
		Username:     "kyomel",
		PasswordHash: "bcrypt-placeholder-hash",
		Phone:        "+628123456789",
		Email:        "kyomel@example.com",
		Name:         "Kyo",
		JID:          "kyomel@s.whatsapp.net",
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if user.ID == "" {
		t.Fatal("Create() did not assign user.ID")
	}

	byPhone, err := repo.FindByPhone(user.Phone)
	if err != nil {
		t.Fatalf("FindByPhone() error = %v", err)
	}
	assertUserAccount(t, byPhone, user)

	byID, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	assertUserAccount(t, byID, user)

	if err := repo.UpdateName(user.ID, "Kyo Baru"); err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}
	updated, err := repo.FindByPhone(user.Phone)
	if err != nil {
		t.Fatalf("FindByPhone() after update error = %v", err)
	}
	if updated.Name != "Kyo Baru" {
		t.Fatalf("UpdateName() did not persist, name = %q", updated.Name)
	}
	if updated.Email != user.Email || updated.Username != user.Username {
		t.Fatalf("UpdateName() clobbered account fields: %#v", updated)
	}
}

func assertUserAccount(t *testing.T, got, want *domain.User) {
	t.Helper()
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.PasswordHash != want.PasswordHash {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, want.PasswordHash)
	}
	if got.Phone != want.Phone {
		t.Errorf("Phone = %q, want %q", got.Phone, want.Phone)
	}
	if got.Email != want.Email {
		t.Errorf("Email = %q, want %q", got.Email, want.Email)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.JID != want.JID {
		t.Errorf("JID = %q, want %q", got.JID, want.JID)
	}
}

// TestPostgresScheduleRepositoryStreakUpsert verifies the streak upsert logic
// works against real PostgreSQL with UUID keys and timestamptz columns.
func TestPostgresScheduleRepositoryStreakUpsert(t *testing.T) {
	db := newPostgresTestDB(t)
	resetPostgresTables(t, db)
	userRepo := NewUserRepositoryPostgres(db)
	scheduleRepo := NewScheduleRepositoryPostgres(db)

	user := &domain.User{Phone: "+628111222333", Name: "Tester"}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	schedule := &domain.FastingSchedule{
		UserID:          user.ID,
		FastStart:       "2026-06-01 05:00",
		FastEnd:         "2026-06-01 21:00",
		FastingTypeName: "IF 16 jam",
	}
	if err := scheduleRepo.Create(schedule); err != nil {
		t.Fatalf("Create schedule error = %v", err)
	}
	if schedule.ID == "" {
		t.Fatal("Create schedule did not assign ID")
	}

	active, err := scheduleRepo.FindActiveByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindActiveByUserID() error = %v", err)
	}
	if active.ID != schedule.ID || active.UserID != user.ID || !active.IsActive {
		t.Fatalf("FindActiveByUserID() = %#v, want matching active schedule", active)
	}

	record := &domain.FastingRecord{
		UserID:          user.ID,
		ScheduleID:      schedule.ID,
		FastingTypeName: "IF 16 jam",
		FastStart:       "2026-06-01 05:00",
		PlannedFastEnd:  "2026-06-01 21:00",
		OpenedAt:        "2026-06-01 21:30",
		DurationMinutes: 990,
		CompletedDate:   "2026-06-01",
		StreakQualified: true,
	}
	if err := scheduleRepo.CreateFastingRecord(record); err != nil {
		t.Fatalf("CreateFastingRecord() error = %v", err)
	}
	if record.ID == "" {
		t.Fatal("CreateFastingRecord did not assign ID")
	}

	if err := scheduleRepo.UpsertFastingStats(record); err != nil {
		t.Fatalf("UpsertFastingStats() error = %v", err)
	}

	stats, err := scheduleRepo.FindFastingStatsByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindFastingStatsByUserID() error = %v", err)
	}
	if stats.TotalSessions != 1 || stats.TotalMinutes != 990 || stats.CurrentStreakDays != 1 {
		t.Fatalf("stats = %#v, want 1 session, 990 minutes, streak 1", stats)
	}

	if err := scheduleRepo.DeactivateByUserID(user.ID); err != nil {
		t.Fatalf("DeactivateByUserID() error = %v", err)
	}
	if _, err := scheduleRepo.FindActiveByUserID(user.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("FindActiveByUserID() after deactivate = %v, want repository.ErrNotFound", err)
	}
}

// newPostgresTestDB connects to TEST_DATABASE_URL and applies the goose
// migrations so tests run against the real schema.
func newPostgresTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.MigratePostgres(db); err != nil {
		t.Fatalf("MigratePostgres() error = %v", err)
	}
	return db
}

// resetPostgresTables wipes all rows so repeated test runs against the same
// database are deterministic. child tables first due to FK constraints.
func resetPostgresTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		TRUNCATE user_badges, notification_logs, fasting_records,
			user_fasting_stats, fasting_schedules, users RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("TRUNCATE error = %v", err)
	}
}
