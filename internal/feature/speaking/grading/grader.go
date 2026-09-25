package grading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
	"log"
	"path"
	"strings"
	"sync"
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

// Config picks the models. JudgeModel rates the descriptors;
// it is a judgement task, so a stronger model than the writing default
// pays for itself at four calls per test.
type Config struct {
	WhisperModel string
	JudgeModel   string
	// PronunciationSampleSeconds is how much speech is sent to the
	// pronunciation service per test.
	PronunciationSampleSeconds float64
}

type Grader struct {
	audio audioSource
	asr   transcriber
	pron  pronunciationAssessor // nil: always estimate
	judge completer
	cfg   Config
}

func New(audio audioSource, asr transcriber, pron pronunciationAssessor, judge completer, cfg Config) *Grader {
	if cfg.WhisperModel == "" {
		cfg.WhisperModel = "whisper-1"
	}
	if cfg.PronunciationSampleSeconds <= 0 {
		// Enough for a whole test (11-14 minutes), so every answer gets a
		// word-by-word comparison on the review page.
		cfg.PronunciationSampleSeconds = 900
	}
	return &Grader{audio: audio, asr: asr, pron: pron, judge: judge, cfg: cfg}
}

// fillerPrompt biases Whisper toward a verbatim transcript. Without it,
// Whisper drops "um"s and restarts — exactly what fluency is rated on.
const fillerPrompt = "Umm, let me think, like, hmm... Okay, so, uh, I- I think that, er, it's- it's quite, you know, interesting."

// errPronunciationUnavailable wraps a pronunciation-service failure, so the
// service can retry the job before settling for an estimate.
var errPronunciationUnavailable = errors.New("pronunciation service unavailable")

// Grade rates one submission. When the pronunciation service fails and
// estimateOnFailure is false, it returns errPronunciationUnavailable so
// the job is retried; on the last attempt the caller sets it and
// Pronunciation is estimated instead.
func (g *Grader) Grade(ctx context.Context, content ielts_test.SpeakingContent, payload ielts_test.SpeakingPayload, estimateOnFailure bool) (*Details, float64, error) {
	questions := map[string]ielts_test.SpeakingQuestionRef{}
	for _, q := range content.Questions() {
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

	caps := capsFor(mode, ev, pron)
	details := &Details{
		Mode:          mode,
		Indicative:    mode != ielts_test.SpeakingModeFull,
		Criteria:      map[string]Criterion{},
		Evidence:      ev,
		Pronunciation: pron,
		Answers:       answers,
	}
	var bands []float64
	for _, c := range criteria {
		v := verdicts[c]
		band, applied := applyCaps(v.Band, caps[c])
		details.Criteria[c] = Criterion{
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
	return details, ielts_test.IELTSOverall(bands), nil
}

// transcribeAll fetches and transcribes every answer, a few at a time.
func (g *Grader) transcribeAll(ctx context.Context, payload ielts_test.SpeakingPayload, questions map[string]ielts_test.SpeakingQuestionRef, audio [][]byte, answers []answerTranscript) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		sem      = make(chan struct{}, 4)
	)
	for i, a := range payload.Answers {
		wg.Add(1)
		go func(i int, a ielts_test.SpeakingAnswer) {
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
func (g *Grader) assessPronunciation(ctx context.Context, answers []answerTranscript, audio [][]byte) (*pronunciationSummary, error) {
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

// GradeSpeaking implements ielts_test.SpeakingGrader: Grade, with the
// details marshalled the way they are stored.
func (g *Grader) GradeSpeaking(ctx context.Context, content ielts_test.SpeakingContent, payload ielts_test.SpeakingPayload, lastAttempt bool) (float64, json.RawMessage, error) {
	details, overall, err := g.Grade(ctx, content, payload, lastAttempt)
	if err != nil {
		return 0, nil, err
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal score details: %w", err)
	}
	return overall, raw, nil
}

var _ ielts_test.SpeakingGrader = (*Grader)(nil)
