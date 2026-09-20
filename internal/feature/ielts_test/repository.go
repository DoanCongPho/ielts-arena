package ielts_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const testColumns = `id, skill, task_type, content_data, thumbnail_url, source, is_current, xp_gain, created_at`
const submissionColumns = `id, user_id, test_id, payload, status, submitted_at, attempts, last_error, next_attempt_at, claimed_at`
const scoreColumns = `id, submission_id, overall_band, details, graded_at`

type Repository interface {
	// ---test---
	CreateTest(ctx context.Context, t *Test) (*Test, error)
	GetTestByID(ctx context.Context, id uint64) (*Test, error)
	GetListTest(ctx context.Context, skill string, limit, offset int) ([]Test, int, error)
	// --- submission ---
	CreateSubmission(ctx context.Context, s *Submission) (*Submission, error)
	GetSubmissionByID(ctx context.Context, id uint64) (*Submission, error)
	GetListSubmission(ctx context.Context, userID uint64, limit, offset int) ([]SubmissionSummary, int, error)
	UpdateSubmissionStatus(ctx context.Context, id uint64, status string) error
	// --- grading queue ---
	// ClaimNextForGrading atomically hands one submission to the caller and
	// marks it "grading". It returns (nil, nil) when the queue is empty.
	// Rows still marked "grading" whose claim predates leaseCutoff are
	// treated as abandoned (worker crashed) and reclaimed.
	ClaimNextForGrading(ctx context.Context, now, leaseCutoff time.Time) (*Submission, error)
	// RescheduleGrading returns a submission to the queue after a transient
	// failure, recording the cause and when it may be retried.
	RescheduleGrading(ctx context.Context, id uint64, cause string, nextAttemptAt time.Time) error
	// FailGrading settles a submission as permanently failed.
	FailGrading(ctx context.Context, id uint64, cause string) error
	// ---score---
	CreateScore(ctx context.Context, sc *Score) (*Score, error)
	GetScoreBySubmissionID(ctx context.Context, submissionID uint64) (*Score, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func scanTest(row *sql.Row) (*Test, error) {
	var test Test
	var thumbnailURL sql.NullString
	err := row.Scan(
		&test.ID,
		&test.Skill,
		&test.TaskType,
		&test.ContentData,
		&thumbnailURL,
		&test.Source,
		&test.IsCurrent,
		&test.XPGain,
		&test.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan test: %w", err)
	}
	test.ThumbnailURL = thumbnailURL.String
	return &test, nil
}

func (r *repository) CreateTest(ctx context.Context, t *Test) (*Test, error) {
	if t == nil {
		return nil, errors.New("test cannot be nil")
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}

	res, err := r.db.ExecContext(ctx,
		"INSERT INTO tests (skill, task_type, content_data, thumbnail_url, source, is_current, xp_gain, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		t.Skill, t.TaskType, t.ContentData, t.ThumbnailURL, t.Source, t.IsCurrent, t.XPGain, t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert test: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get insert id: %w", err)
	}
	t.ID = uint64(id)
	return t, nil
}

func (r *repository) GetTestByID(ctx context.Context, id uint64) (*Test, error) {
	query := "SELECT " + testColumns + " FROM tests WHERE id = ?"
	return scanTest(r.db.QueryRowContext(ctx, query, id))
}

func (r *repository) GetListTest(ctx context.Context, skill string, limit, offset int) ([]Test, int, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+testColumns+", COUNT(*) OVER() FROM tests WHERE skill = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		skill, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list tests: %w", err)
	}
	defer rows.Close()

	var (
		tests []Test
		total int
	)
	for rows.Next() {
		var t Test
		var thumbnailURL sql.NullString
		if err := rows.Scan(&t.ID, &t.Skill, &t.TaskType, &t.ContentData, &thumbnailURL, &t.Source, &t.IsCurrent, &t.XPGain, &t.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan test: %w", err)
		}
		t.ThumbnailURL = thumbnailURL.String
		tests = append(tests, t)
	}
	return tests, total, rows.Err()
}

func (r *repository) GetListSubmission(ctx context.Context, userID uint64, limit, offset int) ([]SubmissionSummary, int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.id, s.user_id, s.test_id, s.payload, s.status, s.submitted_at,
		        t.skill, t.task_type, t.xp_gain,
		        sc.overall_band, sc.graded_at,
		        COUNT(*) OVER()
		 FROM submissions s
		 JOIN tests t ON t.id = s.test_id
		 LEFT JOIN scores sc ON sc.submission_id = s.id
		 WHERE s.user_id = ?
		 ORDER BY s.submitted_at DESC
		 LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list submissions: %w", err)
	}
	defer rows.Close()

	var (
		summaries []SubmissionSummary
		total     int
	)
	for rows.Next() {
		var sm SubmissionSummary
		var overallBand sql.NullFloat64
		var gradedAt sql.NullTime
		if err := rows.Scan(
			&sm.ID, &sm.UserID, &sm.TestID, &sm.Payload, &sm.Status, &sm.SubmittedAt,
			&sm.TestSkill, &sm.TestTaskType, &sm.TestXPGain,
			&overallBand, &gradedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan submission summary: %w", err)
		}
		if overallBand.Valid {
			sm.OverallBand = &overallBand.Float64
		}
		if gradedAt.Valid {
			sm.GradedAt = &gradedAt.Time
		}
		summaries = append(summaries, sm)
	}
	return summaries, total, rows.Err()
}

func (r *repository) CreateSubmission(ctx context.Context, s *Submission) (*Submission, error) {
	if s == nil {
		return nil, errors.New("submission cannot be nil")
	}
	if s.SubmittedAt.IsZero() {
		s.SubmittedAt = time.Now()
	}

	res, err := r.db.ExecContext(ctx,
		"INSERT INTO submissions (user_id, test_id, payload, status, submitted_at) VALUES (?, ?, ?, ?, ?)",
		s.UserID, s.TestID, s.Payload, s.Status, s.SubmittedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert submission: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get insert id: %w", err)
	}
	s.ID = uint64(id)
	return s, nil
}

// scanSubmission reads the submissionColumns projection, including the
// nullable grading-queue columns.
func scanSubmission(row interface{ Scan(dest ...any) error }) (*Submission, error) {
	var s Submission
	var lastError sql.NullString
	var nextAttemptAt, claimedAt sql.NullTime
	if err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.TestID,
		&s.Payload,
		&s.Status,
		&s.SubmittedAt,
		&s.Attempts,
		&lastError,
		&nextAttemptAt,
		&claimedAt,
	); err != nil {
		return nil, err
	}
	s.LastError = lastError.String
	if nextAttemptAt.Valid {
		s.NextAttemptAt = &nextAttemptAt.Time
	}
	if claimedAt.Valid {
		s.ClaimedAt = &claimedAt.Time
	}
	return &s, nil
}

func (r *repository) GetSubmissionByID(ctx context.Context, id uint64) (*Submission, error) {
	query := "SELECT " + submissionColumns + " FROM submissions WHERE id = ?"
	s, err := scanSubmission(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find submission by id: %w", err)
	}
	return s, nil
}

// ClaimNextForGrading uses SELECT ... FOR UPDATE SKIP LOCKED so several
// workers can poll the same queue concurrently without ever handing the
// same submission to two of them, and without one worker's lock stalling
// the others.
func (r *repository) ClaimNextForGrading(ctx context.Context, now, leaseCutoff time.Time) (*Submission, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sub, err := scanSubmission(tx.QueryRowContext(ctx,
		"SELECT "+submissionColumns+` FROM submissions
		 WHERE (status = ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?))
		    OR (status = ? AND claimed_at IS NOT NULL AND claimed_at < ?)
		 ORDER BY submitted_at
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		StatusPending, now, StatusGrading, leaseCutoff,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // queue empty — not an error
	}
	if err != nil {
		return nil, fmt.Errorf("claim submission: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE submissions SET status = ?, attempts = attempts + 1, claimed_at = ? WHERE id = ?",
		StatusGrading, now, sub.ID,
	); err != nil {
		return nil, fmt.Errorf("mark submission claimed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}

	// Mirror what the UPDATE just wrote, so the caller sees the attempt
	// count this claim represents.
	sub.Status = StatusGrading
	sub.Attempts++
	sub.ClaimedAt = &now
	return sub, nil
}

func (r *repository) RescheduleGrading(ctx context.Context, id uint64, cause string, nextAttemptAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE submissions SET status = ?, last_error = ?, next_attempt_at = ?, claimed_at = NULL WHERE id = ?",
		StatusPending, truncateCause(cause), nextAttemptAt, id,
	)
	if err != nil {
		return fmt.Errorf("reschedule grading: %w", err)
	}
	return nil
}

func (r *repository) FailGrading(ctx context.Context, id uint64, cause string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE submissions SET status = ?, last_error = ?, next_attempt_at = NULL, claimed_at = NULL WHERE id = ?",
		StatusFailed, truncateCause(cause), id,
	)
	if err != nil {
		return fmt.Errorf("fail grading: %w", err)
	}
	return nil
}

// truncateCause keeps an error message inside last_error's VARCHAR(500)
// so a long driver or API error can never make the bookkeeping write fail.
func truncateCause(cause string) string {
	const maxLen = 500
	if len(cause) <= maxLen {
		return cause
	}
	return cause[:maxLen-3] + "..."
}

func (r *repository) UpdateSubmissionStatus(ctx context.Context, id uint64, status string) error {
	res, err := r.db.ExecContext(ctx, "UPDATE submissions SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}
	if n == 0 {
		return ErrSubmissionNotFound
	}
	return nil
}

func (r *repository) CreateScore(ctx context.Context, sc *Score) (*Score, error) {
	if sc == nil {
		return nil, errors.New("score cannot be nil")
	}
	if sc.GradedAt == nil {
		now := time.Now()
		sc.GradedAt = &now
	}

	res, err := r.db.ExecContext(ctx,
		"INSERT INTO scores (submission_id, overall_band, details, graded_at) VALUES (?, ?, ?, ?)",
		sc.SubmissionID, sc.OverallBand, sc.Details, sc.GradedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert score: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get insert id: %w", err)
	}
	sc.ID = uint64(id)
	return sc, nil
}

func (r *repository) GetScoreBySubmissionID(ctx context.Context, submissionID uint64) (*Score, error) {
	var sc Score
	var overallBand sql.NullFloat64
	var gradedAt sql.NullTime

	query := "SELECT " + scoreColumns + " FROM scores WHERE submission_id = ?"
	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(
		&sc.ID,
		&sc.SubmissionID,
		&overallBand,
		&sc.Details,
		&gradedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrScoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find score by submission id: %w", err)
	}

	if overallBand.Valid {
		sc.OverallBand = &overallBand.Float64
	}
	if gradedAt.Valid {
		sc.GradedAt = &gradedAt.Time
	}
	return &sc, nil
}
