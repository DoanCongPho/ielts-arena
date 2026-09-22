package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailTaken is the unique-index violation on users.email, turned
	// into a domain error at the repository boundary. Without this the
	// driver's raw "Error 1062 ... Duplicate entry 'x' for key
	// 'idx_users_email'" went straight into the registration response.
	ErrEmailTaken = errors.New("email already registered")
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uint64) (*User, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	UpdateUser(ctx context.Context, user *User) (*User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) Repository {
	return &userRepository{db: db}
}

// Identity columns only. xp, level and rank_score
// are the progression feature's to read and write.
const userColumns = `id, name, email, password_hash, image_url, role, created_at, updated_at`

func scanUser(row interface{ Scan(dest ...any) error }) (*User, error) {
	var user User
	var imageURL sql.NullString
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&imageURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.ImageURL = imageURL.String
	return &user, nil
}

func (u *userRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	query := "SELECT " + userColumns + " FROM users WHERE email = ?"
	user, err := scanUser(u.db.QueryRowContext(ctx, query, email))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (u *userRepository) FindByID(ctx context.Context, id uint64) (*User, error) {
	query := "SELECT " + userColumns + " FROM users WHERE id = ?"
	user, err := scanUser(u.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

func (u *userRepository) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, errors.New("user cannot be nil")
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	if user.Role == "" {
		user.Role = RoleUser
	}

	// xp, level and rank_score are left to their column defaults — this
	// package does not own them.
	res, err := u.db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash, image_url, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		user.Name, user.Email, user.PasswordHash, user.ImageURL, user.Role, user.CreatedAt, user.UpdatedAt,
	)
	if isDuplicateKeyErr(err) {
		return nil, ErrEmailTaken
	}
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
		"UPDATE users SET name = ?, email = ?, password_hash = ?, image_url = ?, role = ?, updated_at = ? WHERE id = ?",
		user.Name, user.Email, user.PasswordHash, user.ImageURL, user.Role, user.UpdatedAt, user.ID,
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

func isDuplicateKeyErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
