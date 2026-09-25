package ielts_test

import (
	"encoding/json"
	"errors"
)

const defaultPageSize = 12

type ListTestRequest struct {
	Page int `json:"page"`
	TestFilter
}

// TestFilter narrows a test list. Filtering happens in the query, not on
// the returned page, so the total and the page count reflect the filter.
type TestFilter struct {
	// TaskType matches exactly: "task1", "part2", "full", "test1"…
	TaskType string
	// Series is a series name ("cambridge"), or SeriesNone for tests that
	// belong to no series.
	Series string
}

// SeriesNone selects tests outside any series.
const SeriesNone = "none"

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

type CreateTestRequest struct {
	Skill        string          `json:"skill"`
	TaskType     string          `json:"task_type"`
	Series       string          `json:"series,omitempty"`
	Volume       int             `json:"volume,omitempty"`
	TestNumber   int             `json:"test_number,omitempty"`
	ContentData  json.RawMessage `json:"content_data"`
	ThumbnailURL string          `json:"thumbnail_url"`
	Source       string          `json:"source"`
	IsCurrent    bool            `json:"is_current"`
	XPGain       int             `json:"xp_gain"`
}

// Test is the Test this request creates.
func (r *CreateTestRequest) Test() Test {
	return Test{
		Skill:        r.Skill,
		TaskType:     r.TaskType,
		Series:       r.Series,
		Volume:       r.Volume,
		TestNumber:   r.TestNumber,
		ContentData:  r.ContentData,
		ThumbnailURL: r.ThumbnailURL,
		Source:       r.Source,
		IsCurrent:    r.IsCurrent,
		XPGain:       r.XPGain,
	}
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
	if r.Series != "" && (r.Volume <= 0 || r.TestNumber <= 0) {
		return errors.New("a test in a series needs a positive volume and test_number")
	}
	if r.XPGain < 0 {
		return errors.New("xp_gain cannot be negative")
	}
	// The grader's rubric and the attempt page's timer both key on it.
	if r.Skill == "writing" && r.TaskType != "task1" && r.TaskType != "task2" {
		return errors.New("a writing test's task_type must be task1 or task2")
	}
	canonical, err := canonicalContentData(r.Skill, r.ContentData)
	if err != nil {
		return err
	}
	r.ContentData = canonical
	if err := validateContentData(r.Skill, r.ContentData); err != nil {
		return err
	}
	if r.Skill == "speaking" {
		// A speaking test's task type is its mode, whatever was sent, so
		// the list filters (full test / one part) always agree with it.
		r.TaskType = speakingModeOf(r.ContentData)
	}
	return nil
}
