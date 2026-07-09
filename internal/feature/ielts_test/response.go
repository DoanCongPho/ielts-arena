package ielts_test

import (
	"encoding/json"

	"github/DoanCongPho/game-arena/internal/platform/httpx"
)

// timeLayout is the wire format used for all timestamps returned by this feature.
const timeLayout = "2006-01-02T15:04:05Z"

type ListTestResponse struct {
	Data       []TestResponse   `json:"data"`
	Pagination httpx.Pagination `json:"pagination"`
}

type TestResponse struct {
	ID           uint64          `json:"id"`
	Skill        string          `json:"skill"`
	TaskType     string          `json:"task_type"`
	ContentData  json.RawMessage `json:"content_data"`
	ThumbnailURL string          `json:"thumbnail_url,omitempty"`
	XPGain       int             `json:"xp_gain"`
}

// newTestResponse builds the public wire shape for a Test — content is
// pre-sanitized by the caller (publicContentData strips answer keys for
// reading/listening before it gets here).
func newTestResponse(t *Test, content json.RawMessage) TestResponse {
	return TestResponse{
		ID:           t.ID,
		Skill:        t.Skill,
		TaskType:     t.TaskType,
		ContentData:  content,
		ThumbnailURL: t.ThumbnailURL,
		XPGain:       t.XPGain,
	}
}

type SubmissionResponse struct {
	ID          uint64          `json:"id"`
	TestID      uint64          `json:"test_id"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	SubmittedAt string          `json:"submitted_at"`
}

func newSubmissionResponse(sub *Submission) SubmissionResponse {
	return SubmissionResponse{
		ID:          sub.ID,
		TestID:      sub.TestID,
		Payload:     json.RawMessage(sub.Payload),
		Status:      sub.Status,
		SubmittedAt: sub.SubmittedAt.Format(timeLayout),
	}
}

type ListSubmissionResponse struct {
	Data       []SubmissionSummaryResponse `json:"data"`
	Pagination httpx.Pagination            `json:"pagination"`
}

// SubmissionSummaryResponse is the shape returned by the submission history
// list — it folds in the test's skill/task_type and the score's overall_band
// (if graded yet) so the list page doesn't need per-row follow-up requests.
type SubmissionSummaryResponse struct {
	ID           uint64          `json:"id"`
	TestID       uint64          `json:"test_id"`
	TestSkill    string          `json:"test_skill"`
	TestTaskType string          `json:"test_task_type"`
	TestXPGain   int             `json:"test_xp_gain"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	SubmittedAt  string          `json:"submitted_at"`
	OverallBand  *float64        `json:"overall_band"`
	GradedAt     *string         `json:"graded_at"`
}

func newSubmissionSummaryResponse(sm SubmissionSummary) SubmissionSummaryResponse {
	resp := SubmissionSummaryResponse{
		ID:           sm.ID,
		TestID:       sm.TestID,
		TestSkill:    sm.TestSkill,
		TestTaskType: sm.TestTaskType,
		TestXPGain:   sm.TestXPGain,
		Payload:      json.RawMessage(sm.Payload),
		Status:       sm.Status,
		SubmittedAt:  sm.SubmittedAt.Format(timeLayout),
		OverallBand:  sm.OverallBand,
	}
	if sm.GradedAt != nil {
		formatted := sm.GradedAt.Format(timeLayout)
		resp.GradedAt = &formatted
	}
	return resp
}

type ScoreResponse struct {
	ID           uint64          `json:"id"`
	SubmissionID uint64          `json:"submission_id"`
	OverallBand  *float64        `json:"overall_band"`
	Details      json.RawMessage `json:"details"`
	GradedAt     *string         `json:"graded_at"`
}

func newScoreResponse(sc *Score) ScoreResponse {
	resp := ScoreResponse{
		ID:           sc.ID,
		SubmissionID: sc.SubmissionID,
		OverallBand:  sc.OverallBand,
		Details:      json.RawMessage(sc.Details),
	}
	if sc.GradedAt != nil {
		formatted := sc.GradedAt.Format(timeLayout)
		resp.GradedAt = &formatted
	}
	return resp
}
