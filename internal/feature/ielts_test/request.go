package ielts_test

import (
	"encoding/json"
	"errors"
	"time"

	"github/DoanCongPho/game-arena/internal/platform/httpx"
)

const defaultPageSize = 10

// timeLayout is the wire format used for all timestamps returned by this feature.
const timeLayout = "2006-01-02T15:04:05Z"

type ListTestRequest struct {
	Page int `json:"page"`
}

func (r *ListTestRequest) Limit() int { return defaultPageSize }

func (r *ListTestRequest) Offset() int {
	if r.Page <= 1 {
		return 0
	}
	return (r.Page - 1) * defaultPageSize
}

type ListTestResponse struct {
	Data       []TestResponse   `json:"data"`
	Pagination httpx.Pagination `json:"pagination"`
}

type ListSubmissionRequest struct {
	Page int `json:"page"`
}

func (r *ListSubmissionRequest) Limit() int { return defaultPageSize }

func (r *ListSubmissionRequest) Offset() int {
	if r.Page <= 1 {
		return 0
	}
	return (r.Page - 1) * defaultPageSize
}

type ListSubmissionResponse struct {
	Data       []SubmissionResponse `json:"data"`
	Pagination httpx.Pagination     `json:"pagination"`
}

type SubmitRequest struct {
	TestID  uint64          `json:"test_id"`
	Payload json.RawMessage `json:"payload"`
}

type CreateTestRequest struct {
	Skill       string          `json:"skill"`
	TaskType    string          `json:"task_type"`
	ContentData json.RawMessage `json:"content_data"`
	Source      string          `json:"source"`
	IsCurrent   bool            `json:"is_current"`
	XPGain      int             `json:"xp_gain"`
}

func (r *CreateTestRequest) Validate() error {
	switch r.Skill {
	case "writing", "speaking", "reading", "listening":
	default:
		return errors.New("skill must be one of: writing, speaking, reading, listening")
	}
	if r.TaskType == "" {
		return errors.New("task_type is required")
	}
	if len(r.ContentData) == 0 {
		return errors.New("content_data is required")
	}
	if r.XPGain < 0 {
		return errors.New("xp_gain cannot be negative")
	}
	return validateContentData(r.Skill, r.ContentData)
}

type TestResponse struct {
	ID          uint64          `json:"id"`
	Skill       string          `json:"skill"`
	TaskType    string          `json:"task_type"`
	ContentData json.RawMessage `json:"content_data"`
	XPGain      int             `json:"xp_gain"`
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

type SubmissionRequest struct {
	ID          uint64
	UserID      uint64
	TestID      uint64
	Payload     []byte
	Status      string
	SubmittedAt time.Time
}

func (r *SubmitRequest) Validate() error {
	if r.TestID == 0 {
		return errors.New("test_id is required")
	}
	if len(r.Payload) == 0 {
		return errors.New("payload cannot be empty")
	}
	return nil
}
