package profile

import "errors"

var ErrFrameLocked = errors.New("frame_level exceeds current level")
