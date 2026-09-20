package progression

import "errors"

var (
	// ErrFrameLocked means the requested avatar frame is above the
	// user's current level.
	ErrFrameLocked = errors.New("frame_level exceeds current level")
	// ErrNotFound means there is no user row to read progression from.
	ErrNotFound = errors.New("user progression not found")
)
