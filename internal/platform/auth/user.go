package auth

import "time"

type User struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Level        int       `json:"level"`
	XP           int       `json:"xp"`
	RankScore    int       `json:"rank_score"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
