package persistence

import (
	"database/sql"
	"strings"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type BadgeRepositorySQLite struct {
	db *sql.DB
}

func NewBadgeRepository(db *sql.DB) repository.BadgeRepository {
	return &BadgeRepositorySQLite{db: db}
}

func (r *BadgeRepositorySQLite) EarnedBadges(userID domain.ID) (map[domain.BadgeKey]struct{}, error) {
	rows, err := r.db.Query(`SELECT badge_key FROM user_badges WHERE user_id = ?`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	earned := make(map[domain.BadgeKey]struct{})
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		earned[domain.BadgeKey(key)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return earned, nil
}

func (r *BadgeRepositorySQLite) AwardBadges(userID domain.ID, keys []domain.BadgeKey) error {
	if len(keys) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*2)
	for _, key := range keys {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, string(userID), string(key))
	}

	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO user_badges (user_id, badge_key) VALUES `+strings.Join(placeholders, ", "),
		args...,
	)
	return err
}
