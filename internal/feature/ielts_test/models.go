package ielts_test

import "time"

const (
	StatusPending   = "pending"
	StatusSubmitted = "submitted"
	StatusGraded    = "graded"
	StatusFailed    = "failed"
)

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

// SpeakingContent is the shape of Test.ContentData when Skill == "speaking".
type SpeakingContent struct {
	Prompt string `json:"prompt"`
	Part   int    `json:"part"` // IELTS speaking part 1, 2, or 3
}

// SpeakingPayload is the shape of Submission.Payload when Skill == "speaking".
// Text holds the transcript of the candidate's spoken answer — this feature
// grades text only, transcription happens upstream of it.
type SpeakingPayload struct {
	Text string `json:"text"`
}

// QuestionType enumerates the auto-gradable reading/listening question formats.
type QuestionType string

const (
	QuestionMultipleChoice    QuestionType = "multiple_choice"
	QuestionTrueFalseNotGiven QuestionType = "true_false_not_given"
	QuestionFillBlank         QuestionType = "fill_blank"
)

// Question is one auto-gradable item within a reading or listening test.
// AnswerKey must never be sent to clients before grading — see
// Question.public in service.go.
type Question struct {
	ID        string       `json:"id"`
	Type      QuestionType `json:"type"`
	Text      string       `json:"text"`
	Options   []string     `json:"options,omitempty"`
	AnswerKey string       `json:"answer_key"`
}

// ReadingContent is the shape of Test.ContentData when Skill == "reading".
type ReadingContent struct {
	Passage   string     `json:"passage"`
	Questions []Question `json:"questions"`
}

// ListeningContent is the shape of Test.ContentData when Skill == "listening".
type ListeningContent struct {
	AudioURL  string     `json:"audio_url"`
	Questions []Question `json:"questions"`
}

// AnswerPayload is the shape of Submission.Payload for auto-graded skills
// (reading, listening): question id -> submitted answer.
type AnswerPayload struct {
	Answers map[string]string `json:"answers"`
}

// QuestionResult is the per-question outcome of auto-grading.
type QuestionResult struct {
	Correct         bool   `json:"correct"`
	SubmittedAnswer string `json:"submitted_answer"`
	CorrectAnswer   string `json:"correct_answer"`
}

// AutoGradeDetails is what gets marshalled into Score.Details for
// auto-graded skills (reading, listening).
type AutoGradeDetails struct {
	CorrectCount int                       `json:"correct_count"`
	TotalCount   int                       `json:"total_count"`
	Results      map[string]QuestionResult `json:"results"`
}
