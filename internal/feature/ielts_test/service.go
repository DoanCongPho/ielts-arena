package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github/DoanCongPho/game-arena/internal/platform/httpx"
)

// Service is the business contract for this feature. It deliberately
// mentions no router, no http.Request and no status codes — Handler in
// handler.go owns all of that. That separation is what lets this interface
// be exercised in tests with nothing but a context.
type Service interface {
	//test
	GetTest(ctx context.Context, id uint64) (*TestResponse, error)
	GetListTest(ctx context.Context, skill string, req ListTestRequest) (*ListTestResponse, error)
	PostTest(ctx context.Context, test Test) (*Test, error)
	//submit
	SubmitAnswer(ctx context.Context, userID uint64, req SubmitRequest) (*Submission, error)
	GetSubmissionByID(ctx context.Context, userID uint64, submissionID uint64) (*Submission, error)
	GetListSubmission(ctx context.Context, userID uint64, req ListSubmissionRequest) (*ListSubmissionResponse, error)
	//score
	GetScore(ctx context.Context, userID uint64, submissionID uint64) (*Score, error)
	//grading queue
	// GradeNextPending claims and grades at most one queued submission,
	// reporting whether there was anything to do. It is the single unit of
	// work a grading worker performs — exposing it here (rather than
	// burying a loop inside the worker) is what makes the whole async path
	// testable synchronously, with no goroutines or sleeps.
	GradeNextPending(ctx context.Context) (worked bool, err error)
}

// XPGrantRepository is the minimal user-progression dependency SubmitAnswer
// needs: granting XP for a graded submission, exactly once per (user, test)
// pair. Defined here, at the point of use, rather than imported from the
// auth package, so this package doesn't need to know about auth's other
// concerns — auth.Repository satisfies this interface structurally.
type XPGrantRepository interface {
	GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (granted bool, level int, xp int, err error)
}

// Grading-queue policy. These govern how a failed grade is retried, not
// how often the queue is polled (that is the worker's concern, see
// worker.go).
const (
	// maxGradingAttempts is how many times a transient failure is retried
	// before the submission settles as permanently failed.
	maxGradingAttempts = 3
	// gradingLease is how long a claimed submission may stay "grading"
	// before another worker assumes the claimer died and reclaims it.
	gradingLease = 5 * time.Minute
	// retryBaseDelay is the first retry delay; it doubles per attempt.
	retryBaseDelay = 30 * time.Second
)

// ErrUngradable marks a failure that retrying cannot fix — malformed
// content or payload, or a skill with no grader behind it. These settle as
// "failed" immediately instead of burning retry attempts and, for
// LLM-graded skills, OpenAI calls.
var ErrUngradable = errors.New("submission cannot be graded")

type service struct {
	repo     Repository
	grader   Grader
	xpGrants XPGrantRepository
}

func NewService(repo Repository, grader Grader, xpGrants XPGrantRepository) Service {
	return &service{repo: repo, grader: grader, xpGrants: xpGrants}
}

func (s *service) GetTest(ctx context.Context, id uint64) (*TestResponse, error) {
	t, err := s.repo.GetTestByID(ctx, id)
	if err != nil {
		return nil, err
	}
	content, err := publicContentData(t.Skill, t.ContentData)
	if err != nil {
		return nil, fmt.Errorf("prepare test content: %w", err)
	}
	resp := newTestResponse(t, content)
	return &resp, nil
}

func (s *service) GetListTest(ctx context.Context, skill string, req ListTestRequest) (*ListTestResponse, error) {
	tests, total, err := s.repo.GetListTest(ctx, skill, req.Limit(), req.Offset())
	if err != nil {
		return nil, err
	}
	resp := &ListTestResponse{Pagination: httpx.NewPagination(total, req.Page, req.Limit())}
	for _, t := range tests {
		content, err := publicContentData(t.Skill, t.ContentData)
		if err != nil {
			return nil, fmt.Errorf("prepare test content: %w", err)
		}
		resp.Data = append(resp.Data, newTestResponse(&t, content))
	}
	return resp, nil
}

func (s *service) PostTest(ctx context.Context, test Test) (*Test, error) {
	if err := validateContentData(test.Skill, test.ContentData); err != nil {
		return nil, err
	}
	return s.repo.CreateTest(ctx, &test)
}

// SubmitAnswer persists an answer and queues it for grading. It does not
// grade: grading calls a paid third-party API for writing/speaking and is
// far too slow to hold an HTTP request open for, so it happens in the
// background worker (worker.go) which also bounds how many grades run at
// once. The caller gets a "pending" submission back and polls
// GET /submissions/{id} until the status settles.
func (s *service) SubmitAnswer(ctx context.Context, userID uint64, req SubmitRequest) (*Submission, error) {
	// Validate the test exists before accepting the answer, so a bad
	// test_id is a 404 at submit time rather than a failed grade later.
	if _, err := s.repo.GetTestByID(ctx, req.TestID); err != nil {
		return nil, err
	}

	return s.repo.CreateSubmission(ctx, &Submission{
		UserID:  userID,
		TestID:  req.TestID,
		Payload: req.Payload,
		Status:  StatusPending,
	})
}

// GradeNextPending claims one queued submission and grades it. Returns
// worked=false when the queue is empty. A grading failure is reported as
// an error for logging, but is already recorded on the submission (either
// rescheduled or settled as failed) by the time it returns.
func (s *service) GradeNextPending(ctx context.Context) (bool, error) {
	now := time.Now()
	sub, err := s.repo.ClaimNextForGrading(ctx, now, now.Add(-gradingLease))
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, nil
	}
	return true, s.gradeClaimed(ctx, sub)
}

// gradeClaimed grades a submission this worker has already claimed, and is
// responsible for settling its status either way.
func (s *service) gradeClaimed(ctx context.Context, sub *Submission) error {
	test, err := s.repo.GetTestByID(ctx, sub.TestID)
	if err != nil {
		// The test is gone or unreadable — no retry will bring it back.
		return s.settleFailure(ctx, sub, fmt.Errorf("%w: load test: %v", ErrUngradable, err))
	}

	var gradeErr error
	switch test.Skill {
	case "writing", "speaking":
		gradeErr = s.gradeSubmission(ctx, test, sub)
	case "reading", "listening":
		gradeErr = s.autoGradeSubmission(ctx, test, sub)
	default:
		gradeErr = fmt.Errorf("%w: no grader for skill %q", ErrUngradable, test.Skill)
	}
	if gradeErr != nil {
		return s.settleFailure(ctx, sub, gradeErr)
	}

	// Server-authoritative XP grant: only reached once grading above has
	// set sub.Status = StatusGraded, and test.XPGain comes from the tests
	// row (set only via admin-only POST /api/tests) — never from anything
	// the submitting client sent. GrantIfFirstAttempt also caps this to the
	// user's first graded attempt at this test, so resubmitting the same
	// test for practice doesn't grant XP again.
	if sub.Status == StatusGraded && test.XPGain > 0 && s.xpGrants != nil {
		if _, _, _, err := s.xpGrants.GrantIfFirstAttempt(ctx, sub.UserID, test.ID, sub.ID, test.XPGain); err != nil {
			log.Printf("ielts_test: xp grant failed user=%d submission=%d: %v", sub.UserID, sub.ID, err)
			// Don't fail the job — the score is already persisted; XP is a
			// secondary side effect of grading, not the primary result.
		}
	}
	return nil
}

// settleFailure records a failed grading attempt: permanently for an
// ungradable submission or one out of attempts, otherwise back on the
// queue with exponential backoff.
//
// The bookkeeping write deliberately uses a context detached from
// cancellation — if the job was cancelled by shutdown or the job timeout,
// the write must still land, or the submission would sit in "grading"
// until its lease expires.
func (s *service) settleFailure(ctx context.Context, sub *Submission, cause error) error {
	writeCtx := context.WithoutCancel(ctx)

	permanent := errors.Is(cause, ErrUngradable) || sub.Attempts >= maxGradingAttempts
	if permanent {
		sub.Status = StatusFailed
		if err := s.repo.FailGrading(writeCtx, sub.ID, cause.Error()); err != nil {
			return fmt.Errorf("submission %d: settle failed: %w (original cause: %v)", sub.ID, err, cause)
		}
		return fmt.Errorf("submission %d failed permanently on attempt %d: %w", sub.ID, sub.Attempts, cause)
	}

	delay := retryBaseDelay << (sub.Attempts - 1)
	sub.Status = StatusPending
	if err := s.repo.RescheduleGrading(writeCtx, sub.ID, cause.Error(), time.Now().Add(delay)); err != nil {
		return fmt.Errorf("submission %d: reschedule failed: %w (original cause: %v)", sub.ID, err, cause)
	}
	return fmt.Errorf("submission %d attempt %d failed, retrying in %s: %w", sub.ID, sub.Attempts, delay, cause)
}

// gradeSubmission runs the LLM grader for a writing/speaking submission and
// persists the resulting score, updating the submission status in place.
func (s *service) gradeSubmission(ctx context.Context, test *Test, sub *Submission) error {
	var content WritingContent
	if err := json.Unmarshal(test.ContentData, &content); err != nil {
		return fmt.Errorf("%w: unmarshal test content: %v", ErrUngradable, err)
	}
	var payload WritingPayload
	if err := json.Unmarshal(sub.Payload, &payload); err != nil {
		return fmt.Errorf("%w: unmarshal submission payload: %v", ErrUngradable, err)
	}

	result, err := s.grader.Grade(ctx, test.Skill, test.TaskType, content.Prompt, content.ImageURL, payload.Text)
	if err != nil {
		return fmt.Errorf("grade: %w", err)
	}

	details, err := json.Marshal(ScoreDetails{
		Criteria:    result.Criteria,
		Corrections: result.Corrections,
		ModelAnswer: result.ModelAnswer,
	})
	if err != nil {
		return fmt.Errorf("marshal score details: %w", err)
	}

	overallBand := result.OverallBand
	if _, err := s.repo.CreateScore(ctx, &Score{
		SubmissionID: sub.ID,
		OverallBand:  &overallBand,
		Details:      details,
	}); err != nil {
		return fmt.Errorf("create score: %w", err)
	}

	sub.Status = StatusGraded
	return s.repo.UpdateSubmissionStatus(ctx, sub.ID, StatusGraded)
}

func (s *service) GetSubmissionByID(ctx context.Context, userID uint64, submissionID uint64) (*Submission, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub.UserID != userID {
		return nil, ErrSubmissionNotFound
	}
	return sub, nil
}

func (s *service) GetListSubmission(ctx context.Context, userID uint64, req ListSubmissionRequest) (*ListSubmissionResponse, error) {
	submissions, total, err := s.repo.GetListSubmission(ctx, userID, req.Limit(), req.Offset())
	if err != nil {
		return nil, err
	}
	resp := &ListSubmissionResponse{Pagination: httpx.NewPagination(total, req.Page, req.Limit())}
	for _, sub := range submissions {
		resp.Data = append(resp.Data, newSubmissionSummaryResponse(sub))
	}
	return resp, nil
}

func (s *service) GetScore(ctx context.Context, userID uint64, submissionID uint64) (*Score, error) {
	if _, err := s.GetSubmissionByID(ctx, userID, submissionID); err != nil {
		return nil, err
	}
	return s.repo.GetScoreBySubmissionID(ctx, submissionID)
}
