package shop

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Repository interface {
	Wallet(ctx context.Context, userID uint64) (*Wallet, error)
	Owned(ctx context.Context, userID uint64) ([]string, error)
	// Purchase records that userID bought item and takes its price out of
	// their balance, atomically: ErrAlreadyOwned if they have it,
	// ErrInsufficientXP if they can't afford it — in both cases nothing
	// changes.
	Purchase(ctx context.Context, userID uint64, item Item, now time.Time) (*Wallet, error)
	// Equip puts itemID (nil to clear) in the user's `kind` slot. Whether
	// they own it is the service's check.
	Equip(ctx context.Context, userID uint64, kind Kind, itemID *string) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

const walletColumns = "xp, xp_spent, equipped_avatar, equipped_frame"

func scanWallet(row interface{ Scan(dest ...any) error }) (*Wallet, error) {
	var w Wallet
	var avatar, frame sql.NullString
	if err := row.Scan(&w.XP, &w.Spent, &avatar, &frame); err != nil {
		return nil, err
	}
	if avatar.Valid {
		w.EquippedAvatar = &avatar.String
	}
	if frame.Valid {
		w.EquippedFrame = &frame.String
	}
	return &w, nil
}

func (r *repository) Wallet(ctx context.Context, userID uint64) (*Wallet, error) {
	w, err := scanWallet(r.db.QueryRowContext(ctx, "SELECT "+walletColumns+" FROM users WHERE id = ?", userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return w, nil
}

func (r *repository) Owned(ctx context.Context, userID uint64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT item_id FROM user_items WHERE user_id = ? ORDER BY purchased_at", userID)
	if err != nil {
		return nil, fmt.Errorf("list owned items: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan owned item: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *repository) Purchase(ctx context.Context, userID uint64, item Item, now time.Time) (*Wallet, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin purchase tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock the user's row first, so two purchases at once can't both see
	// the same balance and overspend it.
	w, err := scanWallet(tx.QueryRowContext(ctx, "SELECT "+walletColumns+" FROM users WHERE id = ? FOR UPDATE", userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock wallet: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO user_items (user_id, item_id, price, purchased_at) VALUES (?, ?, ?, ?)",
		userID, item.ID, item.Price, now,
	)
	if isDuplicateKeyErr(err) {
		return nil, ErrAlreadyOwned
	}
	if err != nil {
		return nil, fmt.Errorf("record purchase: %w", err)
	}

	if w.Balance() < item.Price {
		return nil, ErrInsufficientXP
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE users SET xp_spent = xp_spent + ?, updated_at = ? WHERE id = ?",
		item.Price, now, userID,
	); err != nil {
		return nil, fmt.Errorf("spend xp: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit purchase: %w", err)
	}
	w.Spent += item.Price
	return w, nil
}

func (r *repository) Equip(ctx context.Context, userID uint64, kind Kind, itemID *string) error {
	var column string
	switch kind {
	case KindAvatar:
		column = "equipped_avatar"
	case KindFrame:
		column = "equipped_frame"
	default:
		return ErrWrongSlot
	}
	res, err := r.db.ExecContext(ctx, "UPDATE users SET "+column+" = ? WHERE id = ?", itemID, userID)
	if err != nil {
		return fmt.Errorf("equip %s: %w", kind, err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		// MySQL reports 0 for an unchanged row too, so only a missing
		// user is an error.
		if _, werr := r.Wallet(ctx, userID); errors.Is(werr, ErrNotFound) {
			return ErrNotFound
		}
	}
	return nil
}

func isDuplicateKeyErr(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
