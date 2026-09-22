package progression

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// progressionColumns is deliberately narrow: this repository reads and
// writes only the game-progress columns of the users row. Identity
// columns (name, email, password_hash, role) belong to platform/auth and
// are never touched here.
const progressionColumns = `xp, level, rank_score`

type Repository interface {
	Get(ctx context.Context, userID uint64) (*Progression, error)
	// GrantIfFirstAttempt grants amount XP to userID and recomputes their
	// level, but only if this is the first time XP has ever been granted
	// for this (userID, testID) pair — enforced atomically via the
	// submission_xp_grants table's primary key, not by a check-then-write
	// race. Safe to call once per graded submission.
	GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (granted bool, level int, xp int, err error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func scanProgression(userID uint64, row interface{ Scan(dest ...any) error }) (*Progression, error) {
	var p Progression
	var cachedLevel int

	if err := row.Scan(&p.XP, &cachedLevel, &p.RankScore); err != nil {
		return nil, err
	}

	p.UserID = userID
	// cachedLevel is intentionally discarded: the level is derived from
	// lifetime XP so the denormalised column can never be the source of
	// truth for a rule or for what the user sees.
	p.Level, _, _ = levelForXP(p.XP)
	return &p, nil
}

func (r *repository) Get(ctx context.Context, userID uint64) (*Progression, error) {
	query := "SELECT " + progressionColumns + " FROM users WHERE id = ?"
	p, err := scanProgression(userID, r.db.QueryRowContext(ctx, query, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get progression: %w", err)
	}
	return p, nil
}

func (r *repository) GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (bool, int, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("begin xp grant tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		"INSERT INTO submission_xp_grants (user_id, test_id, submission_id, xp_awarded, granted_at) VALUES (?, ?, ?, ?, ?)",
		userID, testID, submissionID, amount, time.Now(),
	)
	if isDuplicateKeyErr(err) {
		// XP for this (user, test) pair was already granted by an earlier
		// submission — return the user's current standing, no new grant.
		cur, ferr := scanProgression(userID, tx.QueryRowContext(ctx,
			"SELECT "+progressionColumns+" FROM users WHERE id = ?", userID))
		if ferr != nil {
			if errors.Is(ferr, sql.ErrNoRows) {
				return false, 0, 0, ErrNotFound
			}
			return false, 0, 0, fmt.Errorf("load progression after duplicate xp grant: %w", ferr)
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
			return false, 0, 0, ErrNotFound
		}
		return false, 0, 0, fmt.Errorf("lock user for xp grant: %w", err)
	}

	newXP := currentXP + amount
	newLevel, _, _ := levelForXP(newXP)

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
