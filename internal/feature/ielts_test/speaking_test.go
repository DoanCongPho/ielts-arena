package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
)

func TestIeltsOverall(t *testing.T) {
	cases := []struct {
		bands []float64
		want  float64
	}{
		{[]float64{6, 6, 6, 7}, 6.5}, // 6.25 rounds up to 6.5
		{[]float64{6, 7, 7, 7}, 7},   // 6.75 rounds up to 7
		{[]float64{6, 6, 7, 7}, 6.5}, // 6.5 stays
		{[]float64{6, 6, 6, 6}, 6},   // whole stays
		{[]float64{6, 6.5, 6, 6}, 6}, // 6.125 rounds down
		{[]float64{8, 9, 9, 9}, 9},   // 8.75 rounds up to 9
		{[]float64{4, 5, 5, 5}, 5},   // 4.75
		{[]float64{5, 5, 5, 6}, 5.5}, // 5.25
	}
	for _, c := range cases {
		if got := ieltsOverall(c.bands); got != c.want {
			t.Errorf("ieltsOverall(%v) = %v, want %v", c.bands, got, c.want)
		}
	}
}

func fullSpeakingContent() SpeakingContent {
	c := SpeakingContent{
		Part1: &SpeakingPart1{Topics: []SpeakingTopic{
			{Topic: "Your hometown", Questions: []SpeakingQuestion{{Text: "Where is your hometown?"}, {Text: "Do you like living there?"}}},
			{Topic: "Music", Questions: []SpeakingQuestion{{Text: "What music do you like?"}}},
		}},
		Part2: &SpeakingPart2{
			Topic:     "Describe a book you enjoyed reading.",
			Bullets:   []string{"what the book was", "when you read it", "what it was about"},
			Explain:   "and explain why you enjoyed it.",
			FollowUps: []SpeakingQuestion{{Text: "Do you often read?"}},
		},
		Part3: &SpeakingPart3{Questions: []SpeakingQuestion{
			{Text: "Why do people read less today?"}, {Text: "Should schools make reading compulsory?"}, {Text: "Will printed books disappear?"},
		}},
	}
	normalizeSpeakingContent(&c)
	return c
}

func TestSpeakingContent_ModesAndValidation(t *testing.T) {
	c := fullSpeakingContent()
	if err := validateSpeakingContent(c); err != nil {
		t.Fatalf("full test rejected: %v", err)
	}
	if c.Mode() != SpeakingModeFull {
		t.Errorf("Mode() = %q, want full", c.Mode())
	}
	if got := c.Part1.Topics[1].Questions[0].ID; got != "p1.t2.q1" {
		t.Errorf("question id = %q, want p1.t2.q1", got)
	}

	twoParts := SpeakingContent{Part1: c.Part1, Part2: c.Part2}
	if err := validateSpeakingContent(twoParts); err == nil {
		t.Error("two-part content accepted; the real test has one or three")
	}

	part3Only := SpeakingContent{Part3: &SpeakingPart3{Questions: c.Part3.Questions}}
	if err := validateSpeakingContent(part3Only); err == nil {
		t.Error("Part 3 on its own without a theme accepted")
	}
	part3Only.Part3.Theme = "reading"
	if err := validateSpeakingContent(part3Only); err != nil {
		t.Errorf("Part 3 with a theme rejected: %v", err)
	}

	bad := fullSpeakingContent()
	bad.Part2.Bullets = bad.Part2.Bullets[:2]
	if err := validateSpeakingContent(bad); err == nil {
		t.Error("cue card with two bullet points accepted")
	}
}

func TestBuildSpeakingScript(t *testing.T) {
	lines := buildSpeakingScript(fullSpeakingContent(), defaultExaminerVoice)

	var kinds []string
	for _, l := range lines {
		kinds = append(kinds, l.Kind)
		if l.AudioKey == "" || !strings.HasPrefix(l.AudioKey, "examiner/") {
			t.Errorf("line %q has audio key %q", l.Text, l.AudioKey)
		}
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, "say,cue_card,long_turn,say,ask") {
		t.Errorf("Part 2 sequence wrong: %s", joined)
	}
	var leadIn, closing string
	for _, l := range lines {
		if l.Part == 3 && l.Kind == LineSay && leadIn == "" {
			leadIn = l.Text
		}
		closing = l.Text
	}
	if !strings.Contains(leadIn, "talking about a book you enjoyed reading") {
		t.Errorf("Part 3 lead-in = %q", leadIn)
	}
	if closing != "Thank you. That is the end of the speaking test." {
		t.Errorf("closing = %q", closing)
	}

	// Identical lines share one recording across tests.
	again := buildSpeakingScript(fullSpeakingContent(), defaultExaminerVoice)
	if again[0].AudioKey != lines[0].AudioKey {
		t.Error("audio key is not deterministic")
	}
	other := defaultExaminerVoice
	other.Voice = "alloy"
	if buildSpeakingScript(fullSpeakingContent(), other)[0].AudioKey == lines[0].AudioKey {
		t.Error("audio key ignores the voice")
	}
}

// words builds a timed word list: each word lasts 0.3 s, and a gap after a
// word is given in pauses (by word index).
func words(text string, pauses map[int]float64) []llm.TranscriptWord {
	var out []llm.TranscriptWord
	t := 0.0
	for i, w := range strings.Fields(text) {
		w = strings.Trim(w, ".,?!")
		out = append(out, llm.TranscriptWord{Word: w, Start: t, End: t + 0.3})
		t += 0.3 + 0.05 + pauses[i]
	}
	return out
}

func TestExtractEvidence(t *testing.T) {
	longTurn := "Well, the book I want to talk about is, um, a novel. I read it last year because a friend recommended it, and I I really enjoyed it."
	answers := []answerTranscript{
		{QuestionID: "p2", Part: 2, Text: longTurn, Duration: 60,
			// A 1.2 s pause mid-clause after "is" (index 8), and a 1.5 s
			// pause at the sentence end after "novel." (index 11).
			Words: words(longTurn, map[int]float64{8: 1.2, 11: 1.5})},
		{QuestionID: "p3.q1", Part: 3, Text: "Maybe.", Duration: 3, Words: words("Maybe.", nil)},
	}
	ev := extractEvidence(answers)
	f := ev.Fluency

	if f.WordCount != 28 {
		t.Errorf("WordCount = %d, want 28 (27 + \"Maybe\", filler excluded)", f.WordCount)
	}
	if f.FillersPer100Words == 0 {
		t.Error("the 'um' was not counted as a filler")
	}
	if f.Repairs != 1 {
		t.Errorf("Repairs = %d, want 1 (\"I I\")", f.Repairs)
	}
	if len(f.LongPauses) != 2 {
		t.Fatalf("LongPauses = %+v, want 2", f.LongPauses)
	}
	if !f.LongPauses[0].MidClause || f.LongPauses[1].MidClause {
		t.Errorf("mid-clause flags = %v, %v; want true, false", f.LongPauses[0].MidClause, f.LongPauses[1].MidClause)
	}
	if f.LongTurnSeconds < 10 || f.LongTurnSeconds > 13 {
		t.Errorf("LongTurnSeconds = %.1f", f.LongTurnSeconds)
	}
	if ev.Coherence.MinimalAnswers != 1 {
		t.Errorf("MinimalAnswers = %d, want 1", ev.Coherence.MinimalAnswers)
	}
	if ev.Coherence.DiscourseMarkers["because"] != 1 || ev.Coherence.DiscourseMarkers["well"] != 1 {
		t.Errorf("DiscourseMarkers = %v", ev.Coherence.DiscourseMarkers)
	}
	if ev.Grammar.ComplexSentenceShare == 0 {
		t.Error("the 'because' clause was not counted as complex")
	}
}

func TestSpeakingCaps(t *testing.T) {
	ev := SpeakingEvidence{Fluency: FluencyMetrics{WordCount: 400, SpeechRateWPM: 120, LongTurnSeconds: 40}}
	caps := speakingCaps(SpeakingModeFull, ev, &pronunciationSummary{Intelligibility: 0.95})
	if band, c := applyCaps(7, caps[CriterionFC]); band != 5 || c == nil {
		t.Errorf("a 40 s long turn left FC at %d", band)
	}
	if band, _ := applyCaps(8, caps[CriterionP]); band != 8 {
		t.Errorf("clear pronunciation capped to %d", band)
	}

	caps = speakingCaps(SpeakingModePart1, SpeakingEvidence{Fluency: FluencyMetrics{WordCount: 200, SpeechRateWPM: 140}}, estimatedPronunciation(SpeakingEvidence{}))
	if band, _ := applyCaps(9, caps[CriterionFC]); band != 7 {
		t.Errorf("Part 1 alone let FC reach %d", band)
	}
	if band, _ := applyCaps(9, caps[CriterionP]); band != 7 {
		t.Errorf("estimated pronunciation let P reach %d", band)
	}

	caps = speakingCaps(SpeakingModeFull, SpeakingEvidence{Fluency: FluencyMetrics{WordCount: 10}}, nil)
	for _, c := range speakingCriteria {
		if band, _ := applyCaps(6, caps[c]); band != 3 {
			t.Errorf("%s = %d with ten words spoken, want 3", c, band)
		}
	}
}

func TestSummarizePronunciation(t *testing.T) {
	s := summarizePronunciation([]clipAssessment{
		{QuestionID: "p2", Seconds: 90, A: &pronunciation.Assessment{
			Intelligibility: 0.9, Prosody: pronunciation.Prosody{PitchStdST: 3},
			Words: []pronunciation.Word{{Word: "think", Score: 40, Phonemes: []pronunciation.Phoneme{{Expected: "θ", Heard: "t", Score: 20}}}, {Word: "book", Score: 90}},
		}},
		{QuestionID: "p3.q1", Seconds: 30, A: &pronunciation.Assessment{
			Intelligibility: 0.5, Prosody: pronunciation.Prosody{PitchStdST: 1},
			Words: []pronunciation.Word{{Word: "Think", Score: 55, Phonemes: []pronunciation.Phoneme{{Expected: "θ", Heard: "t", Score: 30}}}, {Word: "uh", Score: 0}},
		}},
	})
	if s.Intelligibility != 0.8 { // (0.9*90 + 0.5*30) / 120
		t.Errorf("Intelligibility = %v, want 0.8", s.Intelligibility)
	}
	if s.Prosody.PitchStdST != 2.5 {
		t.Errorf("PitchStdST = %v, want 2.5", s.Prosody.PitchStdST)
	}
	if len(s.Mispronounced) != 1 || s.Mispronounced[0].Count != 2 || s.Mispronounced[0].Expected != "θ" {
		t.Errorf("Mispronounced = %+v", s.Mispronounced)
	}
	if got := s.Words["p2"]; len(got) != 1 || got[0].Index != 0 || got[0].Expected != "θ" {
		t.Errorf("Words[p2] = %+v (book has no phonemes, so only think is kept)", got)
	}
}

// --- speaking grading through the service, with every dependency faked ---

type fakeAudio map[string][]byte

func (f fakeAudio) Get(_ context.Context, key string) ([]byte, error) {
	b, ok := f[key]
	if !ok {
		return nil, errors.New("no such recording")
	}
	return b, nil
}

// fakeASR "transcribes" a recording by reading its bytes as the text.
type fakeASR struct{}

func (fakeASR) Transcribe(_ context.Context, _ string, audio []byte, _, _ string) (*llm.Transcript, error) {
	w := words(string(audio), nil)
	return &llm.Transcript{Text: string(audio), Duration: w[len(w)-1].End + 0.5, Words: w,
		Segments: []llm.TranscriptSegment{{Start: 0, End: w[len(w)-1].End, AvgLogprob: -0.2}}}, nil
}

type fakePron struct {
	err   error
	calls int
}

func (f *fakePron) Assess(context.Context, []byte, string, string, any) (*pronunciation.Assessment, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &pronunciation.Assessment{Intelligibility: 0.95, GOPMean: 85, Prosody: pronunciation.Prosody{PitchStdST: 3}}, nil
}

// fakeJudge gives every criterion the same band and records prompts. The
// four criteria are judged concurrently, hence the lock.
type fakeJudge struct {
	band    int
	mu      sync.Mutex
	prompts []string
}

func (f *fakeJudge) Complete(_ context.Context, system, user, _ string, _ llm.CompletionParams) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prompts = append(f.prompts, system+"\n"+user)
	return fmt.Sprintf(`{"checks":[{"band":%d,"feature":"x","verdict":"met","evidence":"y"}],"band":%d,"feedback":"ok","improvements":["z"]}`, f.band, f.band), nil
}

func speakingFixture(t *testing.T, pron *fakePron, judge *fakeJudge) (Service, *MockTestRepository, *Test, fakeAudio) {
	t.Helper()
	repo := NewMockTestRepository()
	content := fullSpeakingContent()
	test, _ := repo.CreateTest(context.Background(), &Test{Skill: "speaking", TaskType: "full", ContentData: mustMarshal(t, content)})

	sentence := "I think it is a really good question and I would say that it depends on the situation because people differ"
	audio := fakeAudio{}
	for _, q := range content.questions() {
		audio["speaking/1/"+q.ID+".webm"] = []byte(sentence + " " + sentence)
	}
	var assessor pronunciationAssessor
	if pron != nil {
		assessor = pron
	}
	grader := NewSpeakingGrader(audio, fakeASR{}, assessor, judge, SpeakingGraderConfig{JudgeModel: "judge"})
	svc := NewService(repo, &fakeGrader{}, &fakeXPGrantRepository{}, WithSpeakingGrader(grader, 0))
	return svc, repo, test, audio
}

func speakingSubmission(t *testing.T, test *Test, userID uint64) SubmitRequest {
	t.Helper()
	var c SpeakingContent
	_ = json.Unmarshal(test.ContentData, &c)
	var p SpeakingPayload
	for _, q := range c.questions() {
		p.Answers = append(p.Answers, SpeakingAnswer{QuestionID: q.ID, AudioKey: fmt.Sprintf("speaking/%d/%s.webm", userID, q.ID), DurationSec: 20})
	}
	return SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, p)}
}

func TestSpeaking_GradedAgainstDescriptors(t *testing.T) {
	judge := &fakeJudge{band: 7}
	svc, repo, test, _ := speakingFixture(t, &fakePron{}, judge)

	sub := submitAndGrade(t, svc, repo, 1, speakingSubmission(t, test, 1))
	if sub.Status != StatusGraded {
		t.Fatalf("status = %q (last error %q)", sub.Status, sub.LastError)
	}
	score, _ := repo.GetScoreBySubmissionID(context.Background(), sub.ID)
	var d SpeakingDetails
	if err := json.Unmarshal(score.Details, &d); err != nil {
		t.Fatal(err)
	}
	if len(judge.prompts) != 4 {
		t.Errorf("judge called %d times, want one per criterion", len(judge.prompts))
	}
	for _, p := range judge.prompts {
		if !strings.Contains(p, "=== Part 3 ===") || !strings.Contains(p, "Band 7:") {
			t.Error("a judge prompt is missing the whole-test transcript or the descriptors")
		}
	}
	if d.Indicative || d.Mode != SpeakingModeFull {
		t.Errorf("mode = %q, indicative = %v", d.Mode, d.Indicative)
	}
	if d.Pronunciation == nil || d.Pronunciation.Estimated {
		t.Errorf("pronunciation = %+v, want measured", d.Pronunciation)
	}
	// The fake long turn is ~7 s, so the guardrail lowers FC to 5 and the
	// overall is (5+7+7+7)/4 = 6.5.
	if fc := d.Criteria[CriterionFC]; fc.Score != 5 || fc.JudgedBand != 7 || fc.Cap == nil {
		t.Errorf("FC = %+v, want capped from 7 to 5", fc)
	}
	if *score.OverallBand != 6.5 {
		t.Errorf("overall = %v, want 6.5", *score.OverallBand)
	}
}

func TestSpeaking_PronunciationOutageRetriesThenEstimates(t *testing.T) {
	pron := &fakePron{err: errors.New("pronunciation service unreachable")}
	svc, repo, test, _ := speakingFixture(t, pron, &fakeJudge{band: 6})
	ctx := context.Background()

	sub, err := svc.SubmitAnswer(ctx, 1, speakingSubmission(t, test, 1))
	if err != nil {
		t.Fatal(err)
	}
	// Early attempts are rescheduled for a retry, not settled.
	if _, err := svc.GradeNextPending(ctx); err == nil {
		t.Fatal("first attempt succeeded without the pronunciation service")
	}
	if got, _ := repo.GetSubmissionByID(ctx, sub.ID); got.Status != StatusPending {
		t.Fatalf("status after a failed attempt = %q, want pending", got.Status)
	}

	// On the last attempt the grade goes through with an estimate.
	got, _ := repo.GetSubmissionByID(ctx, sub.ID)
	got.Attempts = maxGradingAttempts - 1
	got.NextAttemptAt = nil
	if _, err := svc.GradeNextPending(ctx); err != nil {
		t.Fatalf("last attempt: %v", err)
	}
	score, err := repo.GetScoreBySubmissionID(ctx, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	var d SpeakingDetails
	_ = json.Unmarshal(score.Details, &d)
	if d.Pronunciation == nil || !d.Pronunciation.Estimated {
		t.Errorf("pronunciation = %+v, want estimated", d.Pronunciation)
	}
}

func TestSpeaking_SubmissionValidation(t *testing.T) {
	svc, _, test, _ := speakingFixture(t, &fakePron{}, &fakeJudge{band: 6})
	ctx := context.Background()

	// User 2 submits user 1's recordings.
	req := speakingSubmission(t, test, 1)
	if _, err := svc.SubmitAnswer(ctx, 2, req); !errors.Is(err, ErrInvalidSubmission) {
		t.Errorf("someone else's recordings: err = %v, want ErrInvalidSubmission", err)
	}

	bad := SpeakingPayload{Answers: []SpeakingAnswer{{QuestionID: "p9", AudioKey: "speaking/1/x.webm", DurationSec: 5}}}
	if _, err := svc.SubmitAnswer(ctx, 1, SubmitRequest{TestID: test.ID, Payload: mustMarshal(t, bad)}); !errors.Is(err, ErrInvalidSubmission) {
		t.Errorf("unknown question: err = %v, want ErrInvalidSubmission", err)
	}
}

func TestSpeaking_PartPracticeIsIndicative(t *testing.T) {
	judge := &fakeJudge{band: 6}
	svc, repo, _, audio := speakingFixture(t, &fakePron{}, judge)
	part2 := fullSpeakingContent()
	part2.Part1, part2.Part3 = nil, nil
	test, _ := repo.CreateTest(context.Background(), &Test{Skill: "speaking", TaskType: "part2", ContentData: mustMarshal(t, part2)})
	_ = audio

	sub := submitAndGrade(t, svc, repo, 1, speakingSubmission(t, test, 1))
	score, _ := repo.GetScoreBySubmissionID(context.Background(), sub.ID)
	var d SpeakingDetails
	_ = json.Unmarshal(score.Details, &d)
	if !d.Indicative || d.Mode != SpeakingModePart2 {
		t.Errorf("mode = %q, indicative = %v; want part2, true", d.Mode, d.Indicative)
	}
	if !strings.Contains(judge.prompts[0], "practised on its own") {
		t.Error("the judge wasn't told this is part practice")
	}
}

func TestCustomTestsAreOwnerOnly(t *testing.T) {
	repo := NewMockTestRepository()
	svc := newTestService(repo, &fakeGrader{})
	ctx := context.Background()
	custom, _ := repo.CreateTest(ctx, &Test{OwnerID: 7, Skill: "speaking", TaskType: "part2", ContentData: mustMarshal(t, fullSpeakingContent())})

	if _, err := svc.GetTest(ctx, 7, custom.ID); err != nil {
		t.Errorf("owner can't open their test: %v", err)
	}
	if _, err := svc.GetTest(ctx, 8, custom.ID); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user opened it: err = %v", err)
	}
	if _, err := svc.SubmitAnswer(ctx, 8, SubmitRequest{TestID: custom.ID, Payload: []byte(`{}`)}); !errors.Is(err, ErrTestNotFound) {
		t.Errorf("another user submitted to it: err = %v", err)
	}
	list, _ := svc.GetListTest(ctx, "speaking", ListTestRequest{Page: 1})
	if len(list.Data) != 0 {
		t.Errorf("custom test listed with the official ones: %+v", list.Data)
	}
}

func TestAnchoredCorrections(t *testing.T) {
	answers := []answerTranscript{
		{QuestionID: "p2", Text: "He go to the school of magic. He have two friend."},
		{QuestionID: "p3.q1", Text: "Because people use phone."},
	}
	grammar := []SpeakingCorrection{
		{QuestionID: "p2", Original: "He go", Correction: "He goes"},
		{QuestionID: "p2", Original: "two friend", Correction: "two friends"},
		{QuestionID: "p2", Original: "She were happy", Correction: "She was happy"}, // not said
		{QuestionID: "p3.q1", Original: "He go", Correction: "He goes"},             // wrong answer
		{QuestionID: "p2", Original: "he go", Correction: "he goes"},                // duplicate
		{QuestionID: "p2", Original: "magic", Correction: "magic"},                  // no change
	}
	vocab := []SpeakingCorrection{
		{QuestionID: "p3.q1", Original: "use phone", Correction: "use their phones"},
		{QuestionID: "p2", Original: "two friend", Correction: "a couple of friends"}, // already flagged
	}
	got := anchoredCorrections(answers, grammar, "grammar", vocab, "vocabulary")
	if len(got) != 3 {
		t.Fatalf("got %d corrections, want 3: %+v", len(got), got)
	}
	if got[0].Type != "grammar" || got[2].Type != "vocabulary" || got[2].Correction != "use their phones" {
		t.Errorf("corrections = %+v", got)
	}
}

func TestJudgePrompts_AskForCorrectionsFromGrammarAndVocabularyOnly(t *testing.T) {
	for _, c := range speakingCriteria {
		p := judgeSystemPrompt(c, SpeakingModeFull)
		want := c == CriterionGRA || c == CriterionLR
		if got := strings.Contains(p, `"corrections"`); got != want {
			t.Errorf("%s prompt asks for corrections = %v, want %v", c, got, want)
		}
	}
}
