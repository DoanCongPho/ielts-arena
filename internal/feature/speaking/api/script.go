package api

import (
	"context"
	"github/DoanCongPho/game-arena/internal/feature/speaking/examiner"
)

// ScriptResponse is what the exam runner plays through.
type ScriptResponse struct {
	TestID uint64          `json:"test_id"`
	Mode   string          `json:"mode"`
	Lines  []examiner.Line `json:"lines"`
}

func (s *Service) Script(ctx context.Context, userID, testID uint64) (*ScriptResponse, error) {
	t, content, err := s.speakingTest(ctx, userID, testID)
	if err != nil {
		return nil, err
	}
	lines := examiner.Build(*content, s.examiner.Voice())
	lines = s.examiner.Resolve(ctx, lines)
	return &ScriptResponse{TestID: t.ID, Mode: content.Mode(), Lines: lines}, nil
}
