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

func TestSmartNotificationPlanForFastingTypes(t *testing.T) {
	tests := map[string]struct {
		name          string
		plannedHours  int
		wantHydration []int
		wantDrySafety []int
		wantPreBreak  int
	}{
		"short IF only gets final reminder": {
			name:          "IF 12:12",
			plannedHours:  12,
			wantHydration: nil,
			wantPreBreak:  1,
		},
		"IF 16 gets one hydration check": {
			name:          "IF 16:8",
			plannedHours:  16,
			wantHydration: []int{8},
			wantPreBreak:  2,
		},
		"OMAD stays sparse": {
			name:          "OMAD-2",
			plannedHours:  23,
			wantHydration: []int{12},
			wantPreBreak:  2,
		},
		"water fasting gets electrolyte-spaced checks": {
			name:          "Water Fasting 36 jam",
			plannedHours:  36,
			wantHydration: []int{16, 24},
			wantPreBreak:  3,
		},
		"dry fasting has safety checks without hydration": {
			name:          "Dry Fasting 24 jam",
			plannedHours:  24,
			wantHydration: nil,
			wantDrySafety: []int{8, 16},
			wantPreBreak:  3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := SmartNotificationPlanFor(tt.name, tt.plannedHours)
			assertIntSlice(t, got.HydrationReminderHours, tt.wantHydration)
			assertIntSlice(t, got.DrySafetyReminderHours, tt.wantDrySafety)
			if got.PreBreakLeadHours != tt.wantPreBreak {
				t.Fatalf("PreBreakLeadHours = %d, want %d", got.PreBreakLeadHours, tt.wantPreBreak)
			}
		})
	}
}

func assertIntSlice(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("slice = %#v, want %#v", got, want)
		}
	}
}
