package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type BadgeRepositoryPostgres struct {
	db *sql.DB
}

func NewBadgeRepositoryPostgres(db *sql.DB) repository.BadgeRepository {
	return &BadgeRepositoryPostgres{db: db}
}

func (r *BadgeRepositoryPostgres) EarnedBadges(userID domain.ID) (map[domain.BadgeKey]struct{}, error) {
	rows, err := r.db.Query(`SELECT badge_key FROM user_badges WHERE user_id = $1`, string(userID))
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

func (r *BadgeRepositoryPostgres) AwardBadges(userID domain.ID, keys []domain.BadgeKey) error {
	if len(keys) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*2)
	for i, key := range keys {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, string(userID), string(key))
	}

	_, err := r.db.Exec(
		`INSERT INTO user_badges (user_id, badge_key) VALUES `+strings.Join(placeholders, ", ")+
			` ON CONFLICT (user_id, badge_key) DO NOTHING`,
		args...,
	)
	return err
}
