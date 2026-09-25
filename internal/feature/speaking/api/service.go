package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/feature/speaking/examiner"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"time"
)

const (
	// maxCustomTests bounds how many custom speaking tests one user keeps.
	maxCustomTests = 100
	// maxUploadSlots is how many upload links one request may ask for — a
	// full test has at most 15 + 3 + 8 answers.
	maxUploadSlots = 30
	uploadLinkTTL  = 30 * time.Minute
)

type (
	uploadStore interface {
		UploadURL(key string, expires time.Duration) (string, error)
		DownloadURL(key string, expires time.Duration) (string, error)
	}
	moderator interface {
		Flagged(ctx context.Context, texts []string) (bool, error)
	}
)

// Service serves everything speaking needs besides submitting and
// grading, which go through Service like every other skill.
type Service struct {
	repo      ielts_test.Repository
	store     uploadStore
	examiner  *examiner.Audio
	author    completer // nil: no content generation or style check
	moderator moderator // nil: no moderation
}

func NewService(repo ielts_test.Repository, store uploadStore, examiner *examiner.Audio, author completer, mod moderator) *Service {
	return &Service{repo: repo, store: store, examiner: examiner, author: author, moderator: mod}
}

// speakingTest loads a speaking test the user may take.
func (s *Service) speakingTest(ctx context.Context, userID, testID uint64) (*ielts_test.Test, *ielts_test.SpeakingContent, error) {
	t, err := s.repo.GetTestByID(ctx, testID)
	if err != nil {
		return nil, nil, err
	}
	if t.Skill != "speaking" || (t.OwnerID != 0 && t.OwnerID != userID) {
		return nil, nil, ielts_test.ErrTestNotFound
	}
	var c ielts_test.SpeakingContent
	if err := json.Unmarshal(t.ContentData, &c); err != nil {
		return nil, nil, fmt.Errorf("read test content: %w", err)
	}
	return t, &c, nil
}

func (s *Service) ownedTest(ctx context.Context, userID, testID uint64) (*ielts_test.Test, error) {
	t, err := s.repo.GetTestByID(ctx, testID)
	if err != nil {
		return nil, err
	}
	if t.OwnerID != userID || t.Skill != "speaking" {
		return nil, ielts_test.ErrTestNotFound
	}
	return t, nil
}

// completer is the LLM call behind "fill the rest" and the style check.
type completer interface {
	Complete(ctx context.Context, system, user, imageURL string, params llm.CompletionParams) (string, error)
}
