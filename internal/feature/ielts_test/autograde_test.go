package ielts_test

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func sentenceCompletionGroup(order int, questionOrder int, answer string, accepted []string) QuestionGroup {
	return QuestionGroup{
		GroupOrder:   order,
		QuestionType: QTypeSentenceCompletion,
		Instructions: "Complete the sentences.",
		Questions: []Question{
			{QuestionOrder: questionOrder, Answer: rawAnswer(answer), AcceptedAnswers: accepted},
		},
	}
}

// rawAnswer builds an AnswerValue for a scalar string without needing *testing.T.
func rawAnswer(s string) AnswerValue {
	var a AnswerValue
	b, _ := json.Marshal(s)
	_ = a.UnmarshalJSON(b)
	return a
}

func rawAnswerArray(vals []string) AnswerValue {
	var a AnswerValue
	b, _ := json.Marshal(vals)
	_ = a.UnmarshalJSON(b)
	return a
}

func validReadingContent() ReadingContent {
	return ReadingContent{
		Passages: []ReadingPassage{
			{
				Title:      "Passage 1",
				Paragraphs: []Paragraph{{Label: "A", Text: "Some text."}},
				QuestionGroups: []QuestionGroup{
					sentenceCompletionGroup(1, 1, "cat", []string{"cat", "cats"}),
					{
						GroupOrder:   2,
						QuestionType: QTypeMultipleChoice,
						Instructions: "Choose the best answer.",
						SharedOptions: []Option{
							{ID: "A", Text: "opt a"},
							{ID: "B", Text: "opt b"},
						},
						Questions: []Question{
							{QuestionOrder: 2, Answer: rawAnswer("A")},
						},
					},
					{
						GroupOrder:   3,
						QuestionType: QTypeMultipleChoiceMulti,
						Instructions: "Choose two answers.",
						SelectCount:  2,
						SharedOptions: []Option{
							{ID: "A", Text: "opt a"},
							{ID: "B", Text: "opt b"},
							{ID: "C", Text: "opt c"},
						},
						Questions: []Question{
							{QuestionOrder: 3, Answer: rawAnswerArray([]string{"A", "B"})},
						},
					},
				},
			},
		},
	}
}

func marshalContent(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	return b
}

// ---------------------------------------------------------------------------
// validateContentData
// ---------------------------------------------------------------------------

func TestValidateContentData_Writing(t *testing.T) {
	if err := validateContentData("writing", []byte(`{"prompt":"Describe the chart."}`)); err != nil {
		t.Errorf("valid writing content rejected: %v", err)
	}
	if err := validateContentData("writing", []byte(`{"prompt":""}`)); err == nil {
		t.Error("expected error for missing prompt")
	}
	if err := validateContentData("writing", []byte(`not json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidateContentData_Speaking(t *testing.T) {
	if err := validateContentData("speaking", []byte(`{"prompt":"Talk about your hometown.","part":1}`)); err != nil {
		t.Errorf("valid speaking content rejected: %v", err)
	}
	if err := validateContentData("speaking", []byte(`{"prompt":""}`)); err == nil {
		t.Error("expected error for missing prompt")
	}
}

func TestValidateContentData_Reading(t *testing.T) {
	if err := validateContentData("reading", marshalContent(t, validReadingContent())); err != nil {
		t.Errorf("valid reading content rejected: %v", err)
	}

	t.Run("no passages", func(t *testing.T) {
		if err := validateContentData("reading", marshalContent(t, ReadingContent{})); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("passage without paragraphs", func(t *testing.T) {
		c := ReadingContent{Passages: []ReadingPassage{{QuestionGroups: []QuestionGroup{sentenceCompletionGroup(1, 1, "x", []string{"x"})}}}}
		if err := validateContentData("reading", marshalContent(t, c)); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("question type not allowed for reading", func(t *testing.T) {
		c := validReadingContent()
		c.Passages[0].QuestionGroups[0].QuestionType = QTypeFormCompletion // listening-only
		if err := validateContentData("reading", marshalContent(t, c)); err == nil {
			t.Error("expected error for form-completion in reading")
		}
	})

	t.Run("duplicate question_order", func(t *testing.T) {
		c := validReadingContent()
		c.Passages[0].QuestionGroups[1].Questions[0].QuestionOrder = 1 // collides with group 1's question
		if err := validateContentData("reading", marshalContent(t, c)); err == nil {
			t.Error("expected error for duplicate question_order")
		}
	})

	t.Run("non-contiguous question_order", func(t *testing.T) {
		c := validReadingContent()
		c.Passages[0].QuestionGroups[1].Questions[0].QuestionOrder = 5
		if err := validateContentData("reading", marshalContent(t, c)); err == nil {
			t.Error("expected error for gap in question_order")
		}
	})

	t.Run("fill-blank question missing accepted_answers", func(t *testing.T) {
		c := validReadingContent()
		c.Passages[0].QuestionGroups[0].Questions[0].AcceptedAnswers = nil
		if err := validateContentData("reading", marshalContent(t, c)); err == nil {
			t.Error("expected error for missing accepted_answers")
		}
	})

	t.Run("accepted_answers does not include the answer", func(t *testing.T) {
		c := validReadingContent()
		c.Passages[0].QuestionGroups[0].Questions[0].AcceptedAnswers = []string{"dog"}
		if err := validateContentData("reading", marshalContent(t, c)); err == nil {
			t.Error("expected error when accepted_answers omits the answer value")
		}
	})
}

func TestValidateContentData_Listening(t *testing.T) {
	valid := ListeningContent{
		AudioURL: "https://example.com/audio.mp3",
		Sections: []ListeningSection{
			{
				SectionStartTime: 0,
				SectionEndTime:   60,
				QuestionGroups: []QuestionGroup{
					sentenceCompletionGroup(1, 1, "cat", []string{"cat"}),
				},
			},
		},
	}
	if err := validateContentData("listening", marshalContent(t, valid)); err != nil {
		t.Errorf("valid listening content rejected: %v", err)
	}

	t.Run("missing audio_url", func(t *testing.T) {
		c := valid
		c.AudioURL = ""
		if err := validateContentData("listening", marshalContent(t, c)); err == nil {
			t.Error("expected error for missing audio_url")
		}
	})

	t.Run("section end before start", func(t *testing.T) {
		c := valid
		c.Sections = []ListeningSection{{SectionStartTime: 60, SectionEndTime: 10, QuestionGroups: valid.Sections[0].QuestionGroups}}
		if err := validateContentData("listening", marshalContent(t, c)); err == nil {
			t.Error("expected error for end <= start")
		}
	})
}

// ---------------------------------------------------------------------------
// validateGroupShape (unexported, tested directly with struct literals)
// ---------------------------------------------------------------------------

func TestValidateGroupShape(t *testing.T) {
	cases := []struct {
		name    string
		group   QuestionGroup
		wantErr bool
	}{
		{
			name: "multiple-choice-multi with select_count < 2",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMultipleChoiceMulti, SelectCount: 1,
				SharedOptions: []Option{{ID: "A"}, {ID: "B"}},
				Questions:     []Question{{QuestionOrder: 1, Answer: rawAnswerArray([]string{"A"})}},
			},
			wantErr: true,
		},
		{
			name: "multiple-choice-multi answer count mismatch",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMultipleChoiceMulti, SelectCount: 2,
				SharedOptions: []Option{{ID: "A"}, {ID: "B"}, {ID: "C"}},
				Questions:     []Question{{QuestionOrder: 1, Answer: rawAnswerArray([]string{"A"})}},
			},
			wantErr: true,
		},
		{
			name: "matching-headings without shared_options",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMatchingHeadings,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("i")}},
			},
			wantErr: true,
		},
		{
			name: "matching-sentence-endings without options anywhere",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMatchingSentenceEndings,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("A")}},
			},
			wantErr: true,
		},
		{
			name: "matching-sentence-endings with per-question options",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMatchingSentenceEndings,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("A"), Options: []Option{{ID: "A"}}}},
			},
			wantErr: false,
		},
		{
			name: "map-plan-labelling missing map_image_url",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMapPlanLabelling,
				LocationKey: []Option{{ID: "A"}},
				Questions:   []Question{{QuestionOrder: 1, Answer: rawAnswer("A")}},
			},
			wantErr: true,
		},
		{
			name: "map-plan-labelling missing location_key",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMapPlanLabelling,
				MapImageURL: "https://example.com/map.png",
				Questions:   []Question{{QuestionOrder: 1, Answer: rawAnswer("A")}},
			},
			wantErr: true,
		},
		{
			name: "map-plan-labelling valid",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeMapPlanLabelling,
				MapImageURL: "https://example.com/map.png",
				LocationKey: []Option{{ID: "A"}},
				Questions:   []Question{{QuestionOrder: 1, Answer: rawAnswer("A")}},
			},
			wantErr: false,
		},
		{
			name: "summary-completion missing summary_text",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeSummaryCompletion,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("x"), AcceptedAnswers: []string{"x"}}},
			},
			wantErr: true,
		},
		{
			name: "summary-completion has_word_bank true but empty word_bank",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeSummaryCompletion,
				SummaryText: "A {{gap}} sat on the mat.", HasWordBank: true,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("cat")}},
			},
			wantErr: true,
		},
		{
			name: "summary-completion gap count mismatch",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeSummaryCompletion,
				SummaryText: "A {{gap}} sat on the {{gap}}.",
				Questions:   []Question{{QuestionOrder: 1, Answer: rawAnswer("cat"), AcceptedAnswers: []string{"cat"}}},
			},
			wantErr: true,
		},
		{
			name: "table-completion missing table_structure",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeTableCompletion,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("x"), AcceptedAnswers: []string{"x"}}},
			},
			wantErr: true,
		},
		{
			name: "table-completion gap count mismatch",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeTableCompletion,
				TableStructure: &TableStructure{Columns: []string{"c"}, Rows: [][]string{{"{{gap}}"}, {"{{gap}}"}}},
				Questions:      []Question{{QuestionOrder: 1, Answer: rawAnswer("x"), AcceptedAnswers: []string{"x"}}},
			},
			wantErr: true,
		},
		{
			name: "diagram-label-completion missing diagram_image_url",
			group: QuestionGroup{
				GroupOrder: 1, QuestionType: QTypeDiagramLabelCompletion,
				Questions: []Question{{QuestionOrder: 1, Answer: rawAnswer("x"), AcceptedAnswers: []string{"x"}}},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGroupShape(tc.group)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateGroupShape() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// validateQuestionOrderContinuity / containsFold / countGaps / requireGapCount
// ---------------------------------------------------------------------------

func TestValidateQuestionOrderContinuity(t *testing.T) {
	cases := []struct {
		name    string
		orders  []int
		wantErr bool
	}{
		{"empty", nil, false},
		{"contiguous", []int{1, 2, 3}, false},
		{"contiguous out of order", []int{3, 1, 2}, false},
		{"missing middle", []int{1, 3}, true},
		{"duplicate", []int{1, 1, 2}, true},
		{"non-positive", []int{0, 1}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateQuestionOrderContinuity(tc.orders)
			if (err != nil) != tc.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestContainsFold(t *testing.T) {
	if !containsFold([]string{"Cat", "dog"}, []string{"CAT"}) {
		t.Error("expected case-insensitive match to succeed")
	}
	if containsFold([]string{"dog"}, []string{"cat"}) {
		t.Error("expected no match")
	}
	if containsFold([]string{"dog"}, nil) {
		t.Error("expected no match against empty answer")
	}
}

func TestCountGapsAndRequireGapCount(t *testing.T) {
	if got := countGaps("A {{gap}} sat on the {{gap}}."); got != 2 {
		t.Errorf("countGaps() = %d, want 2", got)
	}
	if countGaps("no gaps here") != 0 {
		t.Error("expected 0 gaps")
	}
	if err := requireGapCount(1, "field", 2, 2); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := requireGapCount(1, "field", 1, 2); err == nil {
		t.Error("expected error on mismatch")
	}
}

// ---------------------------------------------------------------------------
// answersMatch / sameSetNormalized / normalizeAnswer / decodeAnswerStrings
// ---------------------------------------------------------------------------

func TestAnswersMatch_MultipleChoiceMulti(t *testing.T) {
	q := gradableQuestion{
		Question:  Question{Answer: rawAnswerArray([]string{"A", "B"})},
		GroupType: QTypeMultipleChoiceMulti,
	}
	if !answersMatch(q, []string{"b", "a"}) {
		t.Error("expected order/case-independent match to succeed")
	}
	if answersMatch(q, []string{"A"}) {
		t.Error("expected partial answer to fail")
	}
}

func TestAnswersMatch_FillBlank(t *testing.T) {
	q := gradableQuestion{
		Question:  Question{AcceptedAnswers: []string{"cat", "cats"}},
		GroupType: QTypeSentenceCompletion,
	}
	if !answersMatch(q, []string{" Cats "}) {
		t.Error("expected trimmed case-insensitive match against accepted_answers")
	}
	if answersMatch(q, []string{"dog"}) {
		t.Error("expected mismatch to fail")
	}
	if answersMatch(q, nil) {
		t.Error("expected empty submission to fail")
	}
}

func TestAnswersMatch_SummaryCompletionWithWordBank(t *testing.T) {
	// HasWordBank routes summary-completion to the exact-match default path
	// instead of the accepted_answers fill-blank rule.
	q := gradableQuestion{
		Question:    Question{Answer: rawAnswer("A")},
		GroupType:   QTypeSummaryCompletion,
		HasWordBank: true,
	}
	if !answersMatch(q, []string{"a"}) {
		t.Error("expected case-insensitive exact match")
	}
}

func TestAnswersMatch_Default(t *testing.T) {
	q := gradableQuestion{
		Question:  Question{Answer: rawAnswer("True")},
		GroupType: QTypeTrueFalseNotGiven,
	}
	if !answersMatch(q, []string{"true"}) {
		t.Error("expected case-insensitive exact match")
	}
	if answersMatch(q, nil) {
		t.Error("expected empty submission to fail")
	}
}

func TestSameSetNormalized(t *testing.T) {
	if !sameSetNormalized([]string{"A", "b"}, []string{"B", "a"}) {
		t.Error("expected case/order-independent set match")
	}
	if sameSetNormalized([]string{"A"}, []string{"A", "B"}) {
		t.Error("expected size mismatch to fail")
	}
	if sameSetNormalized(nil, nil) {
		t.Error("expected two empty sets to be treated as non-matching")
	}
}

func TestDecodeAnswerStrings(t *testing.T) {
	if got := decodeAnswerStrings(json.RawMessage(`"paris"`)); len(got) != 1 || got[0] != "paris" {
		t.Errorf("got %v, want [paris]", got)
	}
	if got := decodeAnswerStrings(json.RawMessage(`""`)); got != nil {
		t.Errorf("got %v, want nil for empty scalar", got)
	}
	if got := decodeAnswerStrings(json.RawMessage(`["A","B"]`)); len(got) != 2 {
		t.Errorf("got %v, want [A B]", got)
	}
	if got := decodeAnswerStrings(nil); got != nil {
		t.Errorf("got %v, want nil for empty input", got)
	}
}

// ---------------------------------------------------------------------------
// bandFromRawScore
// ---------------------------------------------------------------------------

func TestBandFromRawScore(t *testing.T) {
	cases := []struct {
		correct, total int
		want           float64
	}{
		{0, 0, 0},
		{40, 40, 9},
		{39, 40, 9},
		{35, 40, 8},
		{30, 40, 7},
		{20, 40, 5.5},
		{4, 40, 2.5},
		{0, 40, 2},
	}
	for _, tc := range cases {
		if got := bandFromRawScore(tc.correct, tc.total); got != tc.want {
			t.Errorf("bandFromRawScore(%d, %d) = %v, want %v", tc.correct, tc.total, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// questionsFromContent / publicContentData / redactQuestionsInPlace
// ---------------------------------------------------------------------------

func TestQuestionsFromContent(t *testing.T) {
	questions, err := questionsFromContent("reading", marshalContent(t, validReadingContent()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("got %d questions, want 3", len(questions))
	}
	if questions[2].GroupType != QTypeMultipleChoiceMulti {
		t.Errorf("expected third question's group type to be multiple-choice-multi, got %s", questions[2].GroupType)
	}

	if _, err := questionsFromContent("writing", []byte(`{}`)); err == nil {
		t.Error("expected error for non-auto-gradable skill")
	}
}

func TestPublicContentData_RedactsReadingAnswers(t *testing.T) {
	raw := marshalContent(t, validReadingContent())
	public, err := publicContentData("reading", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var content ReadingContent
	if err := json.Unmarshal(public, &content); err != nil {
		t.Fatalf("unmarshal redacted content: %v", err)
	}
	for _, p := range content.Passages {
		for _, g := range p.QuestionGroups {
			for _, q := range g.Questions {
				if !q.Answer.IsEmpty() {
					t.Errorf("expected answer to be redacted for question_order %d", q.QuestionOrder)
				}
				if q.AcceptedAnswers != nil {
					t.Errorf("expected accepted_answers to be redacted for question_order %d", q.QuestionOrder)
				}
			}
		}
	}
}

func TestPublicContentData_PassesThroughOtherSkills(t *testing.T) {
	raw := []byte(`{"prompt":"Describe the chart."}`)
	public, err := publicContentData("writing", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(public) != string(raw) {
		t.Errorf("got %s, want passthrough of %s", public, raw)
	}
}
