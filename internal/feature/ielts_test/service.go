package ielts_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github/DoanCongPho/game-arena/internal/platform/httpx"

	"github.com/gorilla/mux"
)

type Service interface {
	MountRoutes(r *mux.Router)
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
}

type service struct {
	repo   Repository
	grader Grader
}

func NewService(repo Repository, grader Grader) Service {
	return &service{repo: repo, grader: grader}
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

func (s *service) SubmitAnswer(ctx context.Context, userID uint64, req SubmitRequest) (*Submission, error) {
	test, err := s.repo.GetTestByID(ctx, req.TestID)
	if err != nil {
		return nil, err
	}

	sub, err := s.repo.CreateSubmission(ctx, &Submission{
		UserID:  userID,
		TestID:  req.TestID,
		Payload: req.Payload,
		Status:  StatusPending,
	})
	if err != nil {
		return nil, err
	}

	switch test.Skill {
	case "writing", "speaking":
		if err := s.gradeSubmission(ctx, test, sub); err != nil {
			// Submission is already persisted; surface the grading failure via
			// status so the client can retry/poll instead of losing the answer.
			sub.Status = StatusFailed
			_ = s.repo.UpdateSubmissionStatus(ctx, sub.ID, StatusFailed)
			return sub, nil
		}
	case "reading", "listening":
		if err := s.autoGradeSubmission(ctx, test, sub); err != nil {
			sub.Status = StatusFailed
			_ = s.repo.UpdateSubmissionStatus(ctx, sub.ID, StatusFailed)
			return sub, nil
		}
	}

	return sub, nil
}

// gradeSubmission runs the LLM grader for a writing/speaking submission and
// persists the resulting score, updating the submission status in place.
func (s *service) gradeSubmission(ctx context.Context, test *Test, sub *Submission) error {
	var content WritingContent
	if err := json.Unmarshal(test.ContentData, &content); err != nil {
		return fmt.Errorf("unmarshal test content: %w", err)
	}
	var payload WritingPayload
	if err := json.Unmarshal(sub.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal submission payload: %w", err)
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
