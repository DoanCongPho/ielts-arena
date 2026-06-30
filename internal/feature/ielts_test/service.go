package ielts_test

import (
	"context"
)

type Service interface {
	GetCurrentTest(ctx context.Context, skill string) (*Test, error)
	SubmitAnswer(ctx context.Context, userID uint64, req SubmitRequest) (*Submission, error)
}

type service struct {
	repo   Repository
	grader Grader
}

func NewService(repo Repository, grader Grader) Service {
	return &service{repo: repo, grader: grader}
}

func (s *service) SubmitAnswer(ctx context.Context, userID uint64, req SubmitRequest) (*Submission, error) {
	test, _ := s.repo.GetTestByID(ctx, req.TestID)

	switch test.Skill {
	case "writing", "speaking":

	case "reading", "listening":

	}

	return nil, nil
}
func (s *service) GetCurrentTest(ctx context.Context, skill string) (*Test, error) {
	test, err := s.repo.GetCurrentTest(ctx, skill)
	if err != nil {
		return nil, err
	}
	return test, nil

}
