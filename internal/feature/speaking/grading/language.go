package grading

import "strings"

type LexisMetrics struct {
	// MATTR is the moving-average type-token ratio over a 50-word window:
	// lexical variety independent of how much was said.
	MATTR         float64 `json:"mattr"`
	DistinctWords int     `json:"distinct_words"`
}

type GrammarMetrics struct {
	Sentences         int     `json:"sentences"`
	MeanSentenceWords float64 `json:"mean_sentence_words"`
	// ComplexSentenceShare is the share of sentences with a subordinating
	// word — a rough count; the judge assesses accuracy itself.
	ComplexSentenceShare float64 `json:"complex_sentence_share"`
}

type RecognitionMetrics struct {
	MeanLogprob float64 `json:"mean_logprob"`
	// LowConfidenceShare is the share of speech time in segments whose
	// mean log-probability is below -0.8.
	LowConfidenceShare float64 `json:"low_confidence_share"`
}

var subordinators = map[string]bool{
	"because": true, "although": true, "though": true, "whereas": true, "while": true,
	"which": true, "who": true, "whom": true, "whose": true, "when": true, "whenever": true,
	"if": true, "unless": true, "since": true, "until": true, "after": true, "before": true,
	"where": true, "whether": true,
}

func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == '.' || r == '?' || r == '!' })
}

// mattr is the moving-average type-token ratio; a text shorter than the
// window falls back to the plain ratio.
func mattr(toks []string, window int) float64 {
	if len(toks) == 0 {
		return 0
	}
	if len(toks) <= window {
		return float64(distinct(toks)) / float64(len(toks))
	}
	counts := map[string]int{}
	for _, t := range toks[:window] {
		counts[t]++
	}
	sum := float64(len(counts)) / float64(window)
	for i := window; i < len(toks); i++ {
		out := toks[i-window]
		if counts[out]--; counts[out] == 0 {
			delete(counts, out)
		}
		counts[toks[i]]++
		sum += float64(len(counts)) / float64(window)
	}
	return sum / float64(len(toks)-window+1)
}

func distinct(toks []string) int {
	seen := map[string]bool{}
	for _, t := range toks {
		seen[t] = true
	}
	return len(seen)
}
