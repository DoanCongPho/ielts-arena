package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Validation — checked once at test-creation time (POST /api/tests), so bad
// content_data fails fast instead of surfacing later when a candidate takes
// or a submission gets graded.
// ---------------------------------------------------------------------------

// validateContentData checks that content_data matches the shape expected
// for skill before it's persisted.
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
		// The grader can only show the model an app asset or a public link.
		if u := c.ImageURL; u != "" && !strings.HasPrefix(u, "/assets/") &&
			!strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
			return errors.New("content_data.image_url must be an /assets/ path or an http(s) URL")
		}
	case "speaking":
		var c SpeakingContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("invalid content_data for speaking: %w", err)
		}
		NormalizeSpeakingContent(&c)
		if err := ValidateSpeakingContent(c); err != nil {
			return fmt.Errorf("invalid content_data for speaking: %w", err)
		}
	case "reading":
		var c ReadingContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("invalid content_data for reading: %w", err)
		}
		if len(c.Passages) == 0 {
			return errors.New("content_data.passages is required for reading")
		}
		var orders []int
		for _, p := range c.Passages {
			if len(p.Paragraphs) == 0 {
				return errors.New("each passage needs at least one paragraph")
			}
			if err := validateQuestionGroups(p.QuestionGroups, readingQuestionTypes, &orders); err != nil {
				return err
			}
			if err := validateEvidence(p.Paragraphs, p.QuestionGroups, "passage"); err != nil {
				return err
			}
		}
		if err := validateQuestionOrderContinuity(orders); err != nil {
			return err
		}
	case "listening":
		var c ListeningContent
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("invalid content_data for listening: %w", err)
		}
		if len(c.Sections) == 0 {
			return errors.New("content_data.sections is required for listening")
		}
		if c.AudioURL == "" {
			for _, sec := range c.Sections {
				if sec.AudioURL == "" {
					return errors.New("content_data.audio_url is required for listening unless every section has its own audio_url")
				}
			}
		}
		var orders []int
		for _, sec := range c.Sections {
			if sec.SectionEndTime <= sec.SectionStartTime {
				return errors.New("each section needs section_end_time greater than section_start_time")
			}
			if err := validateQuestionGroups(sec.QuestionGroups, listeningQuestionTypes, &orders); err != nil {
				return err
			}
			if err := validateEvidence(sec.Transcript, sec.QuestionGroups, "transcript"); err != nil {
				return err
			}
		}
		if err := validateQuestionOrderContinuity(orders); err != nil {
			return err
		}
	}
	return nil
}

// validateQuestionOrderContinuity requires every question_order in the test
// to form a contiguous set {1..N} with no duplicates or gaps — question
// numbering is global across the whole test, not reset per passage/section
// (docs/ielts-rl-data-structure.md). It does not require N == 40: the app
// also supports partial/single-passage practice tests, so only contiguity
// from 1 is enforced, not a fixed total.
func validateQuestionOrderContinuity(orders []int) error {
	if len(orders) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(orders))
	max := 0
	for _, o := range orders {
		if o <= 0 {
			return fmt.Errorf("question_order must be positive, got %d", o)
		}
		if seen[o] {
			return fmt.Errorf("duplicate question_order %d — question_order must be unique across the whole test", o)
		}
		seen[o] = true
		if o > max {
			max = o
		}
	}
	for i := 1; i <= max; i++ {
		if !seen[i] {
			return fmt.Errorf("question_order is not contiguous — missing %d (must run 1..%d with no gaps)", i, max)
		}
	}
	return nil
}

// validateQuestionGroups checks every question_group within one
// passage/section: each needs a group_order, a question_type valid for the
// skill (allowed), an instructions string, and at least one question. Each
// question needs a question_order (appended to *orders for the test-wide
// continuity check) and a non-empty answer. Per-type structural
// requirements (options, shared_options, word banks, {{gap}}-bearing
// structures, map/diagram assets) are validated by validateGroupShape.
func validateQuestionGroups(groups []QuestionGroup, allowed map[QuestionType]bool, orders *[]int) error {
	if len(groups) == 0 {
		return errors.New("each passage/section needs at least one question_group")
	}
	for _, g := range groups {
		if !allowed[g.QuestionType] {
			return fmt.Errorf("question_type %q is not valid for this skill", g.QuestionType)
		}
		if g.Instructions == "" {
			return errors.New("each question_group needs instructions")
		}
		if len(g.Questions) == 0 {
			return fmt.Errorf("question_group %d needs at least one question", g.GroupOrder)
		}
		if err := validateGroupShape(g); err != nil {
			return err
		}
		span := questionSpan(g)
		for _, q := range g.Questions {
			for i := range span {
				*orders = append(*orders, q.QuestionOrder+i)
			}
			if fillBlankQuestionTypes[g.QuestionType] && !(g.QuestionType == QTypeSummaryCompletion && g.HasWordBank) {
				if len(q.AcceptedAnswers) == 0 {
					return fmt.Errorf("question_order %d needs accepted_answers", q.QuestionOrder)
				}
				if !containsFold(q.AcceptedAnswers, q.Answer.Strings()) {
					return fmt.Errorf("question_order %d: accepted_answers must include the question's own answer value", q.QuestionOrder)
				}
			} else if q.Answer.IsEmpty() {
				return fmt.Errorf("question_order %d needs an answer", q.QuestionOrder)
			}
		}
	}
	return nil
}

// validateEvidence checks that every evidence entry in groups points at
// one of paragraphs (a reading passage, or a listening section's
// transcript — named by `source` in errors) and quotes it verbatim (modulo
// whitespace and quote style): the check that catches an importer, human
// or AI, citing text that isn't there.
func validateEvidence(paragraphs []Paragraph, groups []QuestionGroup, source string) error {
	for _, g := range groups {
		for _, q := range g.Questions {
			for _, e := range q.Evidence {
				if e.Paragraph < 0 || e.Paragraph >= len(paragraphs) {
					return fmt.Errorf("question_order %d: evidence paragraph %d is out of range (the %s has %d)", q.QuestionOrder, e.Paragraph, source, len(paragraphs))
				}
				quote := normalizeQuote(e.Quote)
				if quote == "" {
					return fmt.Errorf("question_order %d: evidence quote is empty", q.QuestionOrder)
				}
				if !strings.Contains(normalizeQuote(paragraphs[e.Paragraph].Text), quote) {
					return fmt.Errorf("question_order %d: evidence quote %q is not in paragraph %d", q.QuestionOrder, e.Quote, e.Paragraph)
				}
			}
		}
	}
	return nil
}

var quoteFolder = strings.NewReplacer("\u2018", "'", "\u2019", "'", "\u201c", `"`, "\u201d", `"`)

// normalizeQuote collapses whitespace and folds typographic quotes, so an
// evidence quote matches its paragraph however either was typed. The
// frontend mirrors this when it locates the quote to highlight it.
func normalizeQuote(s string) string {
	return strings.Join(strings.Fields(quoteFolder.Replace(s)), " ")
}

// questionSpan is how many question numbers — and marks — each question in
// g occupies. A multiple-choice-multi question covers select_count
// consecutive numbers ("Questions 23-24: choose TWO" is one question worth
// two marks, as in the real test), with question_order as the first of
// them. Every other question covers exactly one.
func questionSpan(g QuestionGroup) int {
	if g.QuestionType == QTypeMultipleChoiceMulti && g.SelectCount > 1 {
		return g.SelectCount
	}
	return 1
}

// containsFold reports whether accepted contains (case-insensitively) at
// least one of answer's values.
func containsFold(accepted []string, answer []string) bool {
	if len(answer) == 0 {
		return false
	}
	for _, a := range accepted {
		for _, want := range answer {
			if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(want)) {
				return true
			}
		}
	}
	return false
}

// validateGroupShape enforces the per-question_type structural fields
// described in docs/ielts-rl-data-structure.md.
func validateGroupShape(g QuestionGroup) error {
	switch g.QuestionType {
	case QTypeMultipleChoice, QTypeMultipleChoiceMulti:
		if len(g.SharedOptions) == 0 {
			for _, q := range g.Questions {
				if len(q.Options) == 0 {
					return fmt.Errorf("question_group %d (%s) needs options (shared or per-question)", g.GroupOrder, g.QuestionType)
				}
			}
		}
		if g.QuestionType == QTypeMultipleChoiceMulti {
			if g.SelectCount < 2 {
				return fmt.Errorf("question_group %d: multiple-choice-multi needs select_count >= 2", g.GroupOrder)
			}
			for _, q := range g.Questions {
				if len(q.Answer.Strings()) != g.SelectCount {
					return fmt.Errorf("question_order %d: answer must have exactly %d keys to match select_count", q.QuestionOrder, g.SelectCount)
				}
			}
		}
	case QTypeMatchingHeadings, QTypeMatchingInformation, QTypeMatchingFeatures, QTypeMatching:
		if len(g.SharedOptions) == 0 {
			return fmt.Errorf("question_group %d (%s) needs shared_options", g.GroupOrder, g.QuestionType)
		}
	case QTypeMatchingSentenceEndings:
		if len(g.SharedOptions) == 0 {
			for _, q := range g.Questions {
				if len(q.Options) == 0 {
					return fmt.Errorf("question_group %d (matching-sentence-endings) needs shared_options or per-question options", g.GroupOrder)
				}
			}
		}
	case QTypeMapPlanLabelling:
		// Explicitly flagged in the spec as a past bug: a missing map image
		// or location key makes the question unusable, so this must hard-fail
		// validation rather than degrade silently.
		if g.MapImageURL == "" {
			return fmt.Errorf("question_group %d (map-plan-labelling) needs map_image_url", g.GroupOrder)
		}
		if len(g.LocationKey) == 0 {
			return fmt.Errorf("question_group %d (map-plan-labelling) needs location_key", g.GroupOrder)
		}
	case QTypeSummaryCompletion:
		if g.SummaryText == "" {
			return fmt.Errorf("question_group %d (summary-completion) needs summary_text", g.GroupOrder)
		}
		if g.HasWordBank && len(g.WordBank) == 0 {
			return fmt.Errorf("question_group %d: has_word_bank is true but word_bank is empty", g.GroupOrder)
		}
		if !g.HasWordBank {
			if err := requireGapCount(g.GroupOrder, "summary_text", countGaps(g.SummaryText), len(g.Questions)); err != nil {
				return err
			}
		}
	case QTypeTableCompletion:
		if g.TableStructure == nil {
			return fmt.Errorf("question_group %d (table-completion) needs table_structure", g.GroupOrder)
		}
		count := 0
		for _, row := range g.TableStructure.Rows {
			for _, cell := range row {
				count += countGaps(cell)
			}
		}
		if err := requireGapCount(g.GroupOrder, "table_structure", count, len(g.Questions)); err != nil {
			return err
		}
	case QTypeNoteCompletion:
		if g.NoteStructure == nil {
			return fmt.Errorf("question_group %d (note-completion) needs note_structure", g.GroupOrder)
		}
		count := 0
		for _, item := range g.NoteStructure.Items {
			count += countGaps(item)
		}
		if err := requireGapCount(g.GroupOrder, "note_structure", count, len(g.Questions)); err != nil {
			return err
		}
	case QTypeFlowChartCompletion:
		if g.FlowStructure == nil {
			return fmt.Errorf("question_group %d (flow-chart-completion) needs flow_structure", g.GroupOrder)
		}
		count := 0
		for _, step := range g.FlowStructure.Steps {
			count += countGaps(step)
		}
		if err := requireGapCount(g.GroupOrder, "flow_structure", count, len(g.Questions)); err != nil {
			return err
		}
	case QTypeFormCompletion:
		if g.FormStructure == nil {
			return fmt.Errorf("question_group %d (form-completion) needs form_structure", g.GroupOrder)
		}
		count := 0
		for _, f := range g.FormStructure.Fields {
			count += countGaps(f)
		}
		if err := requireGapCount(g.GroupOrder, "form_structure", count, len(g.Questions)); err != nil {
			return err
		}
	case QTypeDiagramLabelCompletion:
		if g.DiagramImageURL == "" {
			return fmt.Errorf("question_group %d (diagram-label-completion) needs diagram_image_url", g.GroupOrder)
		}
	}
	return nil
}

const gapMarker = "{{gap}}"

func countGaps(s string) int {
	return strings.Count(s, gapMarker)
}

// requireGapCount enforces that a group's shared structure has exactly one
// "{{gap}}" per question — this keeps the frontend's positional gap-to-
// question mapping (i-th gap -> Questions[i]) correct.
func requireGapCount(groupOrder int, field string, gapCount, questionCount int) error {
	if gapCount != questionCount {
		return fmt.Errorf("question_group %d: %s has %d \"{{gap}}\" marker(s) but the group has %d question(s)", groupOrder, field, gapCount, questionCount)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Grading — auto-grades a reading/listening submission against the test's
// answer key, no LLM call needed (contrast with gradeSubmission in
// service.go, which handles the LLM-graded writing/speaking skills).
// ---------------------------------------------------------------------------

// gradableQuestion pairs a question with the group it belongs to, since
// the grading comparison rule (single-key exact match / accepted-answers
// membership / multi-select set match) depends on the group's
// question_type and, for summary-completion, whether it has a word bank.
type gradableQuestion struct {
	Question
	GroupType   QuestionType
	HasWordBank bool
	Span        int // see questionSpan
}

// autoGradeSubmission scores a reading/listening submission against the
// test's answer key. It persists the resulting score and updates the
// submission status in place, mirroring gradeSubmission.
func (s *service) autoGradeSubmission(ctx context.Context, test *Test, sub *Submission) error {
	questions, err := questionsFromContent(test.Skill, test.ContentData)
	if err != nil {
		return fmt.Errorf("%w: unmarshal test content: %v", ErrUngradable, err)
	}

	var payload AnswerPayload
	if err := json.Unmarshal(sub.Payload, &payload); err != nil {
		return fmt.Errorf("%w: unmarshal submission payload: %v", ErrUngradable, err)
	}

	results := make(map[string]QuestionResult, len(questions))
	correct, total := 0, 0
	for _, q := range questions {
		key := strconv.Itoa(q.QuestionOrder)
		submitted := decodeAnswerStrings(payload.Answers[key])
		points := answerPoints(q, submitted)
		correct += points
		total += q.Span
		correctAnswer := q.AcceptedAnswers
		if len(correctAnswer) == 0 {
			correctAnswer = q.Answer.Strings()
		}
		results[key] = QuestionResult{
			Correct:         points == q.Span,
			Points:          points,
			MaxPoints:       q.Span,
			SubmittedAnswer: submitted,
			CorrectAnswer:   correctAnswer,
		}
	}

	details, err := json.Marshal(AutoGradeDetails{
		CorrectCount: correct,
		TotalCount:   total,
		Results:      results,
	})
	if err != nil {
		return fmt.Errorf("marshal score details: %w", err)
	}

	overallBand := bandFromRawScore(correct, total)
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

func questionsFromContent(skill string, raw []byte) ([]gradableQuestion, error) {
	switch skill {
	case "reading":
		var content ReadingContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		var all []gradableQuestion
		for _, p := range content.Passages {
			for _, g := range p.QuestionGroups {
				for _, q := range g.Questions {
					all = append(all, gradableQuestion{Question: q, GroupType: g.QuestionType, HasWordBank: g.HasWordBank, Span: questionSpan(g)})
				}
			}
		}
		return all, nil
	case "listening":
		var content ListeningContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		var all []gradableQuestion
		for _, sec := range content.Sections {
			for _, g := range sec.QuestionGroups {
				for _, q := range g.Questions {
					all = append(all, gradableQuestion{Question: q, GroupType: g.QuestionType, HasWordBank: g.HasWordBank, Span: questionSpan(g)})
				}
			}
		}
		return all, nil
	default:
		return nil, fmt.Errorf("no auto-gradable questions for skill %q", skill)
	}
}

// answersMatch dispatches to one of three grading rules depending on the
// question's group type:
//   - multiple-choice-multi: order-independent, case-insensitive set match.
//   - fill-in-blank types (and summary-completion without a word bank):
//     correct if submitted matches ANY of the question's accepted_answers.
//   - everything else: exact case-insensitive/trimmed match against the
//     single canonical answer.
func answersMatch(q gradableQuestion, submitted []string) bool {
	switch {
	case q.GroupType == QTypeMultipleChoiceMulti:
		return sameSetNormalized(submitted, q.Answer.Strings())
	case fillBlankQuestionTypes[q.GroupType] && !(q.GroupType == QTypeSummaryCompletion && q.HasWordBank):
		if len(submitted) == 0 {
			return false
		}
		got := normalizeAnswer(submitted[0])
		for _, accepted := range q.AcceptedAnswers {
			if got == normalizeAnswer(accepted) {
				return true
			}
		}
		return false
	default:
		key := q.Answer.Strings()
		if len(submitted) == 0 || len(key) == 0 {
			return false
		}
		return normalizeAnswer(submitted[0]) == normalizeAnswer(key[0])
	}
}

// answerPoints is the marks a submission earns for q, out of q.Span. A
// multiple-choice-multi question scores one mark per correct key, in any
// order. Everything else is all-or-nothing via answersMatch.
func answerPoints(q gradableQuestion, submitted []string) int {
	if q.GroupType != QTypeMultipleChoiceMulti {
		if answersMatch(q, submitted) {
			return 1
		}
		return 0
	}
	key := make(map[string]bool)
	for _, k := range q.Answer.Strings() {
		key[normalizeAnswer(k)] = true
	}
	picked := make(map[string]bool)
	for _, s := range submitted {
		if n := normalizeAnswer(s); n != "" {
			picked[n] = true
		}
	}
	points := 0
	for k := range picked {
		if key[k] {
			points++
		}
	}
	return points
}

func sameSetNormalized(a, b []string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	na := make(map[string]bool, len(a))
	for _, x := range a {
		na[normalizeAnswer(x)] = true
	}
	nb := make(map[string]bool, len(b))
	for _, x := range b {
		nb[normalizeAnswer(x)] = true
	}
	if len(na) != len(nb) {
		return false
	}
	for k := range na {
		if !nb[k] {
			return false
		}
	}
	return true
}

func normalizeAnswer(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// decodeAnswerStrings decodes one submitted answer value (a JSON string or
// a JSON array of strings) into a uniform []string, mirroring
// AnswerValue.Strings().
func decodeAnswerStrings(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	return nil
}

// academicBandTable is the IELTS Listening / Academic Reading raw-score
// conversion (the two share one table): the minimum correct answers out of
// 40 for each band, highest first.
var academicBandTable = []struct {
	minCorrect int
	band       float64
}{
	{39, 9}, {37, 8.5}, {35, 8}, {33, 7.5}, {30, 7}, {27, 6.5}, {23, 6},
	{20, 5.5}, {16, 5}, {13, 4.5}, {10, 4}, {7, 3.5}, {5, 3}, {3, 2.5},
}

// bandFromRawScore maps a raw score to a band with academicBandTable. Tests
// that don't have exactly 40 marks (single-passage practice) are scaled to
// 40 first, rounding to the nearest mark. Below the table's last row (fewer
// than 3/40) it returns 2.
func bandFromRawScore(correct, total int) float64 {
	if total == 0 {
		return 0
	}
	outOf40 := int(math.Round(float64(correct) * 40 / float64(total)))
	for _, row := range academicBandTable {
		if outOf40 >= row.minCorrect {
			return row.band
		}
	}
	return 2
}

// ---------------------------------------------------------------------------
// Redaction — strips answers from reading/listening content before it's
// exposed to clients (GET /api/tests, GET /api/tests/{id}).
// ---------------------------------------------------------------------------

// publicContentData strips answers from reading/listening test content
// before it's exposed to clients. Other skills have nothing secret in
// their content, so it passes through unchanged.
func publicContentData(skill string, raw []byte) (json.RawMessage, error) {
	switch skill {
	case "reading":
		var content ReadingContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		for i := range content.Passages {
			redactQuestionsInPlace(content.Passages[i].QuestionGroups)
		}
		return json.Marshal(content)
	case "listening":
		var content ListeningContent
		if err := json.Unmarshal(raw, &content); err != nil {
			return nil, err
		}
		for i := range content.Sections {
			redactQuestionsInPlace(content.Sections[i].QuestionGroups)
			content.Sections[i].Transcript = nil
		}
		return json.Marshal(content)
	default:
		return json.RawMessage(raw), nil
	}
}

// redactQuestionsInPlace blanks Answer, AcceptedAnswers, Explanation and
// Evidence on every question across every group — each of the last three
// gives the answer away just as surely as Answer itself.
func redactQuestionsInPlace(groups []QuestionGroup) {
	for gi := range groups {
		for qi := range groups[gi].Questions {
			groups[gi].Questions[qi].Answer = AnswerValue{}
			groups[gi].Questions[qi].AcceptedAnswers = nil
			groups[gi].Questions[qi].Explanation = ""
			groups[gi].Questions[qi].Evidence = nil
		}
	}
}
