package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	UpdateUser(ctx context.Context, user *User) (*User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) Repository {
	return &userRepository{db: db}
}

const userColumns = `id, name, email, password_hash, level, xp, rank_score, created_at, updated_at`

func (u *userRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	var user User
	query := "SELECT " + userColumns + " FROM users WHERE email = ?"
	err := u.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Level,
		&user.XP,
		&user.RankScore,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

func (u *userRepository) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, errors.New("user cannot be nil")
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	res, err := u.db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash, level, xp, rank_score, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		user.Name, user.Email, user.PasswordHash, user.Level, user.XP, user.RankScore, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get insert id: %w", err)
	}
	user.ID = uint64(id)
	return user, nil
}

func (u *userRepository) UpdateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, errors.New("user cannot be nil")
	}
	user.UpdatedAt = time.Now()

	res, err := u.db.ExecContext(ctx,
		"UPDATE users SET name = ?, email = ?, password_hash = ?, level = ?, xp = ?, rank_score = ?, updated_at = ? WHERE id = ?",
		user.Name, user.Email, user.PasswordHash, user.Level, user.XP, user.RankScore, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update user rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrUserNotFound
	}
	return user, nil
}
