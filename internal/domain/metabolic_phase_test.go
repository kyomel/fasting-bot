package domain

import "testing"

func TestPhaseForElapsedHoursBoundaries(t *testing.T) {
	tests := map[string]struct {
		hours float64
		want  string
	}{
		"negative clamps to fed": {-1, "fed"},
		"zero is fed":            {0, "fed"},
		"before 4 is fed":        {3.99, "fed"},
		"4 is post absorptive":   {4, "post_absorptive"},
		"before 12 is post":      {11.99, "post_absorptive"},
		"12 is fat burning":      {12, "fat_burning"},
		"before 18 is fat":       {17.99, "fat_burning"},
		"18 is ketosis":          {18, "ketosis"},
		"before 24 is ketosis":   {23.99, "ketosis"},
		"24 is deep fast":        {24, "deep_fast"},
		"after 24 is deep fast":  {72, "deep_fast"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := PhaseForElapsedHours(tt.hours); got.Key != tt.want {
				t.Fatalf("PhaseForElapsedHours(%v) = %q, want %q", tt.hours, got.Key, tt.want)
			}
		})
	}
}

func TestMetabolicPhaseNextPhase(t *testing.T) {
	next, ok := PhaseForElapsedHours(12).NextPhase()
	if !ok || next.Key != "ketosis" {
		t.Fatalf("fat_burning.NextPhase() = (%q, %v), want (ketosis, true)", next.Key, ok)
	}

	if _, ok := PhaseForElapsedHours(24).NextPhase(); ok {
		t.Fatal("deep_fast.NextPhase() should be false")
	}
}
