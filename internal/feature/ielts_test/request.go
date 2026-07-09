package ielts_test

import (
	"encoding/json"
	"errors"
)

const defaultPageSize = 10

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

type SubmitRequest struct {
	TestID  uint64          `json:"test_id"`
	Payload json.RawMessage `json:"payload"`
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

type CreateTestRequest struct {
	Skill        string          `json:"skill"`
	TaskType     string          `json:"task_type"`
	ContentData  json.RawMessage `json:"content_data"`
	ThumbnailURL string          `json:"thumbnail_url"`
	Source       string          `json:"source"`
	IsCurrent    bool            `json:"is_current"`
	XPGain       int             `json:"xp_gain"`
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
