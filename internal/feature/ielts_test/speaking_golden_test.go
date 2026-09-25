package ielts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
)

// Golden tests pin speaking grading's output — the stored details JSON, the
// judge prompts, the measured evidence and the examiner script — so a
// refactor can prove it changed nothing. Regenerate deliberately with
// UPDATE_GOLDEN=1 go test ./...

const goldenDir = "testdata/speaking_golden"

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join(goldenDir, name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with UPDATE_GOLDEN=1 to create it)", path, err)
	}
	if string(want) != string(got) {
		t.Errorf("%s differs from the golden file; diff it against %s", name, path)
		_ = os.WriteFile(path+".got", got, 0o644)
	}
}

func goldenJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

// goldenAnswers are the candidate's answers by question id. "||" marks a
// 1.2 s pause and "|" a 0.5 s one; "um"/"uh" are fillers.
var goldenAnswers = map[string]string{
	"p1.t1.q1": "My hometown is Da Nang, a coastal city in the centre of Vietnam.",
	"p1.t1.q2": "Yes I like it very much, because the beaches is beautiful || and the people are friendly.",
	"p1.t2.q1": "I like pop music and, um, sometimes classical music when I study.",
	"p2": "I want to talk about a book called The Little Prince. I read it when I am young, | maybe ten years old. " +
		"It is about a little prince who travel between planets || and meet many strange adults. " +
		"I I really enjoyed it because it teach me that, uh, the most important things are invisible to the eyes. " +
		"On top of that, the drawings are lovely. All in all, it is a book I would recommend to anyone.",
	"p2.f1": "Not very often these days, maybe one book a month.",
	"p3.q1": "I think people read less because smartphones compete for our attention || and short videos is more fun.",
	"p3.q2": "Personally, I am not sure. If children are forced to read, | they may end up hating it.",
	"p3.q3": "No, I don't think so. Printed books will always have a place, | although e-books are growing.",
}

// goldenTranscriber turns an answer into timed words: 0.3 s per word, 0.05 s
// between words, plus the marked pauses.
type goldenTranscriber struct{}

func (goldenTranscriber) Transcribe(_ context.Context, _ string, audio []byte, _, _ string) (*llm.Transcript, error) {
	var words []llm.TranscriptWord
	var text []string
	at, pending := 0.0, 0.0
	for _, tok := range strings.Fields(string(audio)) {
		switch tok {
		case "||":
			pending = 1.2
			continue
		case "|":
			pending = 0.5
			continue
		}
		at += pending
		pending = 0
		words = append(words, llm.TranscriptWord{Word: strings.Trim(tok, ".,?!"), Start: at, End: at + 0.3})
		text = append(text, tok)
		at += 0.35
	}
	return &llm.Transcript{
		Text: strings.Join(text, " "), Duration: at + 0.5, Words: words,
		Segments: []llm.TranscriptSegment{{Text: strings.Join(text, " "), Start: 0, End: at, AvgLogprob: -0.35}},
	}, nil
}

// goldenPronunciation scores each word by its length, so some words come
// out mispronounced with a phoneme heard wrong.
type goldenPronunciation struct{}

func (goldenPronunciation) Assess(_ context.Context, _ []byte, _ string, _ string, words any) (*pronunciation.Assessment, error) {
	tw := words.([]llm.TranscriptWord)
	a := &pronunciation.Assessment{
		Intelligibility: 0.82, GOPMean: 78.5, PER: 0.21, AccentVariant: "en-gb",
		Prosody: pronunciation.Prosody{PitchRangeST: 5.4, PitchStdST: 2.2, StressMatchRate: 0.7, NPVIVowel: 52.3},
	}
	for i, w := range tw {
		score := 92.0
		phonemes := []pronunciation.Phoneme{{Expected: "ə", Heard: "ə", Score: 92}}
		if len(w.Word)%4 == 0 {
			score = 41.5
			phonemes = []pronunciation.Phoneme{{Expected: "θ", Heard: "t", Score: 20}, {Expected: "ɪ", Heard: "ɪ", Score: 63}}
		}
		if i%11 == 7 {
			phonemes = nil // could not be aligned
		}
		a.Words = append(a.Words, pronunciation.Word{Word: w.Word, Start: w.Start, End: w.End, Score: score, Heard: "t ɪ", Phonemes: phonemes})
	}
	return a, nil
}

// goldenJudge answers each criterion with a fixed band and, for grammar and
// vocabulary, corrections — some anchored in the transcript, some not.
type goldenJudge struct {
	mu      sync.Mutex
	prompts map[string]string
}

func (j *goldenJudge) Complete(_ context.Context, system, user, _ string, params llm.CompletionParams) (string, error) {
	bands := map[string]int{CriterionFC: 7, CriterionLR: 6, CriterionGRA: 6, CriterionP: 8}
	var criterion string
	for c := range bands {
		if strings.Contains(system, "one criterion only: "+c+".") {
			criterion = c
		}
	}
	j.mu.Lock()
	j.prompts[criterion] = fmt.Sprintf("model=%s temperature=%v max_tokens=%d\n--- system\n%s\n--- user\n%s", params.Model, params.Temperature, params.MaxTokens, system, user)
	j.mu.Unlock()

	corrections := "[]"
	switch criterion {
	case CriterionGRA:
		corrections = `[{"question_id":"p2","original":"when I am young","correction":"when I was young","explanation":"Past tense for the past."},
			{"question_id":"p2","original":"who travel between","correction":"who travels between","explanation":"Third person -s."},
			{"question_id":"p1.t1.q2","original":"the beaches is beautiful","correction":"the beaches are beautiful","explanation":"Plural subject."},
			{"question_id":"p2","original":"She were","correction":"She was","explanation":"Not said."}]`
	case CriterionLR:
		corrections = `[{"question_id":"p3.q1","original":"short videos is more fun","correction":"short videos are more entertaining","explanation":"Word choice."},
			{"question_id":"p2","original":"when I am young","correction":"as a child","explanation":"Duplicate of a grammar one."}]`
	}
	return fmt.Sprintf(`{"checks":[{"band":%[1]d,"feature":"key feature","verdict":"met","evidence":"quote"},{"band":%[2]d,"feature":"next band","verdict":"partly","evidence":"quote"}],
		"band":%[1]d,"feedback":"Feedback for %[3]s.","improvements":["One.","Two."],
		"rehearsed_question_ids":["p2.f1"],"off_topic_question_ids":[],"corrections":%[4]s}`,
		bands[criterion], bands[criterion]+1, criterion, corrections), nil
}

func goldenGradeInputs() (SpeakingContent, SpeakingPayload, fakeAudio) {
	content := fullSpeakingContent()
	audio := fakeAudio{}
	var payload SpeakingPayload
	for _, q := range content.questions() {
		key := "speaking/1/" + q.ID + ".webm"
		audio[key] = []byte(goldenAnswers[q.ID])
		payload.Answers = append(payload.Answers, SpeakingAnswer{QuestionID: q.ID, AudioKey: key, DurationSec: 20})
	}
	return content, payload, audio
}

func TestGolden_SpeakingGrade(t *testing.T) {
	content, payload, audio := goldenGradeInputs()
	judge := &goldenJudge{prompts: map[string]string{}}
	g := NewSpeakingGrader(audio, goldenTranscriber{}, goldenPronunciation{}, judge, SpeakingGraderConfig{JudgeModel: "judge-model"})

	details, overall, err := g.Grade(context.Background(), content, payload, false)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := json.Marshal(details) // exactly what gradeSpeaking stores
	if err != nil {
		t.Fatal(err)
	}
	checkGolden(t, "details.json", append(stored, '\n'))
	checkGolden(t, "overall.txt", []byte(fmt.Sprintf("%v\n", overall)))
	for _, c := range speakingCriteria {
		name := strings.ToLower(strings.NewReplacer(" ", "_").Replace(c))
		checkGolden(t, "prompt_"+name+".txt", []byte(judge.prompts[c]))
	}
}

func TestGolden_SpeakingGradeWithoutPronunciationService(t *testing.T) {
	content, payload, audio := goldenGradeInputs()
	content.Part1, content.Part3 = nil, nil
	var part2 SpeakingPayload
	for _, a := range payload.Answers {
		if strings.HasPrefix(a.QuestionID, "p2") {
			part2.Answers = append(part2.Answers, a)
		}
	}
	judge := &goldenJudge{prompts: map[string]string{}}
	g := NewSpeakingGrader(audio, goldenTranscriber{}, nil, judge, SpeakingGraderConfig{})
	details, overall, err := g.Grade(context.Background(), content, part2, false)
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := json.Marshal(details)
	checkGolden(t, "details_part2_estimated.json", append(stored, '\n'))
	checkGolden(t, "overall_part2_estimated.txt", []byte(fmt.Sprintf("%v\n", overall)))
	checkGolden(t, "prompt_part2_pronunciation.txt", []byte(judge.prompts[CriterionP]))
}

func TestGolden_SpeakingEvidence(t *testing.T) {
	content, payload, audio := goldenGradeInputs()
	qs := map[string]speakingQuestion{}
	for _, q := range content.questions() {
		qs[q.ID] = q
	}
	var answers []answerTranscript
	for _, a := range payload.Answers {
		tr, _ := goldenTranscriber{}.Transcribe(context.Background(), "", audio[a.AudioKey], "", "")
		answers = append(answers, answerTranscript{QuestionID: a.QuestionID, Part: qs[a.QuestionID].Part, Text: tr.Text, Duration: tr.Duration, Words: tr.Words, Segments: tr.Segments})
	}
	checkGolden(t, "evidence.json", goldenJSON(t, extractEvidence(answers)))
}

func TestGolden_SpeakingScript(t *testing.T) {
	type line struct {
		Kind, Text, QuestionID, AudioKey string
		Part, Seconds                    int
	}
	flatten := func(ls []ScriptLine) []line {
		out := make([]line, len(ls))
		for i, l := range ls {
			out[i] = line{l.Kind, l.Text, l.QuestionID, l.AudioKey, l.Part, l.Seconds}
		}
		return out
	}
	full := fullSpeakingContent()
	part2 := fullSpeakingContent()
	part2.Part1, part2.Part3 = nil, nil
	part3 := SpeakingContent{Part3: &SpeakingPart3{Theme: "Reading habits", Questions: full.Part3.Questions}}
	scripts := map[string][]line{
		"full":  flatten(buildSpeakingScript(full, defaultExaminerVoice)),
		"part2": flatten(buildSpeakingScript(part2, defaultExaminerVoice)),
		"part3": flatten(buildSpeakingScript(part3, NewExaminerVoice("tts-x", "alloy"))),
	}
	checkGolden(t, "script.json", goldenJSON(t, scripts))
}
