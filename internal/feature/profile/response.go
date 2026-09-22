package profile

import (
	"github/DoanCongPho/game-arena/internal/platform/auth"
	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

type ProfileResponse struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	Level          int    `json:"level"`
	XP             int    `json:"xp"`
	CurrentLevelXP int    `json:"current_level_xp"`
	XPToNextLevel  int    `json:"xp_to_next_level"`
	ImageURL       string `json:"image_url"`
}

// newProfileResponse recomputes level from lifetime XP rather than trusting
// the denormalized users.level cache, so a stale cache can never make it
// into what the user sees.
func newProfileResponse(u *auth.User) *ProfileResponse {
	level, currentLevelXP, xpToNext := leveling.LevelForXP(u.XP)

	return &ProfileResponse{
		ID:             u.ID,
		Name:           u.Name,
		Level:          level,
		XP:             u.XP,
		CurrentLevelXP: currentLevelXP,
		XPToNextLevel:  xpToNext,
		ImageURL:       u.ImageURL,
	}
}
