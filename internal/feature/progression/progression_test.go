package progression

import (
	"context"
	"errors"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

// The users.level column is a cache written by XP grants. Every rule and
// every number shown to the user must come from lifetime XP instead, so a
// wrong or stale cache cannot leak into gameplay.
func TestProgressionDerivesLevelFromXP(t *testing.T) {
	repo := NewMockRepository()
	p := repo.Seed(1, 200)

	wantLevel, wantCurrent, wantNext := leveling.LevelForXP(200)
	if p.Level != wantLevel {
		t.Errorf("Level = %d, want %d", p.Level, wantLevel)
	}
	if p.CurrentLevelXP() != wantCurrent {
		t.Errorf("CurrentLevelXP() = %d, want %d", p.CurrentLevelXP(), wantCurrent)
	}
	if p.XPToNextLevel() != wantNext {
		t.Errorf("XPToNextLevel() = %d, want %d", p.XPToNextLevel(), wantNext)
	}
	if p.UnlockedMaxFrameLevel() != wantLevel {
		t.Errorf("UnlockedMaxFrameLevel() = %d, want %d", p.UnlockedMaxFrameLevel(), wantLevel)
	}
}

func TestEquippedOrDefaultFrame(t *testing.T) {
	p := &Progression{Level: 7}
	if got := p.EquippedOrDefaultFrame(); got != 7 {
		t.Errorf("with nothing equipped, got frame %d, want the highest unlocked (7)", got)
	}

	chosen := 3
	p.EquippedFrameLevel = &chosen
	if got := p.EquippedOrDefaultFrame(); got != 3 {
		t.Errorf("with frame 3 equipped, got %d, want 3", got)
	}
}

func TestCanEquip(t *testing.T) {
	p := &Progression{Level: 5}
	tests := []struct {
		frame int
		want  bool
	}{
		{1, true}, {3, true}, {5, true},
		{6, false}, {100, false}, {0, false}, {-1, false},
	}
	for _, tc := range tests {
		if got := p.CanEquip(tc.frame); got != tc.want {
			t.Errorf("CanEquip(%d) = %v, want %v (user is level %d)", tc.frame, got, tc.want, p.Level)
		}
	}
}

func TestServiceSetEquippedFrame(t *testing.T) {
	tests := []struct {
		name    string
		frame   int
		wantErr error
	}{
		{"below current level", 3, nil},
		{"exactly current level", 5, nil},
		{"one above current level", 6, ErrFrameLocked},
		{"far above current level", 100, ErrFrameLocked},
		{"zero", 0, ErrFrameLocked},
		{"negative", -1, ErrFrameLocked},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMockRepository()
			p := repo.Seed(1, 0)
			p.Level = 5 // pretend this user has earned level 5
			svc := NewService(repo)

			got, err := svc.SetEquippedFrame(context.Background(), 1, tc.frame)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if p.EquippedFrameLevel != nil {
					t.Errorf("a locked frame must not be persisted, but %d was written", *p.EquippedFrameLevel)
				}
				return
			}

			if err != nil {
				t.Fatalf("SetEquippedFrame: %v", err)
			}
			if got.EquippedFrameLevel == nil || *got.EquippedFrameLevel != tc.frame {
				t.Errorf("returned frame = %v, want %d", got.EquippedFrameLevel, tc.frame)
			}
			if p.EquippedFrameLevel == nil || *p.EquippedFrameLevel != tc.frame {
				t.Errorf("persisted frame = %v, want %d", p.EquippedFrameLevel, tc.frame)
			}
		})
	}
}

func TestServiceNotFound(t *testing.T) {
	svc := NewService(NewMockRepository())
	ctx := context.Background()

	if _, err := svc.Get(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := svc.SetEquippedFrame(ctx, 99, 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetEquippedFrame: expected ErrNotFound, got %v", err)
	}
}

// XP is granted once per (user, test). A second graded submission for the
// same test reports the user's standing without awarding again — this is
// what stops a user farming the same test for unlimited XP.
func TestGrantIsIdempotentPerTest(t *testing.T) {
	repo := NewMockRepository()
	repo.Seed(1, 0)
	ctx := context.Background()

	granted, level, xp, err := repo.GrantIfFirstAttempt(ctx, 1, 10, 100, 60)
	if err != nil {
		t.Fatalf("first grant: %v", err)
	}
	if !granted {
		t.Error("the first attempt at a test should grant XP")
	}
	if xp != 60 {
		t.Errorf("xp = %d, want 60", xp)
	}
	if wantLevel, _, _ := leveling.LevelForXP(60); level != wantLevel {
		t.Errorf("level = %d, want %d", level, wantLevel)
	}

	granted, _, xp, err = repo.GrantIfFirstAttempt(ctx, 1, 10, 101, 60)
	if err != nil {
		t.Fatalf("second grant: %v", err)
	}
	if granted {
		t.Error("a repeat attempt at the same test must not grant XP again")
	}
	if xp != 60 {
		t.Errorf("xp = %d after a repeat attempt, want it unchanged at 60", xp)
	}

	// A different test is a separate grant.
	granted, _, xp, err = repo.GrantIfFirstAttempt(ctx, 1, 11, 102, 40)
	if err != nil {
		t.Fatalf("grant for another test: %v", err)
	}
	if !granted {
		t.Error("a different test should grant XP")
	}
	if xp != 100 {
		t.Errorf("xp = %d, want 100", xp)
	}
}
