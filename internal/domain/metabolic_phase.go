package domain

import "math"

// MetabolicPhase describes the broad metabolic state for an elapsed fast.
// Bounds are expressed in hours: MinHours is inclusive, MaxHours is exclusive.
type MetabolicPhase struct {
	Key      string
	Name     string
	Emoji    string
	MinHours float64
	MaxHours float64
}

var MetabolicPhases = []MetabolicPhase{
	{Key: "fed", Name: "Fed State", Emoji: "🍽️", MinHours: 0, MaxHours: 4},
	{Key: "post_absorptive", Name: "Post-Absorptive", Emoji: "⏳", MinHours: 4, MaxHours: 12},
	{Key: "fat_burning", Name: "Fat Burning", Emoji: "🔥", MinHours: 12, MaxHours: 18},
	{Key: "ketosis", Name: "Ketosis", Emoji: "⚡", MinHours: 18, MaxHours: 24},
	{Key: "deep_fast", Name: "Deep Fast", Emoji: "🧬", MinHours: 24, MaxHours: math.Inf(1)},
}

// PhaseForElapsedHours maps fasting duration to the current metabolic phase.
func PhaseForElapsedHours(hours float64) MetabolicPhase {
	if hours < 0 {
		hours = 0
	}

	for _, phase := range MetabolicPhases {
		if hours >= phase.MinHours && hours < phase.MaxHours {
			return phase
		}
	}

	return MetabolicPhases[len(MetabolicPhases)-1]
}

// NextPhase returns the following metabolic phase, if any.
func (p MetabolicPhase) NextPhase() (MetabolicPhase, bool) {
	for i, phase := range MetabolicPhases {
		if phase.Key == p.Key && i+1 < len(MetabolicPhases) {
			return MetabolicPhases[i+1], true
		}
	}

	return MetabolicPhase{}, false
}
