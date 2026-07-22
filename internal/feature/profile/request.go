package profile

import (
	"errors"

	"github/DoanCongPho/game-arena/internal/platform/leveling"
)

type SetEquippedFrameRequest struct {
	FrameLevel int `json:"frame_level"`
}

func (r *SetEquippedFrameRequest) Validate() error {
	if r.FrameLevel < 1 || r.FrameLevel > leveling.MaxLevel {
		return errors.New("frame_level must be between 1 and 100")
	}
	return nil
}
