package grading

import (
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
)

// bandCap is the highest band the measurements allow for one criterion,
// and why. A judge's band above it is lowered to it.
type bandCap struct {
	Max    int    `json:"max"`
	Reason string `json:"reason"`
}

// Guardrail thresholds. They are starting points drawn from the
// descriptors ("long turns", "noticeable pauses", "effort to understand")
// and typical L2 fluency figures, and must be tuned against examiner-rated
// samples (see speaking calibration) before bands are shown as more than
// estimates.
const (
	capLongTurnShort     = 45.0 // seconds of Part 2 talk
	capLongTurnMedium    = 60.0
	capSlowSpeechWPM     = 70.0
	capVerySlowWPM       = 50.0
	capLongPausesHigh    = 6.0 // long pauses per minute
	capLongPausesExtreme = 10.0
	capMinWordsToRate    = 40
)

// capsFor derives the ceilings for each criterion from the evidence.
func capsFor(mode string, ev Evidence, pron *pronunciationSummary) map[string][]bandCap {
	caps := map[string][]bandCap{}
	add := func(criterion string, max int, format string, args ...any) {
		caps[criterion] = append(caps[criterion], bandCap{Max: max, Reason: fmt.Sprintf(format, args...)})
	}
	f := ev.Fluency

	// Too little speech to rate is capped everywhere: descriptors above
	// band 3 all describe sustained speech.
	if f.WordCount < capMinWordsToRate {
		for _, c := range criteria {
			add(c, 3, "only %d words were spoken — not enough language to rate higher", f.WordCount)
		}
	}

	if mode == ielts_test.SpeakingModeFull || mode == ielts_test.SpeakingModePart2 {
		switch {
		case f.LongTurnSeconds < capLongTurnShort:
			add(CriterionFC, 5, "the Part 2 talk lasted %.0f s; band 6 needs a willingness to produce long turns", f.LongTurnSeconds)
		case f.LongTurnSeconds < capLongTurnMedium:
			add(CriterionFC, 6, "the Part 2 talk lasted %.0f s, well short of the two minutes", f.LongTurnSeconds)
		}
	}
	if mode == ielts_test.SpeakingModePart1 {
		add(CriterionFC, 7, "Part 1 alone has only short answers; band 8 needs extended, developed speech")
	}
	switch {
	case f.LongPausesPerMin > capLongPausesExtreme:
		add(CriterionFC, 4, "%.1f pauses of a second or more per minute", f.LongPausesPerMin)
	case f.LongPausesPerMin > capLongPausesHigh:
		add(CriterionFC, 5, "%.1f pauses of a second or more per minute", f.LongPausesPerMin)
	}
	switch {
	case f.WordCount >= capMinWordsToRate && f.SpeechRateWPM < capVerySlowWPM:
		add(CriterionFC, 4, "speech rate of %.0f words per minute", f.SpeechRateWPM)
	case f.WordCount >= capMinWordsToRate && f.SpeechRateWPM < capSlowSpeechWPM:
		add(CriterionFC, 5, "speech rate of %.0f words per minute", f.SpeechRateWPM)
	}

	if pron == nil || pron.Estimated {
		add(CriterionP, 7, "pronunciation was estimated from speech-recognition confidence; features needed for band 8 couldn't be measured")
	}
	if pron != nil && !pron.Estimated {
		switch {
		case pron.Intelligibility < 0.6:
			add(CriterionP, 4, "only %.0f%% of words were clearly intelligible", 100*pron.Intelligibility)
		case pron.Intelligibility < 0.75:
			add(CriterionP, 5, "%.0f%% of words were clearly intelligible; band 6 is understood throughout without much effort", 100*pron.Intelligibility)
		case pron.Intelligibility < 0.85:
			add(CriterionP, 6, "%.0f%% of words were clearly intelligible; band 8 is easily understood throughout", 100*pron.Intelligibility)
		}
		if pron.Prosody.PitchStdST > 0 && pron.Prosody.PitchStdST < 1.5 {
			add(CriterionP, 6, "intonation is flat (pitch varies by %.1f semitones); band 8 needs flexible intonation", pron.Prosody.PitchStdST)
		}
	}
	return caps
}

// applyCaps lowers band to the tightest cap, returning the cap applied.
func applyCaps(band int, caps []bandCap) (int, *bandCap) {
	var tightest *bandCap
	for i := range caps {
		if tightest == nil || caps[i].Max < tightest.Max {
			tightest = &caps[i]
		}
	}
	if tightest != nil && band > tightest.Max {
		return tightest.Max, tightest
	}
	return band, nil
}
