package persistence

import (
	"database/sql"
	"testing"
	"time"

	"fasting-bot/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

func TestNextCurrentStreakDays(t *testing.T) {
	last := mustParseStoredDateTime(t, "2026-05-22 18:00")

	tests := map[string]struct {
		current      int
		lastOpenedAt string
		openedAt     time.Time
		qualified    bool
		want         int
	}{
		"first qualified buka starts streak": {
			openedAt:  mustParseStoredDateTime(t, "2026-05-22 18:00"),
			qualified: true,
			want:      1,
		},
		"qualified buka within 24 hours increments by one": {
			current:      2,
			lastOpenedAt: "2026-05-22 18:00",
			openedAt:     last.Add(23 * time.Hour),
			qualified:    true,
			want:         3,
		},
		"qualified buka after 24 hours restarts streak": {
			current:      2,
			lastOpenedAt: "2026-05-22 18:00",
			openedAt:     last.Add(25 * time.Hour),
			qualified:    true,
			want:         1,
		},
		"early buka does not increment active streak": {
			current:      2,
			lastOpenedAt: "2026-05-22 18:00",
			openedAt:     last.Add(12 * time.Hour),
			qualified:    false,
			want:         2,
		},
		"early buka after 24 hours resets stale streak": {
			current:      2,
			lastOpenedAt: "2026-05-22 18:00",
			openedAt:     last.Add(25 * time.Hour),
			qualified:    false,
			want:         0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := nextCurrentStreakDays(tt.current, tt.lastOpenedAt, tt.openedAt, tt.qualified)
			if got != tt.want {
				t.Fatalf("nextCurrentStreakDays() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResetStaleCurrentStreaksUsesTwentyFourHours(t *testing.T) {
	db := newStreakTestDB(t)
	repo := &ScheduleRepositorySQLite{db: db}

	if _, err := db.Exec(`
		INSERT INTO user_fasting_stats (user_id, current_streak_days, longest_streak_days, last_streak_opened_at)
		VALUES (1, 3, 3, '2026-05-22 18:00'), (2, 2, 2, '2026-05-23 17:30')
	`); err != nil {
		t.Fatal(err)
	}

	if err := repo.ResetStaleCurrentStreaks("2026-05-23", "2026-05-23 19:00"); err != nil {
		t.Fatal(err)
	}

	assertCurrentStreak(t, db, 1, 0)
	assertCurrentStreak(t, db, 2, 2)
}

func TestResetStaleCurrentStreaksKeepsActiveFasting(t *testing.T) {
	db := newStreakTestDB(t)
	repo := &ScheduleRepositorySQLite{db: db}

	if _, err := db.Exec(`
		INSERT INTO user_fasting_stats (user_id, current_streak_days, longest_streak_days, last_streak_opened_at)
		VALUES (1, 3, 3, '2026-05-22 18:00');
		INSERT INTO fasting_schedules (user_id, fast_start, fast_end, is_active)
		VALUES (1, '2026-05-23 18:00', '2026-05-24 10:00', 1)
	`); err != nil {
		t.Fatal(err)
	}

	if err := repo.ResetStaleCurrentStreaks("2026-05-23", "2026-05-23 19:00"); err != nil {
		t.Fatal(err)
	}

	assertCurrentStreak(t, db, 1, 3)
}

func TestFindUsersForElapsedNotificationDedupsWithinCurrentFast(t *testing.T) {
	db := newNotificationTargetTestDB(t)
	repo := &ScheduleRepositorySQLite{db: db}

	if _, err := db.Exec(`
		INSERT INTO users (id, phone, name, jid) VALUES
			(1, '+6201', 'Ari', 'ari@s.whatsapp.net'),
			(2, '+6202', 'Bima', 'bima@s.whatsapp.net'),
			(3, '+6203', 'Cici', 'cici@s.whatsapp.net');
		INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name, is_active) VALUES
			(1, '2026-06-01 00:00', '2026-06-01 18:00', 'IF 18:6', 1),
			(2, '2026-06-01 00:00', '2026-06-01 18:00', 'IF 18:6', 1),
			(3, '2026-06-01 00:00', '2026-06-01 12:00', 'IF 12:12', 1);
		INSERT INTO user_fasting_stats (user_id, current_streak_days) VALUES (1, 2), (2, 3), (3, 4);
		INSERT INTO notification_logs (user_id, notification_type, sent_at) VALUES
			(1, 'phase_fat_burning', '2026-05-31 12:00:00'),
			(2, 'phase_fat_burning', '2026-06-01 12:01:00');
	`); err != nil {
		t.Fatal(err)
	}

	targets, err := repo.FindUsersForElapsedNotification(domain.NotificationTypePhaseFatBurning, 12, "2026-06-01 12:30")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("FindUsersForElapsedNotification() returned %d targets, want 1: %#v", len(targets), targets)
	}
	if targets[0].UserID != 1 || targets[0].FastingTypeName != "IF 18:6" || targets[0].CurrentStreakDays != 2 {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}

func TestFindUsersNearTargetNotificationDedupsWithinCurrentFast(t *testing.T) {
	db := newNotificationTargetTestDB(t)
	repo := &ScheduleRepositorySQLite{db: db}

	if _, err := db.Exec(`
		INSERT INTO users (id, phone, name, jid) VALUES
			(1, '+6201', 'Ari', 'ari@s.whatsapp.net'),
			(2, '+6202', 'Bima', 'bima@s.whatsapp.net'),
			(3, '+6203', 'Cici', 'cici@s.whatsapp.net');
		INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name, is_active) VALUES
			(1, '2026-06-01 00:00', '2026-06-01 18:00', 'IF 18:6', 1),
			(2, '2026-06-01 00:00', '2026-06-01 18:00', 'IF 18:6', 1),
			(3, '2026-06-01 00:00', '2026-06-01 20:00', 'IF 20:4', 1);
		INSERT INTO notification_logs (user_id, notification_type, sent_at) VALUES
			(2, 'near_target', '2026-06-01 16:01:00');
	`); err != nil {
		t.Fatal(err)
	}

	targets, err := repo.FindUsersNearTargetNotification(domain.NotificationTypeNearTarget, "2026-06-01 16:30")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("FindUsersNearTargetNotification() returned %d targets, want 1: %#v", len(targets), targets)
	}
	if targets[0].UserID != 1 {
		t.Fatalf("unexpected target: %#v", targets[0])
	}
}

func mustParseStoredDateTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(storeDateTimeLayout, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func newStreakTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	queries := []string{
		`CREATE TABLE user_fasting_stats (
			user_id INTEGER PRIMARY KEY,
			current_streak_days INTEGER NOT NULL DEFAULT 0,
			longest_streak_days INTEGER NOT NULL DEFAULT 0,
			last_streak_opened_at TEXT NOT NULL DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE fasting_schedules (
			user_id INTEGER NOT NULL,
			fast_start TEXT NOT NULL,
			fast_end TEXT NOT NULL,
			is_active BOOLEAN DEFAULT 1
		);`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func assertCurrentStreak(t *testing.T, db *sql.DB, userID int64, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT current_streak_days FROM user_fasting_stats WHERE user_id = ?`, userID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("current_streak_days for user %d = %d, want %d", userID, got, want)
	}
}

func newNotificationTargetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	queries := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			phone TEXT NOT NULL,
			name TEXT,
			jid TEXT NOT NULL
		);`,
		`CREATE TABLE fasting_schedules (
			user_id INTEGER NOT NULL,
			fast_start TEXT NOT NULL,
			fast_end TEXT NOT NULL,
			fasting_type_name TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT 1
		);`,
		`CREATE TABLE user_fasting_stats (
			user_id INTEGER PRIMARY KEY,
			current_streak_days INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE notification_logs (
			user_id INTEGER NOT NULL,
			notification_type TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
