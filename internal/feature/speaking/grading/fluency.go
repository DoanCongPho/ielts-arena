package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"strings"
)

// Pause thresholds, in seconds. 0.4 s is the conventional floor for a
// silent pause in L2 fluency research; a pause of a second or more is one a
// listener clearly notices.
const (
	pauseMin     = 0.4
	longPauseMin = 1.0
)

type FluencyMetrics struct {
	WordCount int `json:"word_count"`
	// SpeakingSeconds counts from each answer's first word to its last,
	// so the silence before starting doesn't count as disfluency.
	SpeakingSeconds float64 `json:"speaking_seconds"`
	// PartSeconds is SpeakingSeconds per part ("1", "2", "3").
	PartSeconds map[string]float64 `json:"part_seconds"`
	// LongTurnSeconds is how long the Part 2 talk lasted.
	LongTurnSeconds float64 `json:"long_turn_seconds"`
	SpeechRateWPM   float64 `json:"speech_rate_wpm"`
	// ArticulationRateWPM leaves pauses out: how fast the speaker talks
	// when they are talking.
	ArticulationRateWPM float64 `json:"articulation_rate_wpm"`
	PausesPerMin        float64 `json:"pauses_per_min"`
	LongPausesPerMin    float64 `json:"long_pauses_per_min"`
	// MeanLengthOfRun is the mean number of words between pauses.
	MeanLengthOfRun float64 `json:"mean_length_of_run"`
	// FillersPer100Words counts hesitation sounds (um, uh, er…).
	FillersPer100Words float64 `json:"fillers_per_100_words"`
	// Repairs counts repeated words and restarted phrases.
	Repairs         int     `json:"repairs"`
	RepairsPer100   float64 `json:"repairs_per_100_words"`
	MidClausePauses float64 `json:"mid_clause_pause_share"`
	// LongPauses lists each noticeable pause with the words around it,
	// for the judge to tell content-planning pauses from word searches.
	LongPauses []PauseContext `json:"long_pauses"`
}

type PauseContext struct {
	QuestionID string  `json:"question_id"`
	Seconds    float64 `json:"seconds"`
	Before     string  `json:"before"`
	After      string  `json:"after"`
	MidClause  bool    `json:"mid_clause"`
}

var hesitationFillers = map[string]bool{
	"um": true, "umm": true, "uh": true, "uhh": true, "er": true, "erm": true,
	"ah": true, "hmm": true, "mm": true, "eh": true,
}

// clauseBoundaries marks, for each word, whether a clause or sentence ends
// right after it — taken from the punctuation the recogniser put in the
// text, matched to the (unpunctuated) word list in order. A comma before a
// filler ("it is, um, a novel") is the recogniser setting off the
// hesitation, not a clause ending, so it doesn't count.
func clauseBoundaries(text string, words []llm.TranscriptWord) []bool {
	out := make([]bool, len(words))
	toks := strings.Fields(text)
	j := 0
	for i, w := range words {
		want := normalizeToken(w.Word)
		for k := j; k < len(toks) && k < j+4; k++ {
			if normalizeToken(toks[k]) != want {
				continue
			}
			switch last := toks[k][len(toks[k])-1]; {
			case strings.IndexByte(".;:?!", last) >= 0:
				out[i] = true
			case last == ',':
				out[i] = i+1 >= len(words) || !hesitationFillers[normalizeToken(words[i+1].Word)]
			}
			j = k + 1
			break
		}
	}
	return out
}

// repeatedBigrams counts a two-word phrase said twice in a row ("I think
// I think") — a restart rather than deliberate emphasis.
func repeatedBigrams(toks []string) int {
	n := 0
	for i := 0; i+3 < len(toks); i++ {
		if toks[i] == toks[i+2] && toks[i+1] == toks[i+3] {
			n++
		}
	}
	return n
}

// fluencyMetrics measures speed, pauses, fillers and repairs.
func fluencyMetrics(answers []answerTranscript) FluencyMetrics {
	f := FluencyMetrics{PartSeconds: map[string]float64{}}
	var (
		runs       []int
		pauseTotal float64
		pauses     int
		longPauses int
		fillers    int
		midClause  int
	)
	for _, a := range answers {
		if !hasSpeech(a) {
			continue
		}
		words := a.Words
		span := words[len(words)-1].End - words[0].Start
		f.SpeakingSeconds += span
		f.PartSeconds[partKey(a.Part)] += span
		if a.QuestionID == "p2" {
			f.LongTurnSeconds = span
		}

		boundaries := clauseBoundaries(a.Text, words)
		run := 0
		var prev string
		for i, w := range words {
			tok := normalizeToken(w.Word)
			if tok == "" {
				continue
			}
			if hesitationFillers[tok] {
				fillers++
			} else {
				f.WordCount++
				if tok == prev {
					f.Repairs++
				}
				prev = tok
			}
			run++
			if i+1 >= len(words) {
				continue
			}
			gap := words[i+1].Start - w.End
			if gap < pauseMin {
				continue
			}
			pauses++
			pauseTotal += gap
			runs = append(runs, run)
			run = 0
			if gap >= longPauseMin {
				longPauses++
				atBoundary := boundaries[i]
				if !atBoundary {
					midClause++
				}
				f.LongPauses = append(f.LongPauses, PauseContext{
					QuestionID: a.QuestionID,
					Seconds:    round2(gap),
					Before:     joinWords(words, i-4, i+1),
					After:      joinWords(words, i+1, i+5),
					MidClause:  !atBoundary,
				})
			}
		}
		if run > 0 {
			runs = append(runs, run)
		}
		f.Repairs += repeatedBigrams(spokenTokens(a))
	}

	if minutes := f.SpeakingSeconds / 60; minutes > 0 {
		f.SpeechRateWPM = round1(float64(f.WordCount) / minutes)
		f.PausesPerMin = round2(float64(pauses) / minutes)
		f.LongPausesPerMin = round2(float64(longPauses) / minutes)
		if phonation := f.SpeakingSeconds - pauseTotal; phonation > 0 {
			f.ArticulationRateWPM = round1(float64(f.WordCount) / (phonation / 60))
		}
	}
	f.MeanLengthOfRun = round1(meanInts(runs))
	if f.WordCount > 0 {
		f.FillersPer100Words = round1(100 * float64(fillers) / float64(f.WordCount))
		f.RepairsPer100 = round1(100 * float64(f.Repairs) / float64(f.WordCount))
	}
	if longPauses > 0 {
		f.MidClausePauses = round2(float64(midClause) / float64(longPauses))
	}
	return f
}
