package ielts_test

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
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

// speakingCaps derives the ceilings for each criterion from the evidence.
func speakingCaps(mode string, ev SpeakingEvidence, pron *pronunciationSummary) map[string][]bandCap {
	caps := map[string][]bandCap{}
	add := func(criterion string, max int, format string, args ...any) {
		caps[criterion] = append(caps[criterion], bandCap{Max: max, Reason: fmt.Sprintf(format, args...)})
	}
	f := ev.Fluency

	// Too little speech to rate is capped everywhere: descriptors above
	// band 3 all describe sustained speech.
	if f.WordCount < capMinWordsToRate {
		for _, c := range speakingCriteria {
			add(c, 3, "only %d words were spoken — not enough language to rate higher", f.WordCount)
		}
	}

	if mode == SpeakingModeFull || mode == SpeakingModePart2 {
		switch {
		case f.LongTurnSeconds < capLongTurnShort:
			add(CriterionFC, 5, "the Part 2 talk lasted %.0f s; band 6 needs a willingness to produce long turns", f.LongTurnSeconds)
		case f.LongTurnSeconds < capLongTurnMedium:
			add(CriterionFC, 6, "the Part 2 talk lasted %.0f s, well short of the two minutes", f.LongTurnSeconds)
		}
	}
	if mode == SpeakingModePart1 {
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

// pronunciationSummary combines the per-clip assessments, weighting each
// clip by its length. Estimated means the service was unavailable and the
// figures come from speech-recognition confidence instead.
type pronunciationSummary struct {
	Estimated       bool                  `json:"estimated"`
	Intelligibility float64               `json:"intelligibility"`
	GOPMean         float64               `json:"gop_mean,omitempty"`
	PER             float64               `json:"per,omitempty"`
	Prosody         pronunciation.Prosody `json:"prosody"`
	SampledSeconds  float64               `json:"sampled_seconds"`
	Mispronounced   []mispronouncedWord   `json:"mispronounced,omitempty"`
	// Words holds every assessed word per question, so the review can
	// show what the candidate said next to what was expected.
	Words map[string][]assessedWord `json:"words,omitempty"`
}

type mispronouncedWord struct {
	Word     string  `json:"word"`
	Score    float64 `json:"score"`
	Expected string  `json:"expected,omitempty"`
	Heard    string  `json:"heard,omitempty"`
	Count    int     `json:"count"`
}

// assessedWord is one word of an answer as the pronunciation service
// heard it. Index is its position in the answer's recognised word list.
type assessedWord struct {
	Index    int                     `json:"index"`
	Word     string                  `json:"word"`
	Score    float64                 `json:"score"`
	Start    float64                 `json:"start"`
	End      float64                 `json:"end"`
	Expected string                  `json:"expected"`
	Heard    string                  `json:"heard"`
	Phonemes []pronunciation.Phoneme `json:"phonemes"`
}

// mispronouncedBelow is the word score (0-100) under which a word is
// reported as mispronounced.
const mispronouncedBelow = 60

type clipAssessment struct {
	QuestionID string
	Seconds    float64
	A          *pronunciation.Assessment
}

func summarizePronunciation(clips []clipAssessment) *pronunciationSummary {
	s := &pronunciationSummary{Words: map[string][]assessedWord{}}
	byWord := map[string]*mispronouncedWord{}
	var w float64
	for _, c := range clips {
		if c.A == nil || c.Seconds <= 0 {
			continue
		}
		w += c.Seconds
		s.Intelligibility += c.A.Intelligibility * c.Seconds
		s.GOPMean += c.A.GOPMean * c.Seconds
		s.PER += c.A.PER * c.Seconds
		s.Prosody.PitchRangeST += c.A.Prosody.PitchRangeST * c.Seconds
		s.Prosody.PitchStdST += c.A.Prosody.PitchStdST * c.Seconds
		s.Prosody.StressMatchRate += c.A.Prosody.StressMatchRate * c.Seconds
		s.Prosody.NPVIVowel += c.A.Prosody.NPVIVowel * c.Seconds
		for i, word := range c.A.Words {
			// No phonemes means the word couldn't be aligned at all — a
			// timing problem, not evidence of mispronunciation.
			if len(word.Phonemes) == 0 {
				continue
			}
			expected := make([]string, len(word.Phonemes))
			for k, p := range word.Phonemes {
				expected[k] = p.Expected
			}
			s.Words[c.QuestionID] = append(s.Words[c.QuestionID], assessedWord{
				Index: i, Word: word.Word, Score: round1(word.Score), Start: word.Start, End: word.End,
				Expected: strings.Join(expected, " "), Heard: word.Heard, Phonemes: word.Phonemes,
			})
			if word.Score >= mispronouncedBelow {
				continue
			}
			key := normalizeToken(word.Word)
			m, ok := byWord[key]
			if !ok {
				m = &mispronouncedWord{Word: key, Score: word.Score}
				expected, heard := worstPhoneme(word.Phonemes)
				m.Expected, m.Heard = expected, heard
				byWord[key] = m
			}
			m.Count++
			m.Score = math.Min(m.Score, word.Score)
		}
	}
	if w == 0 {
		return nil
	}
	s.SampledSeconds = round1(w)
	s.Intelligibility = round2(s.Intelligibility / w)
	s.GOPMean = round1(s.GOPMean / w)
	s.PER = round2(s.PER / w)
	s.Prosody.PitchRangeST = round1(s.Prosody.PitchRangeST / w)
	s.Prosody.PitchStdST = round1(s.Prosody.PitchStdST / w)
	s.Prosody.StressMatchRate = round2(s.Prosody.StressMatchRate / w)
	s.Prosody.NPVIVowel = round1(s.Prosody.NPVIVowel / w)
	for _, m := range byWord {
		m.Score = round1(m.Score)
		s.Mispronounced = append(s.Mispronounced, *m)
	}
	sort.Slice(s.Mispronounced, func(i, j int) bool {
		if s.Mispronounced[i].Count != s.Mispronounced[j].Count {
			return s.Mispronounced[i].Count > s.Mispronounced[j].Count
		}
		return s.Mispronounced[i].Score < s.Mispronounced[j].Score
	})
	if len(s.Mispronounced) > 25 {
		s.Mispronounced = s.Mispronounced[:25]
	}
	return s
}

func worstPhoneme(ps []pronunciation.Phoneme) (expected, heard string) {
	worst := math.Inf(1)
	for _, p := range ps {
		if p.Score < worst {
			worst, expected, heard = p.Score, p.Expected, p.Heard
		}
	}
	return expected, heard
}

// estimatedPronunciation stands in when the pronunciation service can't be
// reached: speech recognition struggles with the same speech a listener
// would, so its confidence is a rough intelligibility signal.
func estimatedPronunciation(ev SpeakingEvidence) *pronunciationSummary {
	return &pronunciationSummary{
		Estimated:       true,
		Intelligibility: round2(1 - ev.Recognition.LowConfidenceShare),
	}
}

// pronunciationSampleOrder picks which answers to send to the
// pronunciation service, longest stretches of connected speech first
// (the Part 2 talk, then Part 3, then Part 1), until sampleSeconds is
// covered. The service runs on a small free CPU VM, so it rates a sample of the
// test as an examiner's overall impression would, not every second.
func pronunciationSampleOrder(answers []answerTranscript, sampleSeconds float64) []int {
	idx := make([]int, 0, len(answers))
	for i, a := range answers {
		if len(a.Words) >= 5 {
			idx = append(idx, i)
		}
	}
	rank := func(a answerTranscript) int {
		switch {
		case a.QuestionID == "p2":
			return 0
		case a.Part == 3:
			return 1
		case a.Part == 1:
			return 2
		}
		return 3
	}
	sort.SliceStable(idx, func(i, j int) bool {
		ai, aj := answers[idx[i]], answers[idx[j]]
		if rank(ai) != rank(aj) {
			return rank(ai) < rank(aj)
		}
		return ai.Duration > aj.Duration
	})
	var picked []int
	total := 0.0
	for _, i := range idx {
		if total >= sampleSeconds {
			break
		}
		picked = append(picked, i)
		total += answers[i].Duration
	}
	return picked
}
