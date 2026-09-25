package ielts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"
	"sync"

	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
)

// Dependencies of speaking grading, declared at the point of use so tests
// can fake each one.
type (
	audioSource interface {
		Get(ctx context.Context, key string) ([]byte, error)
	}
	transcriber interface {
		Transcribe(ctx context.Context, model string, audio []byte, filename, prompt string) (*llm.Transcript, error)
	}
	pronunciationAssessor interface {
		Assess(ctx context.Context, audio []byte, filename, transcript string, words any) (*pronunciation.Assessment, error)
	}
	completer interface {
		Complete(ctx context.Context, system, user, imageURL string, params llm.CompletionParams) (string, error)
	}
)

// SpeakingGraderConfig picks the models. JudgeModel rates the descriptors;
// it is a judgement task, so a stronger model than the writing default
// pays for itself at four calls per test.
type SpeakingGraderConfig struct {
	WhisperModel string
	JudgeModel   string
	// PronunciationSampleSeconds is how much speech is sent to the
	// pronunciation service per test.
	PronunciationSampleSeconds float64
}

type SpeakingGrader struct {
	audio audioSource
	asr   transcriber
	pron  pronunciationAssessor // nil: always estimate
	judge completer
	cfg   SpeakingGraderConfig
}

func NewSpeakingGrader(audio audioSource, asr transcriber, pron pronunciationAssessor, judge completer, cfg SpeakingGraderConfig) *SpeakingGrader {
	if cfg.WhisperModel == "" {
		cfg.WhisperModel = "whisper-1"
	}
	if cfg.PronunciationSampleSeconds <= 0 {
		// Enough for a whole test (11-14 minutes), so every answer gets a
		// word-by-word comparison on the review page.
		cfg.PronunciationSampleSeconds = 900
	}
	return &SpeakingGrader{audio: audio, asr: asr, pron: pron, judge: judge, cfg: cfg}
}

// fillerPrompt biases Whisper toward a verbatim transcript. Without it,
// Whisper drops "um"s and restarts — exactly what fluency is rated on.
const fillerPrompt = "Umm, let me think, like, hmm... Okay, so, uh, I- I think that, er, it's- it's quite, you know, interesting."

// errPronunciationUnavailable wraps a pronunciation-service failure, so the
// service can retry the job before settling for an estimate.
var errPronunciationUnavailable = errors.New("pronunciation service unavailable")

// SpeakingDetails is Score.Details for a speaking submission.
type SpeakingDetails struct {
	Mode string `json:"mode"`
	// Indicative is set for part practice: IELTS never scores one part
	// alone, so the band is a guide, not a test result.
	Indicative bool                         `json:"indicative"`
	Criteria   map[string]SpeakingCriterion `json:"criteria"`
	// Flags lists answers the judges set aside as rehearsed or off-topic.
	Flags         SpeakingFlags         `json:"flags"`
	Evidence      SpeakingEvidence      `json:"evidence"`
	Pronunciation *pronunciationSummary `json:"pronunciation"`
	Answers       []answerTranscript    `json:"answers"`
	// Corrections are the grammar and word-choice errors in the answers,
	// each anchored to text that appears verbatim in its answer.
	Corrections []SpeakingCorrection `json:"corrections"`
}

// SpeakingCorrection is one error in an answer and how to fix it.
type SpeakingCorrection struct {
	QuestionID  string `json:"question_id"`
	Type        string `json:"type"` // grammar | vocabulary
	Original    string `json:"original"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
}

// Correction limits: enough to show the pattern of errors without
// burying the answer.
const (
	maxCorrectionsPerAnswer = 6
	maxCorrections          = 40
)

type SpeakingCriterion struct {
	Score        float64           `json:"score"`
	JudgedBand   int               `json:"judged_band"`
	Cap          *bandCap          `json:"cap,omitempty"`
	Feedback     string            `json:"feedback"`
	Improvements []string          `json:"improvements"`
	Checks       []descriptorCheck `json:"checks"`
}

type SpeakingFlags struct {
	Rehearsed []string `json:"rehearsed,omitempty"`
	OffTopic  []string `json:"off_topic,omitempty"`
}

// Grade rates one submission. When the pronunciation service fails and
// estimateOnFailure is false, it returns errPronunciationUnavailable so
// the job is retried; on the last attempt the caller sets it and
// Pronunciation is estimated instead.
func (g *SpeakingGrader) Grade(ctx context.Context, content SpeakingContent, payload SpeakingPayload, estimateOnFailure bool) (*SpeakingDetails, float64, error) {
	questions := map[string]speakingQuestion{}
	for _, q := range content.questions() {
		questions[q.ID] = q
	}

	audio := make([][]byte, len(payload.Answers))
	answers := make([]answerTranscript, len(payload.Answers))
	if err := g.transcribeAll(ctx, payload, questions, audio, answers); err != nil {
		return nil, 0, err
	}

	ev := extractEvidence(answers)
	pron, err := g.assessPronunciation(ctx, answers, audio)
	if err != nil {
		// With no service configured, waiting for a retry changes nothing.
		if !estimateOnFailure && g.pron != nil {
			return nil, 0, err
		}
		log.Printf("speaking: %v — estimating pronunciation", err)
		pron = estimatedPronunciation(ev)
	}

	mode := content.Mode()
	verdicts, err := g.judgeAll(ctx, mode, answers, ev, pron)
	if err != nil {
		return nil, 0, err
	}

	caps := speakingCaps(mode, ev, pron)
	details := &SpeakingDetails{
		Mode:          mode,
		Indicative:    mode != SpeakingModeFull,
		Criteria:      map[string]SpeakingCriterion{},
		Evidence:      ev,
		Pronunciation: pron,
		Answers:       answers,
	}
	var bands []float64
	for _, c := range speakingCriteria {
		v := verdicts[c]
		band, applied := applyCaps(v.Band, caps[c])
		details.Criteria[c] = SpeakingCriterion{
			Score: float64(band), JudgedBand: v.Band, Cap: applied,
			Feedback: v.Feedback, Improvements: v.Improvements, Checks: v.Checks,
		}
		details.Flags.Rehearsed = appendUnique(details.Flags.Rehearsed, v.Rehearsed...)
		details.Flags.OffTopic = appendUnique(details.Flags.OffTopic, v.OffTopic...)
		bands = append(bands, float64(band))
	}
	details.Corrections = anchoredCorrections(answers,
		verdicts[CriterionGRA].Corrections, "grammar",
		verdicts[CriterionLR].Corrections, "vocabulary")
	return details, ieltsOverall(bands), nil
}

// anchoredCorrections keeps only corrections whose original text really
// occurs in the answer they name — a judge that misquotes would otherwise
// mark text the candidate never said — deduplicated and capped. Grammar
// ones come first: grammar and vocabulary judges sometimes flag the same
// span, and the first one wins.
func anchoredCorrections(answers []answerTranscript, grammar []SpeakingCorrection, gType string, vocab []SpeakingCorrection, vType string) []SpeakingCorrection {
	text := map[string]string{}
	for _, a := range answers {
		text[a.QuestionID] = strings.ToLower(a.Text)
	}
	out := []SpeakingCorrection{}
	perAnswer := map[string]int{}
	seen := map[string]bool{}
	add := func(list []SpeakingCorrection, typ string) {
		for _, c := range list {
			c.Type = typ
			c.Original = strings.TrimSpace(c.Original)
			c.Correction = strings.TrimSpace(c.Correction)
			key := c.QuestionID + "\x00" + strings.ToLower(c.Original)
			switch {
			case c.Original == "" || strings.EqualFold(c.Original, c.Correction):
			case !strings.Contains(text[c.QuestionID], strings.ToLower(c.Original)):
			case seen[key], perAnswer[c.QuestionID] >= maxCorrectionsPerAnswer, len(out) >= maxCorrections:
			default:
				seen[key] = true
				perAnswer[c.QuestionID]++
				out = append(out, c)
			}
		}
	}
	add(grammar, gType)
	add(vocab, vType)
	return out
}

// transcribeAll fetches and transcribes every answer, a few at a time.
func (g *SpeakingGrader) transcribeAll(ctx context.Context, payload SpeakingPayload, questions map[string]speakingQuestion, audio [][]byte, answers []answerTranscript) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		sem      = make(chan struct{}, 4)
	)
	for i, a := range payload.Answers {
		wg.Add(1)
		go func(i int, a SpeakingAnswer) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fail := func(err error) {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
			b, err := g.audio.Get(ctx, a.AudioKey)
			if err != nil {
				fail(fmt.Errorf("load recording %s: %w", a.QuestionID, err))
				return
			}
			t, err := g.asr.Transcribe(ctx, g.cfg.WhisperModel, b, path.Base(a.AudioKey), fillerPrompt)
			if err != nil {
				fail(fmt.Errorf("transcribe %s: %w", a.QuestionID, err))
				return
			}
			q := questions[a.QuestionID]
			audio[i] = b
			answers[i] = answerTranscript{
				QuestionID: a.QuestionID, Part: q.Part, Question: q.Text, AudioKey: a.AudioKey,
				Text: strings.TrimSpace(t.Text), Duration: t.Duration,
				Words: t.Words, Segments: t.Segments,
			}
		}(i, a)
	}
	wg.Wait()
	return firstErr
}

// assessPronunciation sends a sample of the answers to the pronunciation
// service, one at a time: it runs on a small free CPU VM.
func (g *SpeakingGrader) assessPronunciation(ctx context.Context, answers []answerTranscript, audio [][]byte) (*pronunciationSummary, error) {
	if g.pron == nil {
		return nil, fmt.Errorf("%w: not configured", errPronunciationUnavailable)
	}
	var clips []clipAssessment
	for _, i := range pronunciationSampleOrder(answers, g.cfg.PronunciationSampleSeconds) {
		a := answers[i]
		res, err := g.pron.Assess(ctx, audio[i], path.Base(a.AudioKey), a.Text, a.Words)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errPronunciationUnavailable, err)
		}
		clips = append(clips, clipAssessment{QuestionID: a.QuestionID, Seconds: a.Duration, A: res})
	}
	s := summarizePronunciation(clips)
	if s == nil {
		return nil, fmt.Errorf("%w: no answer long enough to assess", errPronunciationUnavailable)
	}
	return s, nil
}

// criterionVerdict is one judge's answer.
type criterionVerdict struct {
	Checks       []descriptorCheck    `json:"checks"`
	Band         int                  `json:"band"`
	Feedback     string               `json:"feedback"`
	Improvements []string             `json:"improvements"`
	Rehearsed    []string             `json:"rehearsed_question_ids"`
	OffTopic     []string             `json:"off_topic_question_ids"`
	Corrections  []SpeakingCorrection `json:"corrections"`
}

type descriptorCheck struct {
	Band     int    `json:"band"`
	Feature  string `json:"feature"`
	Verdict  string `json:"verdict"` // met | partly | not_met
	Evidence string `json:"evidence"`
}

var judgeParams = llm.CompletionParams{Temperature: 0.1, MaxTokens: 3500}

// judgeAll runs the four criterion judges in parallel.
func (g *SpeakingGrader) judgeAll(ctx context.Context, mode string, answers []answerTranscript, ev SpeakingEvidence, pron *pronunciationSummary) (map[string]criterionVerdict, error) {
	transcript := formatTranscript(answers)
	out := map[string]criterionVerdict{}
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for _, c := range speakingCriteria {
		wg.Add(1)
		go func(criterion string) {
			defer wg.Done()
			v, err := g.judge1(ctx, criterion, mode, transcript, ev, pron)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("judge %s: %w", criterion, err)
				}
				return
			}
			out[criterion] = *v
		}(c)
	}
	wg.Wait()
	return out, firstErr
}

func (g *SpeakingGrader) judge1(ctx context.Context, criterion, mode, transcript string, ev SpeakingEvidence, pron *pronunciationSummary) (*criterionVerdict, error) {
	params := judgeParams
	params.Model = g.cfg.JudgeModel
	raw, err := g.judge.Complete(ctx, judgeSystemPrompt(criterion, mode), judgeUserPrompt(criterion, transcript, ev, pron), "", params)
	if err != nil {
		return nil, err
	}
	var v criterionVerdict
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, fmt.Errorf("parse verdict: %w", err)
	}
	if v.Band < 1 || v.Band > 9 {
		return nil, fmt.Errorf("band %d is not a whole band from 1 to 9", v.Band)
	}
	return &v, nil
}

func judgeSystemPrompt(criterion, mode string) string {
	scope := "a full three-part IELTS Speaking test"
	if mode != SpeakingModeFull {
		scope = fmt.Sprintf("Part %s of an IELTS Speaking test, practised on its own (rate what this part shows; don't penalise the missing parts)", strings.TrimPrefix(mode, "part"))
	}
	return fmt.Sprintf(`You are a certified IELTS Speaking examiner. You are rating %s on one criterion only: %s.

%s

Band descriptors for %s:
%s

Work in this order:
1. For each band from 9 down to 4 (and lower if needed), check the descriptor's key features against the candidate's speech: "met", "partly" or "not_met", each with a short quote or measurement as evidence.
2. Pick the best-fit band.
3. Write feedback an examiner would give: two to four sentences, specific to this candidate, quoting their own words. Then two or three concrete improvements.
%s
Respond with ONLY valid JSON:
{
  "checks": [ { "band": <int>, "feature": "<descriptor feature>", "verdict": "met|partly|not_met", "evidence": "<quote or measurement>" } ],
  "band": <whole number 1-9>,
  "feedback": "<string>",
  "improvements": ["<string>"],
  "rehearsed_question_ids": ["<id of an answer that sounds memorised>"],
  "off_topic_question_ids": ["<id of an answer that doesn't address its question>"]%s
}`, scope, criterion, examinerPrinciples, criterion, speakingDescriptors[criterion], correctionTask(criterion), correctionField(criterion))
}

// correctionTask asks the grammar and vocabulary judges to list the errors
// they rated, for the review to mark on the transcript.
func correctionTask(criterion string) string {
	kind := map[string]string{
		CriterionGRA: "grammar error (tense, agreement, articles, plurals, word order, prepositions, sentence structure)",
		CriterionLR:  "word-choice error (wrong word, wrong collocation, wrong word form, unnatural phrasing)",
	}[criterion]
	if kind == "" {
		return ""
	}
	return fmt.Sprintf(`4. List each %s in the candidate's answers, up to 6 per answer, most important first:
   - "original": the shortest span that contains the error, copied EXACTLY from that answer's transcript (same words, same spelling), with enough words to be unambiguous;
   - "correction": that span rewritten correctly, changing as little as possible;
   - "explanation": one short sentence on the rule.
   Ignore fillers, hesitations, repetitions and self-corrections (they belong to fluency), and words that are clearly speech-recognition mistakes.
`, kind)
}

func correctionField(criterion string) string {
	if criterion != CriterionGRA && criterion != CriterionLR {
		return ""
	}
	return `,
  "corrections": [ { "question_id": "<id>", "original": "<exact text from the answer>", "correction": "<fixed text>", "explanation": "<rule>" } ]`
}

func judgeUserPrompt(criterion, transcript string, ev SpeakingEvidence, pron *pronunciationSummary) string {
	var b strings.Builder
	b.WriteString("Transcript (verbatim speech recognition, by question):\n")
	b.WriteString(transcript)
	b.WriteString("\n\nMeasurements from the recordings:\n")
	switch criterion {
	case CriterionFC:
		writeJSON(&b, map[string]any{"fluency": ev.Fluency, "coherence": ev.Coherence})
		b.WriteString("\nFor each long pause, decide whether the speaker was planning content (acceptable at bands 8-9) or searching for words or grammar (bands 7 and below). Mid-clause pauses usually mean a language search.")
	case CriterionLR:
		writeJSON(&b, map[string]any{"lexis": ev.Lexis, "word_count": ev.Fluency.WordCount})
		b.WriteString("\nJudge range, precision, less common and idiomatic items, collocation and paraphrase from the transcript itself; the numbers only anchor variety.")
	case CriterionGRA:
		writeJSON(&b, map[string]any{"grammar": ev.Grammar})
		b.WriteString("\nCount error-free sentences, and look at the range of structures and how accurate the complex ones are, in the transcript itself.")
	case CriterionP:
		writeJSON(&b, map[string]any{"pronunciation": pron, "speech_rate_wpm": ev.Fluency.SpeechRateWPM, "mid_clause_pause_share": ev.Fluency.MidClausePauses})
		b.WriteString(`
You cannot hear the audio. Rate from these measurements and the transcript:
- intelligibility: share of words heard as intended (0-1). Low values mean the listener needs effort (bands 4-5).
- per and gop_mean: phoneme accuracy. Scores are against both British and American references, so accent alone doesn't lower them.
- prosody.pitch_std_st and pitch_range_st: intonation. Under about 1.5 semitones of variation is flat.
- prosody.stress_match_rate: how often word stress is on the right syllable.
- prosody.npvi_vowel: rhythm. Higher values mean English stress-timing; under about 40 is syllable-timed.
- mid_clause_pause_share: chunking. Pauses inside clauses break meaningful chunks.
- mispronounced: the recurring problem words, with the expected and heard sound.`)
		if pron != nil && pron.Estimated {
			b.WriteString("\nPronunciation could not be measured directly this time: intelligibility is estimated from speech-recognition confidence only, and there is no prosody data. Rate conservatively and say so in the feedback.")
		}
	}
	return b.String()
}

func formatTranscript(answers []answerTranscript) string {
	var b strings.Builder
	part := 0
	for _, a := range answers {
		if a.Part != part {
			part = a.Part
			fmt.Fprintf(&b, "\n=== Part %d ===\n", part)
		}
		q := a.Question
		if a.QuestionID == "p2" {
			q = strings.ReplaceAll(q, "\n", " / ")
		}
		fmt.Fprintf(&b, "[%s] Examiner: %s\nCandidate (%.0f s): %s\n", a.QuestionID, q, a.Duration, orDash(a.Text))
	}
	return b.String()
}

func writeJSON(b *strings.Builder, v any) {
	j, _ := json.MarshalIndent(v, "", "  ")
	b.Write(j)
}

func orDash(s string) string {
	if s == "" {
		return "(no speech)"
	}
	return s
}

func appendUnique(dst []string, src ...string) []string {
	for _, s := range src {
		dup := false
		for _, d := range dst {
			if d == s {
				dup = true
				break
			}
		}
		if !dup && s != "" {
			dst = append(dst, s)
		}
	}
	return dst
}
