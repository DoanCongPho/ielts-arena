package grading

// Details is Score.Details for a speaking submission.
type Details struct {
	Mode string `json:"mode"`
	// Indicative is set for part practice: IELTS never scores one part
	// alone, so the band is a guide, not a test result.
	Indicative bool                 `json:"indicative"`
	Criteria   map[string]Criterion `json:"criteria"`
	// Flags lists answers the judges set aside as rehearsed or off-topic.
	Flags         Flags                 `json:"flags"`
	Evidence      Evidence              `json:"evidence"`
	Pronunciation *pronunciationSummary `json:"pronunciation"`
	Answers       []answerTranscript    `json:"answers"`
	// Corrections are the grammar and word-choice errors in the answers,
	// each anchored to text that appears verbatim in its answer.
	Corrections []Correction `json:"corrections"`
}

// Correction is one error in an answer and how to fix it.
type Correction struct {
	QuestionID  string `json:"question_id"`
	Type        string `json:"type"` // grammar | vocabulary
	Original    string `json:"original"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
}

type Criterion struct {
	Score        float64           `json:"score"`
	JudgedBand   int               `json:"judged_band"`
	Cap          *bandCap          `json:"cap,omitempty"`
	Feedback     string            `json:"feedback"`
	Improvements []string          `json:"improvements"`
	Checks       []descriptorCheck `json:"checks"`
}

type Flags struct {
	Rehearsed []string `json:"rehearsed,omitempty"`
	OffTopic  []string `json:"off_topic,omitempty"`
}
