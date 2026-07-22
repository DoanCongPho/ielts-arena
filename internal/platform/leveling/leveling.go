// Package leveling defines the level 1-100 progression curve shared by any
// feature that needs to turn a lifetime XP total into a level (profile
// display, XP-grant level-up detection, etc.).
package leveling

import "math"

const MaxLevel = 100

// levelThresholds[L] is the cumulative lifetime XP required to be AT level L.
// levelThresholds[1] == 0.
var levelThresholds [MaxLevel + 1]int

func init() {
	const baseXP = 60       // XP cost of level 1 -> 2
	const growthRate = 1.04 // each next level-up costs 4% more than the previous (compound growth)

	cost := float64(baseXP)
	cumulative := 0.0
	for lvl := 1; lvl < MaxLevel; lvl++ {
		cumulative += cost
		levelThresholds[lvl+1] = int(math.Round(cumulative))
		cost *= growthRate
	}
}

// XPRequiredForLevel returns the cumulative lifetime XP needed to be at
// level (clamped to [1, MaxLevel]).
func XPRequiredForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	if level > MaxLevel {
		level = MaxLevel
	}
	return levelThresholds[level]
}

// LevelForXP derives the level (1..MaxLevel) from a lifetime XP total, plus
// how much XP has been earned past that level's threshold and how much more
// is needed to reach the next level (0 once MaxLevel is reached).
func LevelForXP(xp int) (level int, currentLevelXP int, xpToNextLevel int) {
	if xp < 0 {
		xp = 0
	}

	level = 1
	for lvl := MaxLevel; lvl >= 1; lvl-- {
		if xp >= levelThresholds[lvl] {
			level = lvl
			break
		}
	}

	currentLevelXP = xp - levelThresholds[level]
	if level >= MaxLevel {
		return level, currentLevelXP, 0
	}
	xpToNextLevel = levelThresholds[level+1] - xp
	return level, currentLevelXP, xpToNextLevel
}
