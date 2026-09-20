package profile

import (
	"context"
	"errors"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

// fakeUserRepo is an in-memory auth.Repository. Only FindByID and
// UpdateEquippedFrame matter here; the rest satisfy the interface.
type fakeUserRepo struct {
	user            *auth.User
	equippedWritten *int
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*auth.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, auth.ErrUserNotFound
	}
	return f.user, nil
}
func (f *fakeUserRepo) UpdateEquippedFrame(ctx context.Context, userID uint64, frameLevel int) error {
	if f.user == nil || f.user.ID != userID {
		return auth.ErrUserNotFound
	}
	f.equippedWritten = &frameLevel
	return nil
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
	if got.UnlockedMaxFrameLevel != wantLevel {
		t.Errorf("UnlockedMaxFrameLevel = %d, want %d", got.UnlockedMaxFrameLevel, wantLevel)
	}
}

// With nothing equipped the UI shows the highest frame the user has
// earned, rather than nothing at all.
func TestGetProfileDefaultsEquippedFrameToCurrentLevel(t *testing.T) {
	repo := &fakeUserRepo{user: &auth.User{ID: 1, XP: 200}}
	svc := NewService(repo)

	got, err := svc.GetProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.EquippedFrameLevel != got.Level {
		t.Errorf("EquippedFrameLevel = %d, want it to default to the current level %d", got.EquippedFrameLevel, got.Level)
	}
}

func TestGetProfileUserNotFound(t *testing.T) {
	svc := NewService(&fakeUserRepo{})
	if _, err := svc.GetProfile(context.Background(), 99); !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestSetEquippedFrameEnforcesTheLevelGate(t *testing.T) {
	// Level 5 by the cached column; the gate reads u.Level.
	user := &auth.User{ID: 1, XP: 1000, Level: 5}

	tests := []struct {
		name       string
		frameLevel int
		wantErr    error
	}{
		{"a frame below the current level", 3, nil},
		{"the frame at exactly the current level", 5, nil},
		{"a frame one above the current level", 6, ErrFrameLocked},
		{"a frame far above the current level", 100, ErrFrameLocked},
		{"frame level zero", 0, ErrFrameLocked},
		{"a negative frame level", -1, ErrFrameLocked},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeUserRepo{user: user}
			svc := NewService(repo)

			_, err := svc.SetEquippedFrame(context.Background(), 1, tc.frameLevel)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if repo.equippedWritten != nil {
					t.Errorf("a locked frame must not be persisted, but %d was written", *repo.equippedWritten)
				}
				return
			}

			if err != nil {
				t.Fatalf("SetEquippedFrame: %v", err)
			}
			if repo.equippedWritten == nil || *repo.equippedWritten != tc.frameLevel {
				t.Errorf("persisted frame = %v, want %d", repo.equippedWritten, tc.frameLevel)
			}
		})
	}
}

func TestSetEquippedFrameRequestValidation(t *testing.T) {
	for _, frame := range []int{0, -1, leveling.MaxLevel + 1} {
		req := SetEquippedFrameRequest{FrameLevel: frame}
		if err := req.Validate(); err == nil {
			t.Errorf("frame_level %d should fail validation", frame)
		}
	}
	for _, frame := range []int{1, 50, leveling.MaxLevel} {
		req := SetEquippedFrameRequest{FrameLevel: frame}
		if err := req.Validate(); err != nil {
			t.Errorf("frame_level %d should pass validation, got %v", frame, err)
		}
	}
}
