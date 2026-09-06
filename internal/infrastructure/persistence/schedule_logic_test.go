package persistence

import (
	"testing"
	"time"
)

// TestNextCurrentStreakDays is a pure unit test (no database) covering the
// shared streak-increment rules used by the PostgreSQL schedule repository.
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

func mustParseStoredDateTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(storeDateTimeLayout, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
