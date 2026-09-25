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
