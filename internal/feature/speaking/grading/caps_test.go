package grading

import (
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"testing"
)

func TestSpeakingCaps(t *testing.T) {
	ev := Evidence{Fluency: FluencyMetrics{WordCount: 400, SpeechRateWPM: 120, LongTurnSeconds: 40}}
	caps := capsFor(ielts_test.SpeakingModeFull, ev, &pronunciationSummary{Intelligibility: 0.95})
	if band, c := applyCaps(7, caps[CriterionFC]); band != 5 || c == nil {
		t.Errorf("a 40 s long turn left FC at %d", band)
	}
	if band, _ := applyCaps(8, caps[CriterionP]); band != 8 {
		t.Errorf("clear pronunciation capped to %d", band)
	}

	caps = capsFor(ielts_test.SpeakingModePart1, Evidence{Fluency: FluencyMetrics{WordCount: 200, SpeechRateWPM: 140}}, estimatedPronunciation(Evidence{}))
	if band, _ := applyCaps(9, caps[CriterionFC]); band != 7 {
		t.Errorf("Part 1 alone let FC reach %d", band)
	}
	if band, _ := applyCaps(9, caps[CriterionP]); band != 7 {
		t.Errorf("estimated pronunciation let P reach %d", band)
	}

	caps = capsFor(ielts_test.SpeakingModeFull, Evidence{Fluency: FluencyMetrics{WordCount: 10}}, nil)
	for _, c := range criteria {
		if band, _ := applyCaps(6, caps[c]); band != 3 {
			t.Errorf("%s = %d with ten words spoken, want 3", c, band)
		}
	}
}
