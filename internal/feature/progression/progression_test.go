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
}

func TestServiceNotFound(t *testing.T) {
	svc := NewService(NewMockRepository())
	ctx := context.Background()

	if _, err := svc.Get(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get: expected ErrNotFound, got %v", err)
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
