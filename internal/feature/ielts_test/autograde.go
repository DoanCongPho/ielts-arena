package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// validateContentData checks that content_data matches the shape expected
// for skill before it's persisted, so bad data fails fast at creation time
// instead of when a test is later taken/graded.
func validateContentData(skill string, raw []byte) error {
	switch skill {
	case "writing":
		var c WritingContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("invalid content_data for writing: %w", err)
		}
		if c.Prompt == "" {
			return errors.New("content_data.prompt is required for writing")
		}
	case "speaking":
		var c SpeakingContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("invalid content_data for speaking: %w", err)
		}
		if c.Prompt == "" {
			return errors.New("content_data.prompt is required for speaking")
		}
	case "reading", "listening":
		questions, err := questionsFromContent(skill, raw)
		if err != nil {
			return fmt.Errorf("invalid content_data for %s: %w", skill, err)
		}
		if len(questions) == 0 {
			return fmt.Errorf("content_data.questions is required for %s", skill)
		}
		for _, q := range questions {
			if q.ID == "" || q.AnswerKey == "" {
				return errors.New("each question needs an id and answer_key")
			}
		}
	}
	return nil
}

// autoGradeSubmission scores a reading/listening submission against the
// test's answer key — no LLM call needed. It persists the resulting score
// and updates the submission status in place, mirroring gradeSubmission.
func (s *service) autoGradeSubmission(ctx context.Context, test *Test, sub *Submission) error {
	questions, err := questionsFromContent(test.Skill, test.ContentData)
	if err != nil {
		return fmt.Errorf("unmarshal test content: %w", err)
	}

	var payload AnswerPayload
	if err := json.Unmarshal(sub.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal submission payload: %w", err)
	}

	results := make(map[string]QuestionResult, len(questions))
	correct := 0
	for _, q := range questions {
		submitted := payload.Answers[q.ID]
		isCorrect := answersMatch(submitted, q.AnswerKey)
		if isCorrect {
			correct++
		}
		results[q.ID] = QuestionResult{
			Correct:         isCorrect,
			SubmittedAnswer: submitted,
			CorrectAnswer:   q.AnswerKey,
		}
	}

	details, err := json.Marshal(AutoGradeDetails{
		CorrectCount: correct,
		TotalCount:   len(questions),
		Results:      results,
	})
	if err != nil {
		return fmt.Errorf("marshal score details: %w", err)
	}

	overallBand := bandFromRawScore(correct, len(questions))
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

// publicContentData strips answer keys from reading/listening test content
// before it's exposed to clients. Other skills have nothing secret in their
// content, so it passes through unchanged.
func publicContentData(skill string, raw []byte) (json.RawMessage, error) {
	switch skill {
	case "reading":
		var content ReadingContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		content.Questions = redactAnswerKeys(content.Questions)
		return json.Marshal(content)
	case "listening":
		var content ListeningContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		content.Questions = redactAnswerKeys(content.Questions)
		return json.Marshal(content)
	default:
		return json.RawMessage(raw), nil
	}
}

func redactAnswerKeys(qs []Question) []Question {
	out := make([]Question, len(qs))
	for i, q := range qs {
		q.AnswerKey = ""
		out[i] = q
	}
	return out
}

func questionsFromContent(skill string, raw []byte) ([]Question, error) {
	switch skill {
	case "reading":
		var content ReadingContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		return content.Questions, nil
	case "listening":
		var content ListeningContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		return content.Questions, nil
	default:
		return nil, fmt.Errorf("no auto-gradable questions for skill %q", skill)
	}
}

func answersMatch(submitted, key string) bool {
	return normalizeAnswer(submitted) == normalizeAnswer(key)
}

func normalizeAnswer(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// bandFromRawScore approximates the published IELTS Listening / Academic
// Reading raw-to-band conversion table, scaled by percentage so it still
// applies to tests that don't have exactly 40 questions. It's an
// approximation for practice purposes, not the official scoring table.
func bandFromRawScore(correct, total int) float64 {
	if total == 0 {
		return 0
	}
	pct := float64(correct) / float64(total) * 100

	switch {
	case pct >= 97.5:
		return 9
	case pct >= 92.5:
		return 8.5
	case pct >= 87.5:
		return 8
	case pct >= 82.5:
		return 7.5
	case pct >= 75:
		return 7
	case pct >= 67.5:
		return 6.5
	case pct >= 57.5:
		return 6
	case pct >= 47.5:
		return 5.5
	case pct >= 37.5:
		return 5
	case pct >= 32.5:
		return 4.5
	case pct >= 25:
		return 4
	case pct >= 20:
		return 3.5
	case pct >= 15:
		return 3
	case pct >= 10:
		return 2.5
	default:
		return 2
	}
}
