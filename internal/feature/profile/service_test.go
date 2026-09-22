package profile

import (
	"context"
	"errors"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

// fakeUserRepo is an in-memory auth.Repository. Only FindByID matters
// here; the rest satisfy the interface.
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

func (f *fakeUserRepo) GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (bool, int, int, error) {
	return false, 0, 0, nil
}

var _ auth.Repository = (*fakeUserRepo)(nil)

// The users.level column is a denormalised cache that XP grants write to.
// The profile must derive level from lifetime XP instead, so a stale
// cache can never be what the user sees.
func TestGetProfileRecomputesLevelFromXPNotTheCachedColumn(t *testing.T) {
	repo := &fakeUserRepo{user: &auth.User{ID: 1, Name: "Learner", XP: 200, Level: 99}}
	svc := NewService(repo)

	got, err := svc.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}

	wantLevel, wantCurrentXP, wantToNext := leveling.LevelForXP(200)
	if got.Level != wantLevel {
		t.Errorf("Level = %d, want %d — the stale users.level=99 must be ignored", got.Level, wantLevel)
	}
	if got.CurrentLevelXP != wantCurrentXP {
		t.Errorf("CurrentLevelXP = %d, want %d", got.CurrentLevelXP, wantCurrentXP)
	}
	if got.XPToNextLevel != wantToNext {
		t.Errorf("XPToNextLevel = %d, want %d", got.XPToNextLevel, wantToNext)
	}
}

func TestGetProfileUserNotFound(t *testing.T) {
	svc := NewService(&fakeUserRepo{})
	if _, err := svc.GetProfile(context.Background(), 99); !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
