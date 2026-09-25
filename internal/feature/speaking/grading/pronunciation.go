package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
	"math"
	"sort"
	"strings"
)

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
	// Ties are broken by the word itself: the list comes out of a map and is
	// cut to 25, so without that the words kept (and shown to the judge)
	// would vary from run to run.
	sort.Slice(s.Mispronounced, func(i, j int) bool {
		a, b := s.Mispronounced[i], s.Mispronounced[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.Score != b.Score {
			return a.Score < b.Score
		}
		return a.Word < b.Word
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
func estimatedPronunciation(ev Evidence) *pronunciationSummary {
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
