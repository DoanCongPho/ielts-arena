package ielts_test

import "time"

type Test struct {
	ID          uint64
	Skill       string
	TaskType    string
	ContentData []byte
	Source      string
	IsCurrent   bool
	XPGain      int
	CreatedAt   time.Time
}

type Submission struct {
	ID          uint64
	UserID      uint64
	TestID      uint64
	Payload     []byte
	Status      string
	SubmittedAt time.Time
}

type Score struct {
	ID           uint64
	SubmissionID uint64
	OverallBand  *float64
	Details      []byte
	GradedAt     *time.Time
}

// WritingContent is the shape of Test.ContentData when Skill == "writing".
type WritingContent struct {
	Prompt   string `json:"prompt"`
	ImageURL string `json:"image_url,omitempty"`
}

// WritingPayload is the shape of Submission.Payload when the test's
// Skill == "writing".
type WritingPayload struct {
	Text string `json:"text"`
}

// ScoreDetails is what gets marshalled into Score.Details — the grading
// breakdown beyond the single overall_band column.
type ScoreDetails struct {
	Criteria    map[string]CriterionScore `json:"criteria"`
	Corrections []Correction              `json:"corrections"`
	ModelAnswer string                    `json:"model_answer"`
}
