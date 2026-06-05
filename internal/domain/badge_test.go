package domain

import "testing"

func TestEvaluateBadgesFromExistingStats(t *testing.T) {
	stats := &FastingStats{
		TotalSessions:       18,
		CurrentStreakDays:   3,
		LongestStreakDays:   7,
		LastDurationMinutes: 24 * 60,
	}

	got := EvaluateBadges(stats, nil, map[BadgeKey]struct{}{
		BadgeFirstFast: {},
	})

	want := map[BadgeKey]bool{
		BadgeSevenDayStreak: true,
		BadgeNightOwl:       true,
	}
	if len(got) != len(want) {
		t.Fatalf("EvaluateBadges() = %#v, want %d badges", got, len(want))
	}
	for _, key := range got {
		if !want[key] {
			t.Fatalf("EvaluateBadges() unexpected key %q in %#v", key, got)
		}
	}
}

func TestEvaluateBadgesFromCompletedRecord(t *testing.T) {
	stats := &FastingStats{TotalSessions: 1}
	record := &FastingRecord{FastingTypeName: "OMAD-1", DurationMinutes: 25 * 60, StreakQualified: true}

	got := EvaluateBadges(stats, record, map[BadgeKey]struct{}{})

	want := map[BadgeKey]bool{
		BadgeFirstFast: true,
		BadgeFirstOMAD: true,
		BadgeNightOwl:  true,
	}
	if len(got) != len(want) {
		t.Fatalf("EvaluateBadges() = %#v, want %d badges", got, len(want))
	}
	for _, key := range got {
		if !want[key] {
			t.Fatalf("EvaluateBadges() unexpected key %q in %#v", key, got)
		}
	}
}

func TestEvaluateBadgesDoesNotAwardIncompleteOMAD(t *testing.T) {
	stats := &FastingStats{TotalSessions: 1}
	record := &FastingRecord{FastingTypeName: "OMAD-1", DurationMinutes: 60, StreakQualified: false}

	got := EvaluateBadges(stats, record, map[BadgeKey]struct{}{})

	for _, key := range got {
		if key == BadgeFirstOMAD {
			t.Fatalf("incomplete OMAD should not award %q: %#v", BadgeFirstOMAD, got)
		}
	}
}
