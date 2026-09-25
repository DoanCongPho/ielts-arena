package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"strings"
	"testing"
)

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
