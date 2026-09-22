package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github/DoanCongPho/game-arena/internal/platform/leveling"

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
	// GrantIfFirstAttempt grants amount XP to userID and recomputes their
	// level, but only if this is the first time XP has ever been granted
	// for this (userID, testID) pair — enforced atomically via the
	// submission_xp_grants table's primary key, not by a check-then-write
	// race. Safe to call once per graded submission.
	GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (granted bool, level int, xp int, err error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) Repository {
	return &userRepository{db: db}
}

const userColumns = `id, name, email, password_hash, level, xp, rank_score, image_url, role, created_at, updated_at`

func scanUser(row interface{ Scan(dest ...any) error }) (*User, error) {
	var user User
	var imageURL sql.NullString
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Level,
		&user.XP,
		&user.RankScore,
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

	res, err := u.db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash, level, xp, rank_score, image_url, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		user.Name, user.Email, user.PasswordHash, user.Level, user.XP, user.RankScore, user.ImageURL, user.Role, user.CreatedAt, user.UpdatedAt,
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
		"UPDATE users SET name = ?, email = ?, password_hash = ?, level = ?, xp = ?, rank_score = ?, image_url = ?, role = ?, updated_at = ? WHERE id = ?",
		user.Name, user.Email, user.PasswordHash, user.Level, user.XP, user.RankScore, user.ImageURL, user.Role, user.UpdatedAt, user.ID,
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

func (u *userRepository) GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (bool, int, int, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("begin xp grant tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		"INSERT INTO submission_xp_grants (user_id, test_id, submission_id, xp_awarded, granted_at) VALUES (?, ?, ?, ?, ?)",
		userID, testID, submissionID, amount, time.Now(),
	)
	if isDuplicateKeyErr(err) {
		// XP for this (user, test) pair was already granted by an earlier
		// submission — return the user's current standing, no new grant.
		cur, ferr := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", userID))
		if ferr != nil {
			if errors.Is(ferr, sql.ErrNoRows) {
				return false, 0, 0, ErrUserNotFound
			}
			return false, 0, 0, fmt.Errorf("load user after duplicate xp grant: %w", ferr)
		}
		if cerr := tx.Commit(); cerr != nil {
			return false, 0, 0, fmt.Errorf("commit duplicate xp grant read: %w", cerr)
		}
		return false, cur.Level, cur.XP, nil
	}
	if err != nil {
		return false, 0, 0, fmt.Errorf("claim xp grant: %w", err)
	}

	var currentXP int
	if err := tx.QueryRowContext(ctx, "SELECT xp FROM users WHERE id = ? FOR UPDATE", userID).Scan(&currentXP); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, 0, ErrUserNotFound
		}
		return false, 0, 0, fmt.Errorf("lock user for xp grant: %w", err)
	}

	newXP := currentXP + amount
	newLevel, _, _ := leveling.LevelForXP(newXP)

	if _, err := tx.ExecContext(ctx,
		"UPDATE users SET xp = ?, level = ?, updated_at = ? WHERE id = ?",
		newXP, newLevel, time.Now(), userID,
	); err != nil {
		return false, 0, 0, fmt.Errorf("update xp/level: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, 0, 0, fmt.Errorf("commit xp grant: %w", err)
	}
	return true, newLevel, newXP, nil
}

func isDuplicateKeyErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
