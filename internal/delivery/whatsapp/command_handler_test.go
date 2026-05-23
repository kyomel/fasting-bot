package whatsapp

import (
	"testing"

	"fasting-bot/internal/config"
)

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

func TestIsAuthorizedAllowsOnlyConfiguredGroup(t *testing.T) {
	oldAllowedGroupJID := config.AllowedGroupJID
	t.Cleanup(func() {
		config.AllowedGroupJID = oldAllowedGroupJID
	})

	config.AllowedGroupJID = "120363000000000000@g.us"

	tests := map[string]struct {
		chatJID string
		isGroup bool
		want    bool
	}{
		"allowed group chat": {
			chatJID: "120363000000000000@g.us",
			isGroup: true,
			want:    true,
		},
		"other group chat": {
			chatJID: "120363999999999999@g.us",
			isGroup: true,
			want:    false,
		},
		"private chat": {
			chatJID: "628987654321@s.whatsapp.net",
			isGroup: false,
			want:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := isAuthorized(tt.chatJID, tt.isGroup); got != tt.want {
				t.Fatalf("isAuthorized(%q, %v) = %v, want %v", tt.chatJID, tt.isGroup, got, tt.want)
			}
		})
	}

	config.AllowedGroupJID = ""
	if isAuthorized("120363000000000000@g.us", true) {
		t.Fatal("isAuthorized should reject group commands when AllowedGroupJID is empty")
	}
}
