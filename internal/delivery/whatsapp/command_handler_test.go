package whatsapp

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := map[string]string{
		"628123456789":                   "+628123456789",
		"+628123456789":                  "+628123456789",
		"08123456789":                    "+628123456789",
		"+62 812-3456-789":               "+628123456789",
		"628123456789@s.whatsapp.net":    "+628123456789",
		"628123456789:12@s.whatsapp.net": "+628123456789",
		"":                               "",
	}

	for input, want := range tests {
		if got := normalizePhone(input); got != want {
			t.Fatalf("normalizePhone(%q) = %q, want %q", input, got, want)
		}
	}
}
