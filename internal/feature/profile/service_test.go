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
// has to carry identity — no XP, no level, no frames.
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
	if got.UnlockedMaxFrameLevel != wantLevel {
		t.Errorf("UnlockedMaxFrameLevel = %d, want %d", got.UnlockedMaxFrameLevel, wantLevel)
	}
}

func TestGetProfileDefaultsEquippedFrameToCurrentLevel(t *testing.T) {
	svc, _ := newTestService(200)

	got, err := svc.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.EquippedFrameLevel != got.Level {
		t.Errorf("EquippedFrameLevel = %d, want it to default to the current level %d", got.EquippedFrameLevel, got.Level)
	}
}

func TestGetProfileUserNotFound(t *testing.T) {
	users := &fakeUserRepo{}
	svc := NewService(users, progression.NewService(progression.NewMockRepository()))

	if _, err := svc.GetProfile(context.Background(), 99); !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// The unlock rule moved into progression; profile must surface its error
// unchanged so the handler can still map it to a 400.
func TestSetEquippedFrameSurfacesTheLockedError(t *testing.T) {
	svc, progRepo := newTestService(0)
	progRepo.Progressions[1].Level = 5

	if _, err := svc.SetEquippedFrame(context.Background(), 1, 6); !errors.Is(err, progression.ErrFrameLocked) {
		t.Fatalf("expected progression.ErrFrameLocked, got %v", err)
	}
	if progRepo.Progressions[1].EquippedFrameLevel != nil {
		t.Error("a locked frame must not be persisted")
	}
}

func TestSetEquippedFrameReturnsTheUpdatedProfile(t *testing.T) {
	svc, progRepo := newTestService(0)
	progRepo.Progressions[1].Level = 5

	got, err := svc.SetEquippedFrame(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("SetEquippedFrame: %v", err)
	}
	if got.EquippedFrameLevel != 3 {
		t.Errorf("EquippedFrameLevel = %d, want 3", got.EquippedFrameLevel)
	}
	if got.Name != "Learner" {
		t.Errorf("the identity half should still be present, got Name = %q", got.Name)
	}
}

func TestSetEquippedFrameRequestValidation(t *testing.T) {
	for _, frame := range []int{0, -1, leveling.MaxLevel + 1} {
		if err := (&SetEquippedFrameRequest{FrameLevel: frame}).Validate(); err == nil {
			t.Errorf("frame_level %d should fail validation", frame)
		}
	}
	for _, frame := range []int{1, 50, leveling.MaxLevel} {
		if err := (&SetEquippedFrameRequest{FrameLevel: frame}).Validate(); err != nil {
			t.Errorf("frame_level %d should pass validation, got %v", frame, err)
		}
	}
}
