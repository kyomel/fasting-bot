package persistence

import (
	"database/sql"
	"testing"

	"fasting-bot/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

func TestBadgeRepositoryAwardsIdempotently(t *testing.T) {
	db := newBadgeTestDB(t)
	repo := NewBadgeRepository(db)

	if err := repo.AwardBadges(domain.ID("1"), []domain.BadgeKey{domain.BadgeFirstFast, domain.BadgeFirstFast, domain.BadgeNightOwl}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AwardBadges(domain.ID("1"), []domain.BadgeKey{domain.BadgeFirstFast}); err != nil {
		t.Fatal(err)
	}

	earned, err := repo.EarnedBadges(domain.ID("1"))
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

func newBadgeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE user_badges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			badge_key TEXT NOT NULL,
			awarded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, badge_key)
		);
	`); err != nil {
		t.Fatal(err)
	}
	return db
}
