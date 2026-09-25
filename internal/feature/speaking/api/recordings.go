package api

import (
	"context"
	"encoding/json"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"strings"
	"time"
)

// Recordings returns playback links for a submission's answers, keyed by
// question id — only to the user who made them.
func (s *Service) Recordings(ctx context.Context, userID, submissionID uint64) (map[string]string, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub.UserID != userID {
		return nil, ielts_test.ErrSubmissionNotFound
	}
	var p ielts_test.SpeakingPayload
	if err := json.Unmarshal(sub.Payload, &p); err != nil {
		return nil, ielts_test.ErrSubmissionNotFound
	}
	out := map[string]string{}
	for _, a := range p.Answers {
		if !strings.HasPrefix(a.AudioKey, ielts_test.SpeakingAudioPrefix(userID)) {
			continue
		}
		if u, err := s.store.DownloadURL(a.AudioKey, time.Hour); err == nil {
			out[a.QuestionID] = u
		}
	}
	return out, nil
}
