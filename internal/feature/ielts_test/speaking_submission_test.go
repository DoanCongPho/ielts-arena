package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

// fakeSpeakingGrader stands in for speaking/grading: the service only needs
// its verdict. It fails every attempt but the last when failUntilLast.
type fakeSpeakingGrader struct {
	failUntilLast bool
	calls         []bool // lastAttempt of each call
}

func (f *fakeSpeakingGrader) GradeSpeaking(_ context.Context, content SpeakingContent, payload SpeakingPayload, lastAttempt bool) (float64, json.RawMessage, error) {
	f.calls = append(f.calls, lastAttempt)
	if f.failUntilLast && !lastAttempt {
		return 0, nil, errors.New("pronunciation service unavailable")
	}
	details := fmt.Sprintf(`{"mode":%q,"answers":%d}`, content.Mode(), len(payload.Answers))
	return 6.5, json.RawMessage(details), nil
}

func speakingFixture(t *testing.T, grader SpeakingGrader) (Service, *MockTestRepository, *Test) {
	t.Helper()
	repo := NewMockTestRepository()
	test, _ := repo.CreateTest(context.Background(), &Test{Skill: "speaking", TaskType: "full", ContentData: mustMarshal(t, fullSpeakingContent())})
	var opts []ServiceOption
	if grader != nil {
		opts = append(opts, WithSpeakingGrader(grader, 0))
	}
	return NewService(repo, &fakeGrader{}, &fakeXPGrantRepository{}, opts...), repo, test
}

func speakingSubmission(t *testing.T, test *Test, userID uint64) SubmitRequest {
	t.Helper()
	var c SpeakingContent
	_ = json.Unmarshal(test.ContentData, &c)
	var p SpeakingPayload
	for _, q := range c.Questions() {
		p.Answers = append(p.Answers, SpeakingAnswer{QuestionID: q.ID, AudioKey: fmt.Sprintf("speaking/%d/%s.webm", userID, q.ID), DurationSec: 20})
	}
	return SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, p)}
}

func TestSpeaking_GradedAndStored(t *testing.T) {
	grader := &fakeSpeakingGrader{}
	svc, repo, test := speakingFixture(t, grader)

	sub := submitAndGrade(t, svc, repo, 1, speakingSubmission(t, test, 1))
	if sub.Status != StatusGraded {
		t.Fatalf("status = %q (last error %q)", sub.Status, sub.LastError)
	}
	score, err := repo.GetScoreBySubmissionID(context.Background(), sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *score.OverallBand != 6.5 || string(score.Details) != `{"mode":"full","answers":8}` {
		t.Errorf("stored %v %s, want the grader's band and details as they were", *score.OverallBand, score.Details)
	}
	if len(grader.calls) != 1 || grader.calls[0] {
		t.Errorf("grader calls (lastAttempt) = %v, want one first attempt", grader.calls)
	}
}

func TestSpeaking_RetriedUntilTheLastAttempt(t *testing.T) {
	grader := &fakeSpeakingGrader{failUntilLast: true}
	svc, repo, test := speakingFixture(t, grader)
	ctx := context.Background()

	sub, err := svc.SubmitAnswer(ctx, 1, speakingSubmission(t, test, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GradeNextPending(ctx); err == nil {
		t.Fatal("first attempt succeeded while the grader was failing")
	}
	if got, _ := repo.GetSubmissionByID(ctx, sub.ID); got.Status != StatusPending {
		t.Fatalf("status after a failed attempt = %q, want pending", got.Status)
	}

	got, _ := repo.GetSubmissionByID(ctx, sub.ID)
	got.Attempts = maxGradingAttempts - 1
	got.NextAttemptAt = nil
	if _, err := svc.GradeNextPending(ctx); err != nil {
		t.Fatalf("last attempt: %v", err)
	}
	if last := grader.calls[len(grader.calls)-1]; !last {
		t.Error("the last attempt wasn't flagged, so the grader couldn't fall back to an estimate")
	}
	if got, _ := repo.GetSubmissionByID(ctx, sub.ID); got.Status != StatusGraded {
		t.Errorf("status = %q, want graded", got.Status)
	}
}

func TestSpeaking_UngradableWithoutAGrader(t *testing.T) {
	svc, repo, test := speakingFixture(t, nil)
	sub := submitAndGrade(t, svc, repo, 1, speakingSubmission(t, test, 1))
	if sub.Status != StatusFailed {
		t.Errorf("status = %q, want failed without retries", sub.Status)
	}
}

func TestSpeaking_SubmissionValidation(t *testing.T) {
	svc, _, test := speakingFixture(t, &fakeSpeakingGrader{})
	ctx := context.Background()

	// User 2 submits user 1's recordings.
	req := speakingSubmission(t, test, 1)
	if _, err := svc.SubmitAnswer(ctx, 2, req); !errors.Is(err, ErrInvalidSubmission) {
		t.Errorf("someone else's recordings: err = %v, want ErrInvalidSubmission", err)
	}

	bad := SpeakingPayload{Answers: []SpeakingAnswer{{QuestionID: "p9", AudioKey: "speaking/1/x.webm", DurationSec: 5}}}
	if _, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, bad)}); !errors.Is(err, ErrInvalidSubmission) {
		t.Errorf("unknown question: err = %v, want ErrInvalidSubmission", err)
	}
}

func TestCustomTestsAreOwnerOnly(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()
	custom, _ := repo.CreateTest(ctx, &Test{OwnerID: 7, Skill: "speaking", TaskType: "part2", ContentData: mustMarshal(t, fullSpeakingContent())})

	if _, err := svc.GetTest(ctx, 7, custom.ID); err != nil {
		t.Errorf("owner can't open their test: %v", err)
	}
	if _, err := svc.GetTest(ctx, 8, custom.ID); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user opened it: err = %v", err)
	}
	if _, err := svc.SubmitAnswer(ctx, 8, SubmitRequest{TestID: custom.ID, Payload: []byte(`{}`)}); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user submitted to it: err = %v", err)
	}
	list, _ := svc.GetListTest(ctx, "speaking", ListTestRequest{Page: 1})
	if len(list.Data) != 0 {
		t.Errorf("custom test listed with the official ones: %+v", list.Data)
	}
}
