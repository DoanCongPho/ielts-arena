package profile

import (
	"github/DoanCongPho/game-arena/internal/feature/progression"
	"github/DoanCongPho/game-arena/internal/platform/auth"
)

type ProfileResponse struct {
	ID                    uint64 `json:"id"`
	Name                  string `json:"name"`
	Level                 int    `json:"level"`
	XP                    int    `json:"xp"`
	CurrentLevelXP        int    `json:"current_level_xp"`
	XPToNextLevel         int    `json:"xp_to_next_level"`
	ImageURL              string `json:"image_url"`
	EquippedFrameLevel    int    `json:"equipped_frame_level"`
	UnlockedMaxFrameLevel int    `json:"unlocked_max_frame_level"`
}

// newProfileResponse joins the two halves of a user. Every progression
// number comes from the Progression value, whose Level is derived from
// lifetime XP rather than read from the denormalised users.level column.
func newProfileResponse(u *auth.User, p *progression.Progression) *ProfileResponse {
	return &ProfileResponse{
		ID:                    u.ID,
		Name:                  u.Name,
		ImageURL:              u.ImageURL,
		Level:                 p.Level,
		XP:                    p.XP,
		CurrentLevelXP:        p.CurrentLevelXP(),
		XPToNextLevel:         p.XPToNextLevel(),
		EquippedFrameLevel:    p.EquippedOrDefaultFrame(),
		UnlockedMaxFrameLevel: p.UnlockedMaxFrameLevel(),
	}
}
