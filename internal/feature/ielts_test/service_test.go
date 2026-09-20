package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeGrader is a controllable Grader stub so writing/speaking submissions
// can be tested without calling out to an LLM.
type fakeGrader struct {
	result *GradingResult
	err    error
}

func (f *fakeGrader) Grade(ctx context.Context, skill, taskType, prompt, imageURL, answer string) (*GradingResult, error) {
	return f.result, f.err
}

func newTestService(repo Repository, grader Grader) Service {
	return NewService(repo, grader, &fakeXPGrantRepository{})
}

// fakeXPGrantRepository is a no-op XPGrantRepository — none of these tests
// exercise XP granting (no seeded Test sets XPGain), it only needs to
// satisfy the dependency.
type fakeXPGrantRepository struct{}

func (f *fakeXPGrantRepository) GrantIfFirstAttempt(ctx context.Context, userID, testID, submissionID uint64, amount int) (bool, int, int, error) {
	return true, 1, amount, nil
}

// drainGradingQueue runs the worker's unit of work until the queue is
// empty. Grading is asynchronous in production, but GradeNextPending is a
// plain synchronous call, so a test can submit and then observe the
// settled result with no goroutines, no sleeps and no clock.
func drainGradingQueue(t *testing.T, svc Service) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		worked, err := svc.GradeNextPending(ctx)
		if err != nil {
			// Not fatal: a grading failure is recorded on the submission,
			// which is exactly what the failure tests then assert on.
			t.Logf("grading job reported: %v", err)
		}
		if !worked {
			return
		}
	}
	t.Fatal("grading queue did not drain after 50 jobs")
}

// submitAndGrade performs the full "user submits, worker picks it up"
// flow and returns the submission as it settled in the repository.
func submitAndGrade(t *testing.T, svc Service, repo *MockTestRepository, userID uint64, req SubmitRequest) *Submission {
	t.Helper()
	ctx := context.Background()

	sub, err := svc.SubmitAnswer(ctx, userID, req)
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}
	if sub.Status != StatusPending {
		t.Fatalf("SubmitAnswer must queue rather than grade: status = %q, want %q", sub.Status, StatusPending)
	}

	drainGradingQueue(t, svc)

	settled, err := repo.GetSubmissionByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	return settled
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// ---------------------------------------------------------------------------
// GetTest / GetListTest
// ---------------------------------------------------------------------------

func TestService_GetTest(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	created, err := repo.CreateTest(ctx, &Test{
		Skill:       "reading",
		TaskType:    "academic",
		ContentData: mustMarshal(t, validReadingContent()),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	resp, err := svc.GetTest(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTest: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("ID = %d, want %d", resp.ID, created.ID)
	}
	// The answer key must never reach the client.
	if strings.Contains(string(resp.ContentData), "cats") {
		t.Error("expected reading answers to be redacted from the response")
	}

	if _, err := svc.GetTest(ctx, 9999); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("expected ErrTestNotFound, got %v", err)
	}
}

func TestService_GetListTest_FiltersBySkillAndPaginates(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	writingContent := mustMarshal(t, WritingContent{Prompt: "Describe the chart."})
	for range 2 {
		if _, err := repo.CreateTest(ctx, &Test{Skill: "writing", TaskType: "task1", ContentData: writingContent}); err != nil {
			t.Fatalf("seed writing test: %v", err)
		}
	}
	if _, err := repo.CreateTest(ctx, &Test{Skill: "reading", TaskType: "academic", ContentData: mustMarshal(t, validReadingContent())}); err != nil {
		t.Fatalf("seed reading test: %v", err)
	}

	resp, err := svc.GetListTest(ctx, "writing", ListTestRequest{Page: 1})
	if err != nil {
		t.Fatalf("GetListTest: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("got %d writing tests, want 2", len(resp.Data))
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("Pagination.Total = %d, want 2", resp.Pagination.Total)
	}
	for _, d := range resp.Data {
		if d.Skill != "writing" {
			t.Errorf("expected only writing tests, got skill %q", d.Skill)
		}
	}
}

// ---------------------------------------------------------------------------
// PostTest
// ---------------------------------------------------------------------------

func TestService_PostTest_RejectsInvalidContent(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	_, err := svc.PostTest(ctx, Test{Skill: "writing", TaskType: "task2", ContentData: mustMarshal(t, WritingContent{})})
	if err == nil {
		t.Fatal("expected validation error for empty prompt")
	}

	if _, total, _ := repo.GetListTest(ctx, "", 10, 0); total != 0 {
		t.Errorf("invalid test should not have been persisted, total = %d", total)
	}
}

func TestService_PostTest_PersistsValidContent(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	created, err := svc.PostTest(ctx, Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("PostTest: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected an assigned ID")
	}

	if _, err := repo.GetTestByID(ctx, created.ID); err != nil {
		t.Errorf("expected persisted test to be retrievable: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SubmitAnswer — writing/speaking (LLM-graded)
// ---------------------------------------------------------------------------

func TestService_SubmitAnswer_WritingGradedSuccessfully(t *testing.T) {
	repo := NewMockTestRepository()
	grader := &fakeGrader{result: &GradingResult{
		OverallBand: 7,
		Criteria:    map[string]CriterionScore{"Task Achievement": {Score: 7, Feedback: "Good."}},
		ModelAnswer: "A model essay.",
	}}
	svc := newTestService(repo, grader)
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	sub := submitAndGrade(t, svc, repo, 1, SubmitRequest{
		TestID:  test.ID,
		Payload: mustMarshal(t, WritingPayload{Text: "My essay."}),
	})
	if sub.Status != StatusGraded {
		t.Errorf("Status = %q, want %q", sub.Status, StatusGraded)
	}

	score, err := repo.GetScoreBySubmissionID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("expected a score to be created: %v", err)
	}
	if score.OverallBand == nil || *score.OverallBand != 7 {
		t.Errorf("OverallBand = %v, want 7", score.OverallBand)
	}
}

func TestService_GradeNextPending_TransientGraderFailureIsRescheduled(t *testing.T) {
	repo := NewMockTestRepository()
	grader := &fakeGrader{err: errors.New("llm unavailable")}
	svc := newTestService(repo, grader)
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	sub := submitAndGrade(t, svc, repo, 1, SubmitRequest{
		TestID:  test.ID,
		Payload: mustMarshal(t, WritingPayload{Text: "My essay."}),
	})

	// An unreachable LLM is transient: the submission goes back on the
	// queue rather than being written off, which is the whole point of
	// moving grading out of the request.
	if sub.Status != StatusPending {
		t.Errorf("Status = %q, want %q (queued for retry)", sub.Status, StatusPending)
	}
	if sub.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", sub.Attempts)
	}
	if !strings.Contains(sub.LastError, "llm unavailable") {
		t.Errorf("LastError = %q, want it to mention the grader failure", sub.LastError)
	}
	if sub.NextAttemptAt == nil || !sub.NextAttemptAt.After(time.Now()) {
		t.Errorf("NextAttemptAt = %v, want a future retry time", sub.NextAttemptAt)
	}
	if _, err := repo.GetScoreBySubmissionID(ctx, sub.ID); !errors.Is(err, ErrScoreNotFound) {
		t.Errorf("expected no score to be created, got err = %v", err)
	}
}

func TestService_GradeNextPending_GiveUpAfterMaxAttempts(t *testing.T) {
	repo := NewMockTestRepository()
	grader := &fakeGrader{err: errors.New("llm unavailable")}
	svc := newTestService(repo, grader)
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	sub, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{
		TestID:  test.ID,
		Payload: mustMarshal(t, WritingPayload{Text: "My essay."}),
	})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}

	// Each pass is one attempt; clearing the backoff stands in for the
	// wait between them so the test doesn't depend on wall-clock time.
	for i := 0; i < maxGradingAttempts; i++ {
		drainGradingQueue(t, svc)
		current, err := repo.GetSubmissionByID(ctx, sub.ID)
		if err != nil {
			t.Fatalf("reload submission: %v", err)
		}
		if current.Status == StatusFailed {
			break
		}
		current.NextAttemptAt = nil
	}

	settled, err := repo.GetSubmissionByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if settled.Status != StatusFailed {
		t.Errorf("Status = %q, want %q after %d attempts", settled.Status, StatusFailed, maxGradingAttempts)
	}
	if settled.Attempts != maxGradingAttempts {
		t.Errorf("Attempts = %d, want %d", settled.Attempts, maxGradingAttempts)
	}
	if settled.NextAttemptAt != nil {
		t.Errorf("NextAttemptAt = %v, want nil once permanently failed", settled.NextAttemptAt)
	}
}

func TestService_GradeNextPending_UngradablePayloadFailsWithoutRetrying(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{result: &GradingResult{OverallBand: 7}})
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	// A payload that can never unmarshal into WritingPayload. Retrying
	// cannot fix it, so it must not consume attempts (or, for a real
	// grader, paid API calls).
	sub := submitAndGrade(t, svc, repo, 1, SubmitRequest{
		TestID:  test.ID,
		Payload: []byte(`"not an object"`),
	})

	if sub.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", sub.Status, StatusFailed)
	}
	if sub.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1 — an ungradable submission must not be retried", sub.Attempts)
	}
}

func TestService_GradeNextPending_EmptyQueueIsNotAnError(t *testing.T) {
	svc := newTestService(NewMockTestRepository(), &fakeGrader{})

	worked, err := svc.GradeNextPending(context.Background())
	if err != nil {
		t.Fatalf("GradeNextPending on an empty queue: %v", err)
	}
	if worked {
		t.Error("worked = true on an empty queue, want false")
	}
}

func TestService_GradeNextPending_ReclaimsSubmissionAbandonedByDeadWorker(t *testing.T) {
	repo := NewMockTestRepository()
	grader := &fakeGrader{result: &GradingResult{
		OverallBand: 7,
		Criteria:    map[string]CriterionScore{"Task Achievement": {Score: 7}},
	}}
	svc := newTestService(repo, grader)
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	// A worker claimed this submission and then died: it is stuck in
	// "grading" with a claim older than the lease.
	staleClaim := time.Now().Add(-2 * gradingLease)
	stuck, err := repo.CreateSubmission(ctx, &Submission{
		UserID:    1,
		TestID:    test.ID,
		Payload:   mustMarshal(t, WritingPayload{Text: "My essay."}),
		Status:    StatusGrading,
		Attempts:  1,
		ClaimedAt: &staleClaim,
	})
	if err != nil {
		t.Fatalf("seed stuck submission: %v", err)
	}

	drainGradingQueue(t, svc)

	settled, err := repo.GetSubmissionByID(ctx, stuck.ID)
	if err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if settled.Status != StatusGraded {
		t.Errorf("Status = %q, want %q — an expired lease must be reclaimed", settled.Status, StatusGraded)
	}
}

func TestService_SubmitAnswer_TestNotFound(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	_, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: 999, Payload: mustMarshal(t, WritingPayload{Text: "x"})})
	if !errors.Is(err, ErrTestNotFound) {
		t.Errorf("expected ErrTestNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// SubmitAnswer — reading/listening (auto-graded)
// ---------------------------------------------------------------------------

func TestService_SubmitAnswer_ReadingAutoGradesAllCorrect(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "reading",
		TaskType:    "academic",
		ContentData: mustMarshal(t, validReadingContent()),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	payload := AnswerPayload{Answers: map[string]json.RawMessage{
		"1": json.RawMessage(`"cat"`),
		"2": json.RawMessage(`"A"`),
		"3": json.RawMessage(`["A","B"]`),
	}}

	sub := submitAndGrade(t, svc, repo, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, payload)})
	if sub.Status != StatusGraded {
		t.Errorf("Status = %q, want %q", sub.Status, StatusGraded)
	}

	score, err := repo.GetScoreBySubmissionID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("expected a score to be created: %v", err)
	}
	if score.OverallBand == nil || *score.OverallBand != 9 {
		t.Errorf("OverallBand = %v, want 9 (100%% correct)", score.OverallBand)
	}

	var details AutoGradeDetails
	if err := json.Unmarshal(score.Details, &details); err != nil {
		t.Fatalf("unmarshal score details: %v", err)
	}
	if details.CorrectCount != 3 || details.TotalCount != 3 {
		t.Errorf("CorrectCount/TotalCount = %d/%d, want 3/3", details.CorrectCount, details.TotalCount)
	}
}

func TestService_SubmitAnswer_ReadingAutoGradesPartialCredit(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "reading",
		TaskType:    "academic",
		ContentData: mustMarshal(t, validReadingContent()),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	payload := AnswerPayload{Answers: map[string]json.RawMessage{
		"1": json.RawMessage(`"dog"`),     // wrong
		"2": json.RawMessage(`"A"`),       // correct
		"3": json.RawMessage(`["A","B"]`), // correct
	}}

	sub := submitAndGrade(t, svc, repo, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, payload)})

	score, err := repo.GetScoreBySubmissionID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("expected a score to be created: %v", err)
	}
	var details AutoGradeDetails
	if err := json.Unmarshal(score.Details, &details); err != nil {
		t.Fatalf("unmarshal score details: %v", err)
	}
	if details.CorrectCount != 2 {
		t.Errorf("CorrectCount = %d, want 2", details.CorrectCount)
	}
	if details.Results["1"].Correct {
		t.Error("expected question_order 1 to be marked incorrect")
	}
}

// ---------------------------------------------------------------------------
// GetSubmissionByID / GetListSubmission / GetScore — ownership checks
// ---------------------------------------------------------------------------

func TestService_GetSubmissionByID_EnforcesOwnership(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()

	sub, err := repo.CreateSubmission(ctx, &Submission{UserID: 1, TestID: 1, Payload: []byte(`{}`), Status: StatusPending})
	if err != nil {
		t.Fatalf("seed submission: %v", err)
	}

	if _, err := svc.GetSubmissionByID(ctx, 1, sub.ID); err != nil {
		t.Errorf("expected owner to fetch submission, got %v", err)
	}
	if _, err := svc.GetSubmissionByID(ctx, 2, sub.ID); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("expected ErrSubmissionNotFound for a different user, got %v", err)
	}
}

func TestService_GetScore_EnforcesOwnershipAndPendingState(t *testing.T) {
	repo := NewMockTestRepository()
	grader := &fakeGrader{result: &GradingResult{OverallBand: 6, Criteria: map[string]CriterionScore{}}}
	svc := newTestService(repo, grader)
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	sub := submitAndGrade(t, svc, repo, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, WritingPayload{Text: "essay"})})

	score, err := svc.GetScore(ctx, 1, sub.ID)
	if err != nil {
		t.Fatalf("GetScore: %v", err)
	}
	if score.OverallBand == nil || *score.OverallBand != 6 {
		t.Errorf("OverallBand = %v, want 6", score.OverallBand)
	}

	if _, err := svc.GetScore(ctx, 2, sub.ID); !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("expected ErrSubmissionNotFound for a different user, got %v", err)
	}

	pendingSub, err := repo.CreateSubmission(ctx, &Submission{UserID: 1, TestID: test.ID, Payload: []byte(`{}`), Status: StatusPending})
	if err != nil {
		t.Fatalf("seed pending submission: %v", err)
	}
	if _, err := svc.GetScore(ctx, 1, pendingSub.ID); !errors.Is(err, ErrScoreNotFound) {
		t.Errorf("expected ErrScoreNotFound for an ungraded submission, got %v", err)
	}
}

func TestService_GetListSubmission_ReturnsOnlyOwnedSubmissionsIncludingPending(t *testing.T) {
	repo := NewMockTestRepository()
	grader := &fakeGrader{result: &GradingResult{OverallBand: 6, Criteria: map[string]CriterionScore{}}}
	svc := newTestService(repo, grader)
	ctx := context.Background()

	test, err := repo.CreateTest(ctx, &Test{
		Skill:       "writing",
		TaskType:    "task2",
		ContentData: mustMarshal(t, WritingContent{Prompt: "Describe the chart."}),
	})
	if err != nil {
		t.Fatalf("seed test: %v", err)
	}

	// Owned + graded. Drained first, so the rows seeded below stay pending.
	submitAndGrade(t, svc, repo, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, WritingPayload{Text: "essay"})})
	// Owned + still pending (no score yet) — must still show up in the list.
	if _, err := repo.CreateSubmission(ctx, &Submission{UserID: 1, TestID: test.ID, Payload: []byte(`{}`), Status: StatusPending}); err != nil {
		t.Fatalf("seed pending submission: %v", err)
	}
	// Owned by a different user — must be excluded.
	if _, err := repo.CreateSubmission(ctx, &Submission{UserID: 2, TestID: test.ID, Payload: []byte(`{}`), Status: StatusPending}); err != nil {
		t.Fatalf("seed other user's submission: %v", err)
	}

	resp, err := svc.GetListSubmission(ctx, 1, ListSubmissionRequest{Page: 1})
	if err != nil {
		t.Fatalf("GetListSubmission: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("got %d submissions, want 2", len(resp.Data))
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("Pagination.Total = %d, want 2", resp.Pagination.Total)
	}

	var sawGraded, sawPending bool
	for _, d := range resp.Data {
		if d.OverallBand != nil {
			sawGraded = true
		} else {
			sawPending = true
		}
	}
	if !sawGraded || !sawPending {
		t.Errorf("expected both a graded and a pending submission in the list, sawGraded=%v sawPending=%v", sawGraded, sawPending)
	}
}
