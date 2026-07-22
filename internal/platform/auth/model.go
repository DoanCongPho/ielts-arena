package auth

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID                 uint64    `json:"id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	PasswordHash       string    `json:"-"`
	Level              int       `json:"level"`
	XP                 int       `json:"xp"`
	RankScore          int       `json:"rank_score"`
	ImageURL           string    `json:"image_url"`
	Role               string    `json:"role"`
	EquippedFrameLevel *int      `json:"equipped_frame_level,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// IsAdmin reports whether the user has the admin role — used to gate
// admin-only actions like creating tests.
func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}
