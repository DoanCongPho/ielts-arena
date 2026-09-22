package auth

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User is identity only: who someone is and what they may do. Their game
// progress — XP, level, rank, avatar frames — lives in
// internal/feature/progression, which owns those columns of the same row.
//
// Splitting them is what lets this package stop importing the levelling
// curve, and stops every feature that needs to grant XP reaching through
// the identity repository to do it.
type User struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	ImageURL     string    `json:"image_url"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsAdmin reports whether the user has the admin role — used to gate
// admin-only actions like creating tests.
func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}
