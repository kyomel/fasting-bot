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

func TestIsAuthorizedAllowsAdminFromPrivateAndAllowedGroup(t *testing.T) {
	oldAdminNumber := config.AdminNumber
	oldAllowedGroupJID := config.AllowedGroupJID
	t.Cleanup(func() {
		config.AdminNumber = oldAdminNumber
		config.AllowedGroupJID = oldAllowedGroupJID
	})

	config.AdminNumber = "+628123456789"
	config.AllowedGroupJID = "120363000000000000@g.us"

	tests := map[string]struct {
		phone   string
		chatJID string
		isGroup bool
		want    bool
	}{
		"admin private chat": {
			phone:   "+628123456789",
			chatJID: "628123456789@s.whatsapp.net",
			isGroup: false,
			want:    true,
		},
		"admin allowed group chat": {
			phone:   "+628123456789",
			chatJID: "120363000000000000@g.us",
			isGroup: true,
			want:    true,
		},
		"admin other group chat": {
			phone:   "+628123456789",
			chatJID: "120363999999999999@g.us",
			isGroup: true,
			want:    false,
		},
		"non-admin private chat": {
			phone:   "+628987654321",
			chatJID: "628987654321@s.whatsapp.net",
			isGroup: false,
			want:    false,
		},
		"non-admin allowed group chat": {
			phone:   "+628987654321",
			chatJID: "120363000000000000@g.us",
			isGroup: true,
			want:    true,
		},
		"non-admin other group chat": {
			phone:   "+628987654321",
			chatJID: "120363999999999999@g.us",
			isGroup: true,
			want:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := isAuthorized(tt.phone, tt.chatJID, tt.isGroup); got != tt.want {
				t.Fatalf("isAuthorized(%q, %q, %v) = %v, want %v", tt.phone, tt.chatJID, tt.isGroup, got, tt.want)
			}
		})
	}
}
