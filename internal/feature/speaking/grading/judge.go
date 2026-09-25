package grading

import (
	"context"
	"encoding/json"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"strings"
	"sync"
)

// criterionVerdict is one judge's answer.
type criterionVerdict struct {
	Checks       []descriptorCheck `json:"checks"`
	Band         int               `json:"band"`
	Feedback     string            `json:"feedback"`
	Improvements []string          `json:"improvements"`
	Rehearsed    []string          `json:"rehearsed_question_ids"`
	OffTopic     []string          `json:"off_topic_question_ids"`
	Corrections  []Correction      `json:"corrections"`
}

type descriptorCheck struct {
	Band     int    `json:"band"`
	Feature  string `json:"feature"`
	Verdict  string `json:"verdict"` // met | partly | not_met
	Evidence string `json:"evidence"`
}

var judgeParams = llm.CompletionParams{Temperature: 0.1, MaxTokens: 3500}

// judgeAll runs the four criterion judges in parallel.
func (g *Grader) judgeAll(ctx context.Context, mode string, answers []answerTranscript, ev Evidence, pron *pronunciationSummary) (map[string]criterionVerdict, error) {
	transcript := formatTranscript(answers)
	out := map[string]criterionVerdict{}
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for _, c := range criteria {
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

func (g *Grader) judge1(ctx context.Context, criterion, mode, transcript string, ev Evidence, pron *pronunciationSummary) (*criterionVerdict, error) {
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
	if mode != ielts_test.SpeakingModeFull {
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
}`, scope, criterion, examinerPrinciples, criterion, descriptors[criterion], correctionTask(criterion), correctionField(criterion))
}

func judgeUserPrompt(criterion, transcript string, ev Evidence, pron *pronunciationSummary) string {
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
