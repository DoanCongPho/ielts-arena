package ielts_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const testColumns = `id, skill, task_type, content_data, source, is_current, xp_gain, created_at`
const submissionColumns = `id, user_id, test_id, payload, status, submitted_at`
const scoreColumns = `id, submission_id, overall_band, details, graded_at`

type Repository interface {
	GetCurrentTest(ctx context.Context, skill string) (*Test, error)
	GetTestByID(ctx context.Context, id uint64) (*Test, error)

	CreateSubmission(ctx context.Context, s *Submission) (*Submission, error)
	GetSubmissionByID(ctx context.Context, id uint64) (*Submission, error)

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
	err := row.Scan(
		&test.ID,
		&test.Skill,
		&test.TaskType,
		&test.ContentData,
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
	return &test, nil
}

func (r *repository) GetCurrentTest(ctx context.Context, skill string) (*Test, error) {
	query := "SELECT " + testColumns + " FROM tests WHERE skill = ? AND is_current = TRUE LIMIT 1"
	return scanTest(r.db.QueryRowContext(ctx, query, skill))
}

func (r *repository) GetTestByID(ctx context.Context, id uint64) (*Test, error) {
	query := "SELECT " + testColumns + " FROM tests WHERE id = ?"
	return scanTest(r.db.QueryRowContext(ctx, query, id))
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

func (r *repository) GetSubmissionByID(ctx context.Context, id uint64) (*Submission, error) {
	var s Submission
	query := "SELECT " + submissionColumns + " FROM submissions WHERE id = ?"
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.TestID,
		&s.Payload,
		&s.Status,
		&s.SubmittedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find submission by id: %w", err)
	}
	return &s, nil
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
