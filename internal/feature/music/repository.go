package music

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository interface {
	List(ctx context.Context, userID uint64) ([]Track, error)
	Count(ctx context.Context, userID uint64) (int, error)
	Create(ctx context.Context, t *Track) error
	Get(ctx context.Context, userID, id uint64) (*Track, error)
	Delete(ctx context.Context, userID, id uint64) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

const trackColumns = "id, user_id, title, object_key, content_type, size_bytes, created_at"

func scanTrack(row interface{ Scan(dest ...any) error }) (*Track, error) {
	var t Track
	if err := row.Scan(&t.ID, &t.UserID, &t.Title, &t.ObjectKey, &t.ContentType, &t.SizeBytes, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) List(ctx context.Context, userID uint64) ([]Track, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+trackColumns+" FROM user_tracks WHERE user_id = ? ORDER BY created_at, id", userID)
	if err != nil {
		return nil, fmt.Errorf("list tracks: %w", err)
	}
	defer rows.Close()
	out := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *repository) Count(ctx context.Context, userID uint64) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_tracks WHERE user_id = ?", userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count tracks: %w", err)
	}
	return n, nil
}

func (r *repository) Create(ctx context.Context, t *Track) error {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO user_tracks (user_id, title, object_key, content_type, size_bytes, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		t.UserID, t.Title, t.ObjectKey, t.ContentType, t.SizeBytes, t.CreatedAt)
	if err != nil {
		return fmt.Errorf("create track: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("create track id: %w", err)
	}
	t.ID = uint64(id)
	return nil
}

// Get returns the track only if userID owns it, so one learner can't
// reach another's by guessing ids.
func (r *repository) Get(ctx context.Context, userID, id uint64) (*Track, error) {
	t, err := scanTrack(r.db.QueryRowContext(ctx,
		"SELECT "+trackColumns+" FROM user_tracks WHERE id = ? AND user_id = ?", id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get track: %w", err)
	}
	return t, nil
}

func (r *repository) Delete(ctx context.Context, userID, id uint64) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM user_tracks WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("delete track: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
