package ielts_test

import (
	"bytes"
	"encoding/json"
	"time"
)

// Submission lifecycle. A submission is created "pending", a grading
// worker claims it into "grading", and it settles as "graded" or "failed".
// A transient failure goes back to "pending" with next_attempt_at set,
// so "failed" means permanently failed — out of retries, or ungradable.
const (
	StatusPending   = "pending"
	StatusSubmitted = "submitted"
	StatusGrading   = "grading"
	StatusGraded    = "graded"
	StatusFailed    = "failed"
)

type Test struct {
	ID       uint64
	Skill    string
	TaskType string
	// Series/Volume/TestNumber place a test in a book ("cambridge", 20, 1 =
	// Cambridge 20 Test 1). Series is "" for tests outside any series.
	Series       string
	Volume       int
	TestNumber   int
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

	// Grading-queue bookkeeping, written only by the grading worker.
	Attempts      int
	LastError     string
	NextAttemptAt *time.Time
	ClaimedAt     *time.Time
}

// dto for viewing the submission
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

// --- writing test ---
type WritingContent struct {
	Prompt   string `json:"prompt"`
	ImageURL string `json:"image_url,omitempty"`
}

type WritingPayload struct {
	Text string `json:"text"`
}

type ScoreDetails struct {
	Criteria    map[string]CriterionScore `json:"criteria"`
	Corrections []Correction              `json:"corrections"`
	ModelAnswer string                    `json:"model_answer"`
}

// --- speaking test ---
type SpeakingContent struct {
	Prompt string `json:"prompt"`
	Part   int    `json:"part"` // IELTS speaking part 1, 2, or 3
}

type SpeakingPayload struct {
	Text string `json:"text"`
}

// --- Reading/Listening --> docs/ielts-rl-data-structure.md. ---

type ReadingContent struct {
	Passages []ReadingPassage `json:"passages"`
}
type ReadingPassage struct {
	Title          string          `json:"title,omitempty"`
	Paragraphs     []Paragraph     `json:"paragraphs"`
	QuestionGroups []QuestionGroup `json:"question_groups"`
}

type ListeningContent struct {
	// AudioURL is one recording of the whole test, each section a time
	// range in it. It may be empty when every section has its own AudioURL.
	AudioURL string             `json:"audio_url,omitempty"`
	Sections []ListeningSection `json:"sections"`
}

type ListeningSection struct {
	Title string `json:"title,omitempty"`
	// AudioURL, when set, is this section's own recording: its start/end
	// times and its questions' timestamp_hints are then seconds into this
	// file instead of into ListeningContent.AudioURL.
	AudioURL         string  `json:"audio_url,omitempty"`
	SectionStartTime float64 `json:"section_start_time"`
	SectionEndTime   float64 `json:"section_end_time"`
	// Transcript is what's said in the section — the text a listening
	// question's Evidence quotes. It gives the answers away, so it's
	// stripped from public content and served with the answer key.
	Transcript     []Paragraph     `json:"transcript,omitempty"`
	QuestionGroups []QuestionGroup `json:"question_groups"`
}

type Paragraph struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type QuestionGroup struct {
	GroupOrder      int             `json:"group_order"`
	QuestionType    QuestionType    `json:"question_type"`
	Instructions    string          `json:"instructions"`
	Questions       []Question      `json:"questions"`
	SharedOptions   []Option        `json:"shared_options,omitempty"`    // matching-headings/information/features/sentence-endings/matching
	SelectCount     int             `json:"select_count,omitempty"`      // multiple-choice-multi; also how many question numbers each question spans
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
	// DiagramImageURLs are further diagrams, after DiagramImageURL, when one
	// group labels several (Cambridge 16 Test 4, Questions 1-6).
	DiagramImageURLs []string `json:"diagram_image_urls,omitempty"`
	MapImageURL      string   `json:"map_image_url,omitempty"` // map-plan-labelling (required)
	LocationKey      []Option `json:"location_key,omitempty"`  // map-plan-labelling
}

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
	// Explanation (why the answer is right) and Evidence (where the passage,
	// or a listening section's transcript, says so) give the answer away, so
	// like Answer they're stripped from public content and only served by
	// the answer key after grading.
	Explanation string     `json:"explanation,omitempty"`
	Evidence    []Evidence `json:"evidence,omitempty"`
}

// Evidence points at where a question's answer is found: Paragraph indexes
// the passage's Paragraphs (reading) or the section's Transcript
// (listening) and Quote is a verbatim excerpt of it, which the review
// highlights. A quote rather than character offsets, so it survives
// whitespace changes and is easy to author by hand or by an AI importer.
type Evidence struct {
	Paragraph int    `json:"paragraph"`
	Quote     string `json:"quote"`
}

// AnswerKey is what GET /tests/{id}/answer-key serves once unlocked.
type AnswerKey struct {
	// Questions is keyed by question_order, like score results.
	Questions map[string]AnswerKeyEntry `json:"questions"`
	// Transcripts holds each listening section's transcript, by section
	// index — the text that section's Evidence points into.
	Transcripts [][]Paragraph `json:"transcripts,omitempty"`
}

// AnswerKeyEntry is one question's answer and explanation.
type AnswerKeyEntry struct {
	Answer          AnswerValue `json:"answer"`
	AcceptedAnswers []string    `json:"accepted_answers,omitempty"`
	Explanation     string      `json:"explanation,omitempty"`
	Evidence        []Evidence  `json:"evidence,omitempty"`
}

type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// QuestionResult is the per-question outcome of auto-grading. Submitted/
// correct answers are always represented as string slices — a single-
// element slice for single-answer types — so callers don't need to
// special-case multi-select vs. single-select display. Correct means full
// marks; Points/MaxPoints carry partial credit for a multiple-choice-multi
// question, which is worth one mark per key (see questionSpan).
type QuestionResult struct {
	Correct         bool     `json:"correct"`
	Points          int      `json:"points"`
	MaxPoints       int      `json:"max_points"`
	SubmittedAnswer []string `json:"submitted_answer"`
	CorrectAnswer   []string `json:"correct_answer"`
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

func (a AnswerValue) IsEmpty() bool {
	return len(a.raw) == 0 || bytes.Equal(bytes.TrimSpace(a.raw), []byte("null"))
}

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

// AnswerPayload is the shape of Submission.Payload for auto-graded skills
// (reading, listening): question_order (as a decimal string) -> submitted
// answer. The submitted value is a JSON string for single-answer question
// types, or a JSON array of strings for multiple-choice-multi.
type AnswerPayload struct {
	Answers map[string]json.RawMessage `json:"answers"`
}

// AutoGradeDetails is what gets marshalled into Score.Details for
// auto-graded skills (reading, listening). Results is keyed by
// question_order (as a decimal string), matching AnswerPayload.Answers.
// CorrectCount/TotalCount count marks, not question objects: a "Choose TWO"
// question contributes two to TotalCount.
type AutoGradeDetails struct {
	CorrectCount int                       `json:"correct_count"`
	TotalCount   int                       `json:"total_count"`
	Results      map[string]QuestionResult `json:"results"`
}

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
	QTypeFormCompletion          QuestionType = "form-completion"
	QTypeNoteCompletion          QuestionType = "note-completion"
	QTypeMatching                QuestionType = "matching"
	QTypeMapPlanLabelling        QuestionType = "map-plan-labelling"
)

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

var fillBlankQuestionTypes = map[QuestionType]bool{
	QTypeSentenceCompletion:     true,
	QTypeSummaryCompletion:      true,
	QTypeTableCompletion:        true,
	QTypeNoteCompletion:         true,
	QTypeFlowChartCompletion:    true,
	QTypeFormCompletion:         true,
	QTypeShortAnswer:            true,
	QTypeDiagramLabelCompletion: true,
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
