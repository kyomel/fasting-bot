package domain

import "strings"

type BadgeKey string

const (
	BadgeFirstFast        BadgeKey = "first_fast"
	BadgeFirstOMAD        BadgeKey = "first_omad"
	BadgeSevenDayStreak   BadgeKey = "seven_day_streak"
	BadgeThirtyDayWarrior BadgeKey = "thirty_day_warrior"
	BadgeHydrationMaster  BadgeKey = "hydration_master"
	BadgeNightOwl         BadgeKey = "night_owl"
	BadgeConsistencyKing  BadgeKey = "consistency_king"
	BadgeGroupChampion    BadgeKey = "group_champion"
)

type Badge struct {
	Key         BadgeKey
	Emoji       string
	Name        string
	Description string
	Target      int
}

type BadgeProgress struct {
	Badge   Badge
	Earned  bool
	Current int
	Target  int
}

var badgeCatalogue = []Badge{
	{Key: BadgeFirstFast, Emoji: "🌱", Name: "First Fast", Description: "Selesaikan sesi puasa pertama", Target: 1},
	{Key: BadgeFirstOMAD, Emoji: "⚡", Name: "First OMAD", Description: "Selesaikan puasa OMAD pertama kali", Target: 1},
	{Key: BadgeSevenDayStreak, Emoji: "🔥", Name: "7-Day Streak", Description: "Puasa 7 hari berturut-turut", Target: 7},
	{Key: BadgeThirtyDayWarrior, Emoji: "🏆", Name: "30-Day Warrior", Description: "Streak 30 hari", Target: 30},
	{Key: BadgeHydrationMaster, Emoji: "💧", Name: "Hydration Master", Description: "Minum cukup selama puasa dalam sehari", Target: 1},
	{Key: BadgeNightOwl, Emoji: "🕐", Name: "Night Owl", Description: "Selesaikan puasa 24+ jam", Target: 24},
	{Key: BadgeConsistencyKing, Emoji: "🎯", Name: "Consistency King", Description: "Puasa tepat waktu 10x", Target: 10},
	{Key: BadgeGroupChampion, Emoji: "🥇", Name: "Group Champion", Description: "Pernah menjadi peringkat #1 leaderboard grup", Target: 1},
}

func Badges() []Badge {
	badges := make([]Badge, len(badgeCatalogue))
	copy(badges, badgeCatalogue)
	return badges
}

func GetBadge(key BadgeKey) (Badge, bool) {
	for _, badge := range badgeCatalogue {
		if badge.Key == key {
			return badge, true
		}
	}
	return Badge{}, false
}

func EvaluateBadges(stats *FastingStats, lastRecord *FastingRecord, earned map[BadgeKey]struct{}) []BadgeKey {
	if stats == nil {
		return nil
	}

	var newBadges []BadgeKey
	check := func(key BadgeKey, condition bool) {
		if !condition {
			return
		}
		if _, ok := earned[key]; ok {
			return
		}
		newBadges = append(newBadges, key)
	}

	check(BadgeFirstFast, stats.TotalSessions >= 1)
	check(BadgeSevenDayStreak, bestStreak(stats) >= 7)
	check(BadgeThirtyDayWarrior, bestStreak(stats) >= 30)
	check(BadgeNightOwl, stats.LastDurationMinutes >= 24*60 || recordDurationMinutes(lastRecord) >= 24*60)
	check(BadgeFirstOMAD, completedOMAD(lastRecord))

	return newBadges
}

func BadgeProgresses(stats *FastingStats, earned map[BadgeKey]struct{}) []BadgeProgress {
	progresses := make([]BadgeProgress, 0, len(badgeCatalogue))
	for _, badge := range badgeCatalogue {
		_, isEarned := earned[badge.Key]
		progress := BadgeProgress{Badge: badge, Earned: isEarned, Current: progressValue(badge.Key, stats, isEarned), Target: badge.Target}
		if progress.Current > progress.Target {
			progress.Current = progress.Target
		}
		progresses = append(progresses, progress)
	}
	return progresses
}

func bestStreak(stats *FastingStats) int {
	if stats == nil {
		return 0
	}
	if stats.LongestStreakDays > stats.CurrentStreakDays {
		return stats.LongestStreakDays
	}
	return stats.CurrentStreakDays
}

func progressValue(key BadgeKey, stats *FastingStats, earned bool) int {
	if earned {
		if badge, ok := GetBadge(key); ok {
			return badge.Target
		}
		return 1
	}
	if stats == nil {
		return 0
	}

	switch key {
	case BadgeFirstFast:
		return stats.TotalSessions
	case BadgeSevenDayStreak, BadgeThirtyDayWarrior:
		return bestStreak(stats)
	case BadgeNightOwl:
		return stats.LastDurationMinutes / 60
	default:
		return 0
	}
}

func recordDurationMinutes(record *FastingRecord) int {
	if record == nil {
		return 0
	}
	return record.DurationMinutes
}

func completedOMAD(record *FastingRecord) bool {
	if record == nil {
		return false
	}
	return record.StreakQualified && isOMADFasting(record.FastingTypeName)
}

func isOMADFasting(name string) bool {
	return strings.Contains(strings.ToLower(name), "omad")
}
