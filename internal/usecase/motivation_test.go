package usecase

import (
	"database/sql"
	"math/rand"
	"strings"
	"testing"
	"time"

	"fasting-bot/internal/config"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

func TestGetMotivation(t *testing.T) {
	now := time.Now().In(config.Location).Truncate(time.Minute)
	user := &domain.User{ID: 1, Phone: "+628123456789", Name: "Kyo"}

	tests := map[string]struct {
		user     *domain.User
		schedule *domain.FastingSchedule
		want     []string
		wantNot  []string
	}{
		"not registered": {
			want: []string{"belum terdaftar", "/daftar"},
		},
		"registered without active schedule": {
			user: user,
			want: []string{"Motivasi Puasa", "jadwal"},
		},
		"pre start schedule": {
			user:     user,
			schedule: testSchedule(now.Add(time.Hour), now.Add(17*time.Hour), "IF 16:8"),
			want:     []string{"Mulai dalam", "Jadwal mulai"},
		},
		"active fat burning phase": {
			user:     user,
			schedule: testSchedule(now.Add(-13*time.Hour), now.Add(5*time.Hour), "IF 18:6"),
			want:     []string{"Fat Burning", "Sudah berjalan", "Sisa target"},
		},
		"near target takes precedence": {
			user:     user,
			schedule: testSchedule(now.Add(-14*time.Hour), now.Add(90*time.Minute), "IF 16:8"),
			want:     []string{"Finish strong", "Sisa target"},
		},
		"near target long water fasting includes electrolytes": {
			user:     user,
			schedule: testSchedule(now.Add(-35*time.Hour), now.Add(90*time.Minute), "Water Fasting 36 jam"),
			want:     []string{"Finish strong", "elektrolit"},
		},
		"near target dry fasting does not suggest water": {
			user:     user,
			schedule: testSchedule(now.Add(-10*time.Hour), now.Add(90*time.Minute), "Dry Fasting 12 jam"),
			want:     []string{"Finish strong"},
			wantNot:  []string{"💧", "hidrasi", "air"},
		},
		"target met but not opened": {
			user:     user,
			schedule: testSchedule(now.Add(-18*time.Hour), now.Add(-time.Minute), "IF 16:8"),
			want:     []string{"Target:", "/buka"},
		},
		"dry fasting does not suggest water": {
			user:     user,
			schedule: testSchedule(now.Add(-13*time.Hour), now.Add(5*time.Hour), "Dry Fasting 18 jam"),
			want:     []string{"Fat Burning"},
			wantNot:  []string{"💧", "hidrasi", "air"},
		},
		"long water fasting includes electrolytes": {
			user:     user,
			schedule: testSchedule(now.Add(-25*time.Hour), now.Add(11*time.Hour), "Water Fasting 36 jam"),
			want:     []string{"Deep Fast", "elektrolit"},
		},
		"prolonged fasting includes electrolytes": {
			user:     user,
			schedule: testSchedule(now.Add(-25*time.Hour), now.Add(23*time.Hour), "Prolonged Fasting (Water) 48 jam"),
			want:     []string{"Deep Fast", "elektrolit"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			uc := NewFastingUsecase(
				&motivationUserRepo{user: tt.user},
				&motivationScheduleRepo{schedule: tt.schedule},
				&motivationNotificationRepo{},
				&motivationBadgeRepo{},
			)

			got, err := uc.GetMotivation("+628123456789")
			if err != nil {
				t.Fatalf("GetMotivation() error = %v", err)
			}

			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("GetMotivation() = %q, want it to contain %q", got, want)
				}
			}
			for _, unwanted := range tt.wantNot {
				if strings.Contains(strings.ToLower(got), strings.ToLower(unwanted)) {
					t.Fatalf("GetMotivation() = %q, should not contain %q", got, unwanted)
				}
			}
		})
	}
}

func TestPickMotivationVariesWithPool(t *testing.T) {
	motivationRandomMu.Lock()
	motivationRandom = rand.New(rand.NewSource(1))
	motivationRandomMu.Unlock()

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		seen[pickMotivation([]string{"satu", "dua"})] = true
	}

	if len(seen) != 2 {
		t.Fatalf("pickMotivation should vary over repeated calls, saw %v", seen)
	}
}

func TestScheduleTeaserForDryFastingDoesNotSuggestWater(t *testing.T) {
	got := scheduleTeaserForDuration(36, "Dry Fasting 36 jam")
	for _, unwanted := range []string{"hidrasi", "air", "minum"} {
		if strings.Contains(strings.ToLower(got), unwanted) {
			t.Fatalf("scheduleTeaserForDuration() = %q, should not contain %q", got, unwanted)
		}
	}
}

func testSchedule(start, end time.Time, fastingTypeName string) *domain.FastingSchedule {
	return &domain.FastingSchedule{
		ID:              10,
		UserID:          1,
		FastStart:       formatStoredTime(start),
		FastEnd:         formatStoredTime(end),
		FastingTypeName: fastingTypeName,
		IsActive:        true,
	}
}

type motivationUserRepo struct {
	user *domain.User
}

func (r *motivationUserRepo) Create(user *domain.User) error             { return nil }
func (r *motivationUserRepo) UpdateName(userID int64, name string) error { return nil }
func (r *motivationUserRepo) FindByPhone(phone string) (*domain.User, error) {
	if r.user == nil {
		return nil, sql.ErrNoRows
	}
	return r.user, nil
}
func (r *motivationUserRepo) FindByID(id int64) (*domain.User, error) { return r.user, nil }

type motivationScheduleRepo struct {
	schedule *domain.FastingSchedule
}

func (r *motivationScheduleRepo) Create(schedule *domain.FastingSchedule) error { return nil }
func (r *motivationScheduleRepo) DeactivateByUserID(userID int64) error         { return nil }
func (r *motivationScheduleRepo) FindActiveByUserID(userID int64) (*domain.FastingSchedule, error) {
	if r.schedule == nil {
		return nil, sql.ErrNoRows
	}
	return r.schedule, nil
}
func (r *motivationScheduleRepo) CreateFastingRecord(record *domain.FastingRecord) error { return nil }
func (r *motivationScheduleRepo) UpsertFastingStats(record *domain.FastingRecord) error  { return nil }
func (r *motivationScheduleRepo) ResetStaleCurrentStreaks(currentDate, currentDateTime string) error {
	return nil
}
func (r *motivationScheduleRepo) FindFastingStatsByUserID(userID int64) (*domain.FastingStats, error) {
	return nil, sql.ErrNoRows
}
func (r *motivationScheduleRepo) FindRecentFastingRecords(userID int64, limit int) ([]domain.FastingRecord, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) FindFastingLeaderboard() ([]domain.FastingLeaderboardEntry, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) CleanupOldFastingRecords(cutoff string) (int64, error) {
	return 0, nil
}
func (r *motivationScheduleRepo) FindUsersToNotifyStart(currentTime, currentDate, currentDateTime string) ([]repository.NotificationTarget, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) FindUsersToNotifyEnd(currentTime, currentDate, currentDateTime string) ([]repository.NotificationTarget, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) FindUsersForElapsedNotification(notificationType string, triggerAfterHours int, currentDateTime string) ([]repository.NotificationTarget, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) FindUsersBeforeTargetNotification(notificationType string, leadHours int, currentDateTime string) ([]repository.NotificationTarget, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) FindUsersWithActiveFasting(currentDateTime string) ([]repository.NotificationTarget, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) FindUsersWithExpiredStreaks(currentDateTime string) ([]repository.ExpiredStreakTarget, error) {
	return nil, nil
}
func (r *motivationScheduleRepo) ResetStreakByUserID(userID int64) error { return nil }

type motivationNotificationRepo struct{}

func (r *motivationNotificationRepo) LogNotification(userID int64, notificationType string) error {
	return nil
}

type motivationBadgeRepo struct{}

func (r *motivationBadgeRepo) EarnedBadges(userID int64) (map[domain.BadgeKey]struct{}, error) {
	return map[domain.BadgeKey]struct{}{}, nil
}

func (r *motivationBadgeRepo) AwardBadges(userID int64, keys []domain.BadgeKey) error {
	return nil
}
