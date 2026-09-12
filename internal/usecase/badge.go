package usecase

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

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
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return "", fmt.Errorf("gagal mengambil stats badge: %w", err)
	}
	if stats != nil {
		if _, err := u.evaluateAndAwardBadges(user.ID, nil); err != nil {
			log.Printf("[WARN] lazy badge backfill failed for user %s: %v", user.ID, err)
		}
	}

	earned, err := u.earnedBadges(user.ID)
	if err != nil {
		return "", fmt.Errorf("gagal mengambil badge: %w", err)
	}
	return formatBadgeCollection(stats, earned), nil
}

func (u *fastingUsecase) evaluateAndAwardBadges(userID domain.ID, record *domain.FastingRecord) ([]domain.Badge, error) {
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

func (u *fastingUsecase) awardBadges(userID domain.ID, keys []domain.BadgeKey) error {
	if u.badgeRepo == nil || len(keys) == 0 {
		return nil
	}
	return u.badgeRepo.AwardBadges(userID, keys)
}

func (u *fastingUsecase) earnedBadges(userID domain.ID) (map[domain.BadgeKey]struct{}, error) {
	if u.badgeRepo == nil {
		return map[domain.BadgeKey]struct{}{}, nil
	}
	return u.badgeRepo.EarnedBadges(userID)
}

func (u *fastingUsecase) badgeShelf(userID domain.ID) string {
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
