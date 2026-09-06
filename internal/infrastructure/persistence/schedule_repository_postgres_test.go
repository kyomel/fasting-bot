package persistence

import (
	"testing"

	"fasting-bot/internal/domain"
)

// TestPostgresScheduleRepositoryStreakReset verifies stale-streak reset
// against real PostgreSQL. Skips unless TEST_DATABASE_URL is set.
func TestPostgresScheduleRepositoryStreakReset(t *testing.T) {
	db := newPostgresTestDB(t)
	resetPostgresTables(t, db)
	repo := NewScheduleRepositoryPostgres(db)
	userRepo := NewUserRepositoryPostgres(db)

	user := &domain.User{Phone: "+628100000001", Name: "Streak"}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO user_fasting_stats (user_id, current_streak_days, longest_streak_days, last_streak_opened_at) VALUES ($1, 3, 3, '2026-05-22 18:00')`, string(user.ID)); err != nil {
		t.Fatal(err)
	}

	if err := repo.ResetStaleCurrentStreaks("2026-05-23", "2026-05-23 19:00"); err != nil {
		t.Fatal(err)
	}

	var got int
	if err := db.QueryRow(`SELECT current_streak_days FROM user_fasting_stats WHERE user_id = $1`, string(user.ID)).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("current_streak_days = %d, want 0", got)
	}
}

// TestPostgresNotificationTargets verifies elapsed/before-target selection
// and leaderboard ordering against real PostgreSQL.
// Skips unless TEST_DATABASE_URL is set.
func TestPostgresNotificationTargets(t *testing.T) {
	db := newPostgresTestDB(t)
	resetPostgresTables(t, db)
	repo := NewScheduleRepositoryPostgres(db)
	userRepo := NewUserRepositoryPostgres(db)

	seedNotificationUser := func(phone, name, jid string) domain.ID {
		t.Helper()
		u := &domain.User{Phone: phone, Name: name, JID: jid}
		if err := userRepo.Create(u); err != nil {
			t.Fatalf("Create user error = %v", err)
		}
		return u.ID
	}

	t.Run("elapsed dedups within current fast", func(t *testing.T) {
		resetPostgresTables(t, db)
		ari := seedNotificationUser("+628100000011", "Ari", "ari@s.whatsapp.net")
		bima := seedNotificationUser("+628100000012", "Bima", "bima@s.whatsapp.net")
		cici := seedNotificationUser("+628100000013", "Cici", "cici@s.whatsapp.net")

		seedSchedule := func(userID domain.ID, fastEnd, typeName string) {
			t.Helper()
			if _, err := db.Exec(`INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name, is_active) VALUES ($1, '2026-06-01 00:00', $2, $3, true)`, string(userID), fastEnd, typeName); err != nil {
				t.Fatal(err)
			}
		}
		seedSchedule(ari, "2026-06-01 18:00", "IF 18:6")
		seedSchedule(bima, "2026-06-01 18:00", "IF 18:6")
		seedSchedule(cici, "2026-06-01 12:00", "IF 12:12")

		if _, err := db.Exec(`INSERT INTO user_fasting_stats (user_id, current_streak_days) VALUES ($1, 2), ($2, 3), ($3, 4)`, string(ari), string(bima), string(cici)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO notification_logs (user_id, notification_type, sent_at) VALUES ($1, 'phase_fat_burning', '2026-05-31 12:00:00+07'), ($2, 'phase_fat_burning', '2026-06-01 12:01:00+07')`, string(ari), string(bima)); err != nil {
			t.Fatal(err)
		}

		targets, err := repo.FindUsersForElapsedNotification(domain.NotificationTypePhaseFatBurning, 12, "2026-06-01 12:30")
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 {
			t.Fatalf("FindUsersForElapsedNotification() returned %d targets, want 1: %#v", len(targets), targets)
		}
		if targets[0].UserID != ari || targets[0].FastingTypeName != "IF 18:6" || targets[0].CurrentStreakDays != 2 {
			t.Fatalf("unexpected target: %#v", targets[0])
		}
	})

	t.Run("before target respects lead hours", func(t *testing.T) {
		resetPostgresTables(t, db)
		ari := seedNotificationUser("+628100000021", "Ari", "ari2@s.whatsapp.net")
		bima := seedNotificationUser("+628100000022", "Bima", "bima2@s.whatsapp.net")

		if _, err := db.Exec(`INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name, is_active) VALUES ($1, '2026-06-01 00:00', '2026-06-02 00:00', 'Water Fasting 24 jam', true)`, string(ari)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO fasting_schedules (user_id, fast_start, fast_end, fasting_type_name, is_active) VALUES ($1, '2026-06-01 00:00', '2026-06-01 18:00', 'IF 18:6', true)`, string(bima)); err != nil {
			t.Fatal(err)
		}

		targets, err := repo.FindUsersBeforeTargetNotification(domain.PreBreakNotificationType(3), 3, "2026-06-01 21:30")
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 || targets[0].UserID != ari {
			t.Fatalf("FindUsersBeforeTargetNotification() = %#v, want only 24h water fast", targets)
		}
	})

	t.Run("leaderboard tie broken by user id", func(t *testing.T) {
		resetPostgresTables(t, db)
		ari := seedNotificationUser("+628100000031", "Ari", "ari3@s.whatsapp.net")
		bima := seedNotificationUser("+628100000032", "Bima", "bima3@s.whatsapp.net")

		schedule := &domain.FastingSchedule{UserID: ari, FastStart: "2026-06-01 00:00", FastEnd: "2026-06-01 18:00", FastingTypeName: "IF 18:6"}
		if err := repo.Create(schedule); err != nil {
			t.Fatal(err)
		}
		for _, uid := range []domain.ID{ari, bima} {
			record := &domain.FastingRecord{
				UserID:          uid,
				ScheduleID:      schedule.ID,
				FastingTypeName: "IF 18:6",
				FastStart:       "2026-06-01 00:00",
				PlannedFastEnd:  "2026-06-01 18:00",
				OpenedAt:        "2026-06-01 18:00",
				DurationMinutes: 1080,
				CompletedDate:   "2026-06-01",
				StreakQualified: true,
			}
			if err := repo.CreateFastingRecord(record); err != nil {
				t.Fatal(err)
			}
			if err := repo.UpsertFastingStats(record); err != nil {
				t.Fatal(err)
			}
		}

		entries, err := repo.FindFastingLeaderboard()
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("FindFastingLeaderboard() returned %d entries, want 2", len(entries))
		}
	})
}

// TestPostgresBadgeRepositoryAwardsIdempotently verifies badge award
// idempotency against real PostgreSQL. Skips unless TEST_DATABASE_URL is set.
func TestPostgresBadgeRepositoryAwardsIdempotently(t *testing.T) {
	db := newPostgresTestDB(t)
	resetPostgresTables(t, db)
	userRepo := NewUserRepositoryPostgres(db)
	repo := NewBadgeRepositoryPostgres(db)

	user := &domain.User{Phone: "+628100000041", Name: "Badge"}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}

	if err := repo.AwardBadges(user.ID, []domain.BadgeKey{domain.BadgeFirstFast, domain.BadgeFirstFast, domain.BadgeNightOwl}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AwardBadges(user.ID, []domain.BadgeKey{domain.BadgeFirstFast}); err != nil {
		t.Fatal(err)
	}

	earned, err := repo.EarnedBadges(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(earned) != 2 {
		t.Fatalf("EarnedBadges() = %#v, want 2 unique badges", earned)
	}
	if _, ok := earned[domain.BadgeFirstFast]; !ok {
		t.Fatalf("EarnedBadges() missing %q: %#v", domain.BadgeFirstFast, earned)
	}
	if _, ok := earned[domain.BadgeNightOwl]; !ok {
		t.Fatalf("EarnedBadges() missing %q: %#v", domain.BadgeNightOwl, earned)
	}
}
