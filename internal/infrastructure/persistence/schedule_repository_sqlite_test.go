package persistence

import (
	"database/sql"
	"testing"
	"time"

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
