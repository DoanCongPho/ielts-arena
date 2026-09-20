// Package progression owns the game-progress half of a user: lifetime XP,
// the level derived from it, competitive rank, and which avatar frame is
// equipped.
//
// It exists as its own feature because none of that is infrastructure.
// These rules used to live in internal/platform/auth, which meant a
// platform package imported the levelling curve and every feature that
// needed to grant XP had to reach through the identity repository to get
// it. auth is now identity only; the two halves share the users table but
// never each other's columns.
package progression

import "github/DoanCongPho/game-arena/internal/platform/leveling"

// Progression is a user's game-progress state.
//
// Level is always derived from XP rather than read from the users.level
// column. That column is a denormalised cache written by XP grants;
// deriving instead of trusting it means a stale or wrong cache can never
// reach a gameplay rule or the user's screen.
type Progression struct {
	UserID             uint64
	XP                 int
	Level              int
	RankScore          int
	EquippedFrameLevel *int
}

// levelForXP is the package's single entry point to the levelling curve.
func levelForXP(xp int) (level, currentLevelXP, xpToNextLevel int) {
	return leveling.LevelForXP(xp)
}

// CurrentLevelXP is how much XP has been earned past the current level's
// threshold, and XPToNextLevel how much more is needed to advance.
func (p *Progression) CurrentLevelXP() int {
	_, current, _ := levelForXP(p.XP)
	return current
}

func (p *Progression) XPToNextLevel() int {
	_, _, toNext := levelForXP(p.XP)
	return toNext
}

// UnlockedMaxFrameLevel is the highest avatar frame this user has earned.
func (p *Progression) UnlockedMaxFrameLevel() int {
	return p.Level
}

// EquippedOrDefaultFrame is the frame to display: the one explicitly
// equipped, or the highest unlocked when the user has never chosen.
func (p *Progression) EquippedOrDefaultFrame() int {
	if p.EquippedFrameLevel != nil {
		return *p.EquippedFrameLevel
	}
	return p.Level
}

// CanEquip reports whether this user has unlocked frameLevel.
func (p *Progression) CanEquip(frameLevel int) bool {
	return frameLevel >= 1 && frameLevel <= p.Level
}
