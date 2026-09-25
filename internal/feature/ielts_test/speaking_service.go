package ielts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SpeakingAudioPrefix is where a user's recordings live. A submission may
// only reference keys under the submitter's own prefix, so nobody can have
// someone else's recording graded (or read back) as theirs.
func SpeakingAudioPrefix(userID uint64) string {
	return fmt.Sprintf("speaking/%d/", userID)
}

// maxAnswerSeconds is a generous ceiling on one recording. The runner stops
// the Part 2 talk at two minutes; nothing else should come close.
const maxAnswerSeconds = 180

// validateSpeakingSubmission checks a speaking payload before it's queued.
func validateSpeakingSubmission(userID uint64, test *Test, raw json.RawMessage) error {
	var content SpeakingContent
	if err := json.Unmarshal(test.ContentData, &content); err != nil {
		return fmt.Errorf("read test content: %w", err)
	}
	var payload SpeakingPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("%w: payload is not a speaking submission: %v", ErrInvalidSubmission, err)
	}
	if len(payload.Answers) == 0 {
		return fmt.Errorf("%w: no answers recorded", ErrInvalidSubmission)
	}
	known := map[string]bool{}
	for _, q := range content.Questions() {
		known[q.ID] = true
	}
	prefix := SpeakingAudioPrefix(userID)
	seen := map[string]bool{}
	for _, a := range payload.Answers {
		switch {
		case !known[a.QuestionID]:
			return fmt.Errorf("%w: unknown question %q", ErrInvalidSubmission, a.QuestionID)
		case seen[a.QuestionID]:
			return fmt.Errorf("%w: question %q answered twice", ErrInvalidSubmission, a.QuestionID)
		case !strings.HasPrefix(a.AudioKey, prefix) || strings.Contains(a.AudioKey, ".."):
			return fmt.Errorf("%w: recording for %q was not uploaded by you", ErrInvalidSubmission, a.QuestionID)
		case a.DurationSec <= 0 || a.DurationSec > maxAnswerSeconds:
			return fmt.Errorf("%w: recording for %q is %.0f s long", ErrInvalidSubmission, a.QuestionID, a.DurationSec)
		}
		seen[a.QuestionID] = true
	}
	return nil
}

// gradeSpeaking rates a speaking submission against the band descriptors
// and stores the score.
func (s *service) gradeSpeaking(ctx context.Context, test *Test, sub *Submission) error {
	if s.speaking == nil {
		return fmt.Errorf("%w: speaking grading is not configured", ErrUngradable)
	}
	var content SpeakingContent
	if err := json.Unmarshal(test.ContentData, &content); err != nil {
		return fmt.Errorf("%w: unmarshal test content: %v", ErrUngradable, err)
	}
	var payload SpeakingPayload
	if err := json.Unmarshal(sub.Payload, &payload); err != nil {
		return fmt.Errorf("%w: unmarshal submission payload: %v", ErrUngradable, err)
	}

	// Retry while the pronunciation service may just be cold or busy; on
	// the last attempt, estimate pronunciation rather than fail the test.
	lastAttempt := sub.Attempts >= maxGradingAttempts
	details, overall, err := s.speaking.Grade(ctx, content, payload, lastAttempt)
	if err != nil {
		return fmt.Errorf("grade speaking: %w", err)
	}

	raw, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal score details: %w", err)
	}
	if _, err := s.repo.CreateScore(ctx, &Score{SubmissionID: sub.ID, OverallBand: &overall, Details: raw}); err != nil {
		return fmt.Errorf("create score: %w", err)
	}
	sub.Status = StatusGraded
	return s.repo.UpdateSubmissionStatus(ctx, sub.ID, StatusGraded)
}
