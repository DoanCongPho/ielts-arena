package ielts_test

import (
	"bytes"
	"encoding/json"
	"time"
)

const (
	StatusPending   = "pending"
	StatusSubmitted = "submitted"
	StatusGraded    = "graded"
	StatusFailed    = "failed"
)

type Test struct {
	ID           uint64
	Skill        string
	TaskType     string
	ContentData  []byte
	ThumbnailURL string
	Source       string
	IsCurrent    bool
	XPGain       int
	CreatedAt    time.Time
}

type Submission struct {
	ID          uint64
	UserID      uint64
	TestID      uint64
	Payload     []byte
	Status      string
	SubmittedAt time.Time
}

// SubmissionSummary is a Submission enriched with its test's skill/task_type
// and (if graded) its score — the shape used for the submission history
// list, so the list page doesn't need to re-fetch the test and score for
// every row.
type SubmissionSummary struct {
	Submission
	TestSkill    string
	TestTaskType string
	TestXPGain   int
	OverallBand  *float64
	GradedAt     *time.Time
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

// ---------------------------------------------------------------------------
// Reading/Listening question model, per docs/ielts-rl-data-structure.md.
// ---------------------------------------------------------------------------

// QuestionType enumerates the auto-gradable reading/listening question
// formats. Reading and Listening each support a subset of these — see
// readingQuestionTypes/listeningQuestionTypes.
type QuestionType string

const (
	QTypeTrueFalseNotGiven       QuestionType = "true-false-not-given"
	QTypeYesNoNotGiven           QuestionType = "yes-no-not-given"
	QTypeMultipleChoice          QuestionType = "multiple-choice"
	QTypeMultipleChoiceMulti     QuestionType = "multiple-choice-multi"
	QTypeMatchingHeadings        QuestionType = "matching-headings"
	QTypeMatchingInformation     QuestionType = "matching-information"
	QTypeMatchingFeatures        QuestionType = "matching-features"
	QTypeMatchingSentenceEndings QuestionType = "matching-sentence-endings"
	QTypeSentenceCompletion      QuestionType = "sentence-completion"
	QTypeSummaryCompletion       QuestionType = "summary-completion"
	QTypeTableCompletion         QuestionType = "table-completion"
	QTypeShortAnswer             QuestionType = "short-answer"
	QTypeDiagramLabelCompletion  QuestionType = "diagram-label-completion"
	QTypeFlowChartCompletion     QuestionType = "flow-chart-completion"
	// Listening-only.
	QTypeFormCompletion   QuestionType = "form-completion"
	QTypeNoteCompletion   QuestionType = "note-completion"
	QTypeMatching         QuestionType = "matching"
	QTypeMapPlanLabelling QuestionType = "map-plan-labelling"
)

// readingQuestionTypes is every question_type valid inside a reading
// passage's question_groups. Per docs/ielts-rl-data-structure.md's table
// this is 14 types; note-completion is additionally allowed here even
// though that doc scoped it as listening-only, since real IELTS Academic/
// General Reading tests also use note completion (confirmed by real test
// content added to the app).
var readingQuestionTypes = map[QuestionType]bool{
	QTypeTrueFalseNotGiven:       true,
	QTypeYesNoNotGiven:           true,
	QTypeMultipleChoice:          true,
	QTypeMultipleChoiceMulti:     true,
	QTypeMatchingHeadings:        true,
	QTypeMatchingInformation:     true,
	QTypeMatchingFeatures:        true,
	QTypeMatchingSentenceEndings: true,
	QTypeSentenceCompletion:      true,
	QTypeSummaryCompletion:       true,
	QTypeTableCompletion:         true,
	QTypeShortAnswer:             true,
	QTypeDiagramLabelCompletion:  true,
	QTypeFlowChartCompletion:     true,
	QTypeNoteCompletion:          true,
}

// listeningQuestionTypes is every question_type valid inside a listening
// section's question_groups (10 types per docs/ielts-rl-data-structure.md).
var listeningQuestionTypes = map[QuestionType]bool{
	QTypeFormCompletion:      true,
	QTypeNoteCompletion:      true,
	QTypeTableCompletion:     true,
	QTypeFlowChartCompletion: true,
	QTypeSummaryCompletion:   true,
	QTypeSentenceCompletion:  true,
	QTypeMultipleChoice:      true,
	QTypeMultipleChoiceMulti: true,
	QTypeMatching:            true,
	QTypeMapPlanLabelling:    true,
}

// fillBlankQuestionTypes are the types graded by checking the submitted
// text against a per-question AcceptedAnswers list, rather than by exact
// key match or multi-select set match.
var fillBlankQuestionTypes = map[QuestionType]bool{
	QTypeSentenceCompletion:     true,
	QTypeSummaryCompletion:      true, // only when the group has no word bank — see answersMatch
	QTypeTableCompletion:        true,
	QTypeNoteCompletion:         true,
	QTypeFlowChartCompletion:    true,
	QTypeFormCompletion:         true,
	QTypeShortAnswer:            true,
	QTypeDiagramLabelCompletion: true,
}

// Option is a single {id, text} choice, used for per-question multiple
// choice options, group-level shared_options (matching types), word banks,
// and map-plan-labelling location keys.
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// TableStructure backs table-completion groups (Reading and Listening
// share this shape). A cell containing the literal string "{{gap}}" is a
// blank — the i-th gap found in row-major order maps to Questions[i].
type TableStructure struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// NoteStructure backs note-completion groups (listening). Each item may
// contain "{{gap}}"; gaps map to Questions in array order.
type NoteStructure struct {
	Title string   `json:"title,omitempty"`
	Items []string `json:"items"`
}

// FlowStructure backs flow-chart-completion groups. Each step may contain
// "{{gap}}"; gaps map to Questions in array order.
type FlowStructure struct {
	Steps []string `json:"steps"`
}

// FormStructure backs form-completion groups (listening). Each field may
// contain "{{gap}}" (e.g. "Name: {{gap}}"); gaps map to Questions in array
// order.
type FormStructure struct {
	Title  string   `json:"title,omitempty"`
	Fields []string `json:"fields"`
}

// AnswerValue holds a question's canonical answer. On the wire it's a
// plain string for single-key types (true/false, multiple-choice,
// matching-*, summary-completion-with-word-bank) or a JSON array of keys
// for multiple-choice-multi. It round-trips whatever shape was authored
// instead of forcing one canonical internal shape, so redaction/storage
// never need to interpret it — only grading calls Strings().
type AnswerValue struct {
	raw json.RawMessage
}

func (a AnswerValue) MarshalJSON() ([]byte, error) {
	if len(a.raw) == 0 {
		return []byte("null"), nil
	}
	return a.raw, nil
}

func (a *AnswerValue) UnmarshalJSON(b []byte) error {
	a.raw = append(a.raw[:0], b...)
	return nil
}

// IsEmpty reports whether no answer was set (absent field or explicit null).
func (a AnswerValue) IsEmpty() bool {
	return len(a.raw) == 0 || bytes.Equal(bytes.TrimSpace(a.raw), []byte("null"))
}

// Strings decodes the answer into a uniform []string view: a scalar string
// becomes a single-element slice, an array is returned as-is. Returns nil
// for an empty/invalid answer.
func (a AnswerValue) Strings() []string {
	if a.IsEmpty() {
		return nil
	}
	var s string
	if err := json.Unmarshal(a.raw, &s); err == nil {
		return []string{s}
	}
	var arr []string
	if err := json.Unmarshal(a.raw, &arr); err == nil {
		return arr
	}
	return nil
}

// Question is one gradable item within a question_group. Which fields are
// populated depends on the owning group's QuestionType: Text is empty for
// questions answered via a shared structure (table/note/flow/form/summary
// completion), where the blank's position comes from the i-th "{{gap}}"
// marker in that structure (in document order), matching the i-th question
// in the group by QuestionOrder. AnswerKey/AcceptedAnswers must never be
// sent to clients before grading — see redactQuestionsInPlace.
type Question struct {
	QuestionOrder   int         `json:"question_order"`
	Text            string      `json:"text,omitempty"`
	Answer          AnswerValue `json:"answer"`
	AcceptedAnswers []string    `json:"accepted_answers,omitempty"`
	// TimestampHint is listening-only: seconds into the shared audio file
	// where this question is addressed. It's a review-UI convenience only
	// (e.g. "jump to the moment you got this wrong") — never used for
	// grading, and not applicable during a live attempt.
	TimestampHint *float64 `json:"timestamp_hint,omitempty"`
	// Options are per-question multiple-choice/matching-sentence-endings
	// options, used when the group doesn't provide SharedOptions instead.
	Options []Option `json:"options,omitempty"`
}

// QuestionGroup is one block of questions sharing a type and instruction,
// per docs/ielts-rl-data-structure.md. Which fields are populated depends
// on Type — see the field comments below.
type QuestionGroup struct {
	GroupOrder      int             `json:"group_order"`
	QuestionType    QuestionType    `json:"question_type"`
	Instructions    string          `json:"instructions"`
	Questions       []Question      `json:"questions"`
	SharedOptions   []Option        `json:"shared_options,omitempty"`    // matching-headings/information/features/sentence-endings/matching
	SelectCount     int             `json:"select_count,omitempty"`      // multiple-choice-multi
	AllowReuse      *bool           `json:"allow_reuse,omitempty"`       // matching-information (true) / matching-features (false) — UI hint only
	WordLimit       int             `json:"word_limit,omitempty"`        // sentence-completion / short-answer — UI hint only
	HasWordBank     bool            `json:"has_word_bank,omitempty"`     // summary-completion
	WordBank        []Option        `json:"word_bank,omitempty"`         // summary-completion, when HasWordBank
	SummaryText     string          `json:"summary_text,omitempty"`      // summary-completion; contains "{{gap}}"
	TableStructure  *TableStructure `json:"table_structure,omitempty"`   // table-completion
	NoteStructure   *NoteStructure  `json:"note_structure,omitempty"`    // note-completion
	FlowStructure   *FlowStructure  `json:"flow_structure,omitempty"`    // flow-chart-completion
	FormStructure   *FormStructure  `json:"form_structure,omitempty"`    // form-completion
	DiagramImageURL string          `json:"diagram_image_url,omitempty"` // diagram-label-completion
	MapImageURL     string          `json:"map_image_url,omitempty"`     // map-plan-labelling (required)
	LocationKey     []Option        `json:"location_key,omitempty"`      // map-plan-labelling
}

// Paragraph is one lettered paragraph within a reading passage.
type Paragraph struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

// ReadingContent is the shape of Test.ContentData when Skill == "reading".
type ReadingContent struct {
	Passages []ReadingPassage `json:"passages"`
}

// ReadingPassage is one passage within a reading test. QuestionOrder must
// be unique and globally contiguous across every passage in the test (not
// reset per passage) since AnswerPayload/AutoGradeDetails key answers by
// that number alone.
type ReadingPassage struct {
	Title          string          `json:"title,omitempty"`
	Paragraphs     []Paragraph     `json:"paragraphs"`
	QuestionGroups []QuestionGroup `json:"question_groups"`
}

// ListeningContent is the shape of Test.ContentData when Skill ==
// "listening". Unlike ReadingContent, all sections share ONE audio file —
// AudioURL — and each section marks its own start/end offset into it.
type ListeningContent struct {
	AudioURL string             `json:"audio_url"`
	Sections []ListeningSection `json:"sections"`
}

// ListeningSection is one section within a listening test, addressed by a
// start/end offset (seconds) into ListeningContent.AudioURL.
type ListeningSection struct {
	Title            string          `json:"title,omitempty"`
	SectionStartTime float64         `json:"section_start_time"`
	SectionEndTime   float64         `json:"section_end_time"`
	QuestionGroups   []QuestionGroup `json:"question_groups"`
}

// AnswerPayload is the shape of Submission.Payload for auto-graded skills
// (reading, listening): question_order (as a decimal string) -> submitted
// answer. The submitted value is a JSON string for single-answer question
// types, or a JSON array of strings for multiple-choice-multi.
type AnswerPayload struct {
	Answers map[string]json.RawMessage `json:"answers"`
}

// QuestionResult is the per-question outcome of auto-grading. Submitted/
// correct answers are always represented as string slices — a single-
// element slice for single-answer types — so callers don't need to
// special-case multi-select vs. single-select display.
type QuestionResult struct {
	Correct         bool     `json:"correct"`
	SubmittedAnswer []string `json:"submitted_answer"`
	CorrectAnswer   []string `json:"correct_answer"`
}

// AutoGradeDetails is what gets marshalled into Score.Details for
// auto-graded skills (reading, listening). Results is keyed by
// question_order (as a decimal string), matching AnswerPayload.Answers.
type AutoGradeDetails struct {
	CorrectCount int                       `json:"correct_count"`
	TotalCount   int                       `json:"total_count"`
	Results      map[string]QuestionResult `json:"results"`
}
