package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
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

	sub, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{
		TestID:  test.ID,
		Payload: mustMarshal(t, WritingPayload{Text: "My essay."}),
	})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}
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

func TestService_SubmitAnswer_WritingGraderFailureMarksSubmissionFailed(t *testing.T) {
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
	// The submission is preserved so the client can see/retry it, so the
	// grading failure surfaces via Status rather than as a returned error.
	if err != nil {
		t.Fatalf("expected no error, grading failure should surface via status: %v", err)
	}
	if sub.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", sub.Status, StatusFailed)
	}
	if _, err := repo.GetScoreBySubmissionID(ctx, sub.ID); !errors.Is(err, ErrScoreNotFound) {
		t.Errorf("expected no score to be created, got err = %v", err)
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

	sub, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, payload)})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}
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

	sub, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, payload)})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}

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

	sub, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, WritingPayload{Text: "essay"})})
	if err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}

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

	// Owned + graded.
	if _, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, WritingPayload{Text: "essay"})}); err != nil {
		t.Fatalf("SubmitAnswer: %v", err)
	}
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
