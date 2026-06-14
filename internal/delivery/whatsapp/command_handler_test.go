package whatsapp

import (
	"strings"
	"testing"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
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

func TestPemulaCommandReturnsBeginnerGuide(t *testing.T) {
	h := &CommandHandler{}

	got, err := h.processCommand("+628123456789", "628123456789@s.whatsapp.net", "/pemula")
	if err != nil {
		t.Fatalf("processCommand(/pemula) returned error: %v", err)
	}

	for _, want := range []string{"Panduan IF untuk Pemula", "/puasa 12", "Dry fasting", "maksimal 48 jam", "/if-1212"} {
		if !strings.Contains(got, want) {
			t.Fatalf("/pemula response should contain %q, got: %q", want, got)
		}
	}
}

func TestListPuasaCommandIsRemoved(t *testing.T) {
	h := &CommandHandler{}

	got, err := h.processCommand("+628123456789", "628123456789@s.whatsapp.net", "/list-puasa")
	if err != nil {
		t.Fatalf("processCommand(/list-puasa) returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("/list-puasa should be removed and return empty response, got: %q", got)
	}
}

func TestPanduanIncludesWaterFastingQuickCommands(t *testing.T) {
	guide := domain.GetPanduan()

	for _, want := range []string{"/water-24", "/water-36", "/water-48", "/water-56", "/water-64", "/water-72", "/if-1212", "/if-1410", "/if-168", "/if-186", "/if-204", "/omad", "Preset Cepat"} {
		if !strings.Contains(guide, want) {
			t.Fatalf("panduan should contain %q, got: %q", want, guide)
		}
	}
}

func TestProactiveDryFastingMessageHasSafetyWarningWithoutHydration(t *testing.T) {
	msg := buildPhaseMilestoneMessage(repository.NotificationTarget{
		UserID:          1,
		Name:            "Kyo",
		FastStart:       "2026-06-01 00:00",
		FastEnd:         "2026-06-01 18:00",
		FastingTypeName: "Dry Fasting 18 jam",
	}, domain.ProactivePhaseNotifications[0], "2026-06-01 12:00")

	if !strings.Contains(msg, "Dry fasting") {
		t.Fatalf("dry proactive message should include safety warning: %q", msg)
	}
	for _, unwanted := range []string{"💧", "hidrasi", "minum"} {
		if strings.Contains(strings.ToLower(msg), unwanted) {
			t.Fatalf("dry proactive message = %q, should not contain %q", msg, unwanted)
		}
	}
}

func TestLongWaterHydrationReminderIncludesElectrolytes(t *testing.T) {
	msg := buildHydrationReminderMessage(repository.NotificationTarget{
		UserID:          1,
		Name:            "Kyo",
		FastStart:       "2026-06-01 00:00",
		FastEnd:         "2026-06-02 12:00",
		FastingTypeName: "Water Fasting 36 jam",
	}, 28, "2026-06-02 04:00")

	if !strings.Contains(strings.ToLower(msg), "elektrolit") {
		t.Fatalf("long water hydration reminder should include electrolytes: %q", msg)
	}
}
