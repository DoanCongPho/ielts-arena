package leveling

import "testing"

func TestLevelForXP(t *testing.T) {
	tests := []struct {
		name          string
		xp            int
		wantLevel     int
		wantCurrentXP int
		wantToNextLvl int
	}{
		{"zero xp starts at level 1", 0, 1, 0, 60},
		{"negative xp is clamped to zero", -500, 1, 0, 60},
		{"one xp short of level 2", 59, 1, 59, 1},
		{"exactly the level 2 threshold", 60, 2, 0, 62},
		{"partway through level 2", 100, 2, 40, 22},
		{"exactly the level 3 threshold", 122, 3, 0, 65},
		{"max level has nothing left to earn", 71344, MaxLevel, 0, 0},
		{"xp beyond max level stays at max", 999999, MaxLevel, 999999 - 71344, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			level, currentXP, toNext := LevelForXP(tc.xp)
			if level != tc.wantLevel {
				t.Errorf("level = %d, want %d", level, tc.wantLevel)
			}
			if currentXP != tc.wantCurrentXP {
				t.Errorf("currentLevelXP = %d, want %d", currentXP, tc.wantCurrentXP)
			}
			if toNext != tc.wantToNextLvl {
				t.Errorf("xpToNextLevel = %d, want %d", toNext, tc.wantToNextLvl)
			}
		})
	}
}

// The progression only makes sense if each level costs strictly more than
// the last. A regression here (a flat or inverted curve) would silently
// let a user's level go down as they earn XP.
func TestLevelThresholdsAreStrictlyIncreasing(t *testing.T) {
	for lvl := 2; lvl <= MaxLevel; lvl++ {
		prev, cur := XPRequiredForLevel(lvl-1), XPRequiredForLevel(lvl)
		if cur <= prev {
			t.Fatalf("threshold for level %d (%d) is not above level %d (%d)", lvl, cur, lvl-1, prev)
		}
	}
}

// LevelForXP and XPRequiredForLevel are two views of one table; they must
// agree at every boundary, including one XP either side of it.
func TestLevelForXPAgreesWithThresholdsAtEveryBoundary(t *testing.T) {
	for lvl := 2; lvl <= MaxLevel; lvl++ {
		threshold := XPRequiredForLevel(lvl)

		if got, _, _ := LevelForXP(threshold); got != lvl {
			t.Errorf("LevelForXP(%d) = %d, want %d (exactly at the threshold)", threshold, got, lvl)
		}
		if got, _, _ := LevelForXP(threshold - 1); got != lvl-1 {
			t.Errorf("LevelForXP(%d) = %d, want %d (one xp below the threshold)", threshold-1, got, lvl-1)
		}
	}
}

func TestXPRequiredForLevelClampsOutOfRange(t *testing.T) {
	if got := XPRequiredForLevel(1); got != 0 {
		t.Errorf("level 1 should require 0 xp, got %d", got)
	}
	if got := XPRequiredForLevel(0); got != XPRequiredForLevel(1) {
		t.Errorf("level 0 should clamp to level 1, got %d", got)
	}
	if got := XPRequiredForLevel(-10); got != XPRequiredForLevel(1) {
		t.Errorf("negative level should clamp to level 1, got %d", got)
	}
	if got := XPRequiredForLevel(MaxLevel + 50); got != XPRequiredForLevel(MaxLevel) {
		t.Errorf("level above max should clamp to max, got %d", got)
	}
}
