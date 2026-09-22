package profile

import (
	"context"
	"errors"
	"testing"

	"github/DoanCongPho/game-arena/internal/feature/progression"
	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

// fakeUserRepo is an in-memory auth.Repository. Since the split it only
// has to carry identity — no XP, no level.
type fakeUserRepo struct {
	user *auth.User
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*auth.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, auth.ErrUserNotFound
	}
	return f.user, nil
}
func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*auth.User, error) {
	return nil, auth.ErrUserNotFound
}

func (f *fakeUserRepo) CreateUser(ctx context.Context, u *auth.User) (*auth.User, error) {
	return u, nil
}

func (f *fakeUserRepo) UpdateUser(ctx context.Context, u *auth.User) (*auth.User, error) {
	return u, nil
}

var _ auth.Repository = (*fakeUserRepo)(nil)

func newTestService(xp int) (Service, *progression.MockRepository) {
	users := &fakeUserRepo{user: &auth.User{ID: 1, Name: "Learner", ImageURL: "/avatar.png"}}
	progRepo := progression.NewMockRepository()
	progRepo.Seed(1, xp)
	return NewService(users, progression.NewService(progRepo)), progRepo
}

// The response is a join of two sources; this checks both halves land in
// the right fields.
func TestGetProfileJoinsIdentityAndProgression(t *testing.T) {
	svc, _ := newTestService(200)

	got, err := svc.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}

	// identity half
	if got.ID != 1 || got.Name != "Learner" || got.ImageURL != "/avatar.png" {
		t.Errorf("identity fields wrong: %+v", got)
	}

	// progression half
	wantLevel, wantCurrent, wantNext := leveling.LevelForXP(200)
	if got.Level != wantLevel {
		t.Errorf("Level = %d, want %d", got.Level, wantLevel)
	}
	if got.XP != 200 {
		t.Errorf("XP = %d, want 200", got.XP)
	}
	if got.CurrentLevelXP != wantCurrent {
		t.Errorf("CurrentLevelXP = %d, want %d", got.CurrentLevelXP, wantCurrent)
	}
	if got.XPToNextLevel != wantNext {
		t.Errorf("XPToNextLevel = %d, want %d", got.XPToNextLevel, wantNext)
	}
}

func TestGetProfileUserNotFound(t *testing.T) {
	users := &fakeUserRepo{}
	svc := NewService(users, progression.NewService(progression.NewMockRepository()))

	if _, err := svc.GetProfile(context.Background(), 99); !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
