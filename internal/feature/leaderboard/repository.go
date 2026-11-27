package leaderboard

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository interface {
	// All returns every learner with some XP or at least one graded
	// submission.
	All(ctx context.Context) ([]Stats, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Only graded submissions count as a finished test: pending ones may still
// fail, and failed ones never got a result. Per-skill counts come from the
// test each submission answered.
const allQuery = `
SELECT u.id, u.name, COALESCE(u.image_url, ''), u.equipped_avatar, u.equipped_frame, u.xp,
       COALESCE(SUM(t.skill = 'reading'), 0),
       COALESCE(SUM(t.skill = 'listening'), 0),
       COALESCE(SUM(t.skill = 'writing'), 0),
       COALESCE(SUM(t.skill = 'speaking'), 0)
FROM users u
LEFT JOIN submissions s ON s.user_id = u.id AND s.status = 'graded'
LEFT JOIN tests t ON t.id = s.test_id
GROUP BY u.id, u.name, u.image_url, u.equipped_avatar, u.equipped_frame, u.xp
HAVING u.xp > 0 OR COUNT(s.id) > 0`

func (r *repository) All(ctx context.Context) ([]Stats, error) {
	rows, err := r.db.QueryContext(ctx, allQuery)
	if err != nil {
		return nil, fmt.Errorf("leaderboard query: %w", err)
	}
	defer rows.Close()

	var out []Stats
	for rows.Next() {
		var s Stats
		var avatar, frame sql.NullString
		if err := rows.Scan(&s.UserID, &s.Name, &s.ImageURL, &avatar, &frame, &s.XP,
			&s.Reading, &s.Listening, &s.Writing, &s.Speaking); err != nil {
			return nil, fmt.Errorf("scan leaderboard row: %w", err)
		}
		if avatar.Valid {
			s.EquippedAvatar = &avatar.String
		}
		if frame.Valid {
			s.EquippedFrame = &frame.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
