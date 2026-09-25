package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/llm"
)

// answerTranscript is one recorded answer after speech recognition.
type answerTranscript struct {
	QuestionID string                  `json:"question_id"`
	Part       int                     `json:"part"`
	Question   string                  `json:"question"`
	AudioKey   string                  `json:"audio_key"`
	Text       string                  `json:"text"`
	Duration   float64                 `json:"duration"`
	Words      []llm.TranscriptWord    `json:"words"`
	Segments   []llm.TranscriptSegment `json:"-"`
}

// Evidence is everything measured from the recordings, which the
// criterion judges read and the guardrails check. None of it is a band on
// its own: the descriptors are judged as a whole, these are the facts.
type Evidence struct {
	Fluency   FluencyMetrics   `json:"fluency"`
	Coherence CoherenceMetrics `json:"coherence"`
	Lexis     LexisMetrics     `json:"lexis"`
	Grammar   GrammarMetrics   `json:"grammar"`
	// Recognition is how confidently speech recognition made out the
	// words — the fallback intelligibility signal when the pronunciation
	// service is unavailable.
	Recognition RecognitionMetrics `json:"recognition"`
}

// extractEvidence measures a whole test's answers, one dimension at a time.
func extractEvidence(answers []answerTranscript) Evidence {
	return Evidence{
		Fluency:     fluencyMetrics(answers),
		Coherence:   coherenceMetrics(answers),
		Lexis:       lexisMetrics(answers),
		Grammar:     grammarMetrics(answers),
		Recognition: recognitionMetrics(answers),
	}
}

// spokenTokens is an answer's words, lowercased, without hesitation
// fillers: what lexis and answer length are measured on.
func spokenTokens(a answerTranscript) []string {
	var out []string
	for _, w := range a.Words {
		if tok := normalizeToken(w.Word); tok != "" && !hesitationFillers[tok] {
			out = append(out, tok)
		}
	}
	return out
}

// hasSpeech reports whether an answer has any recognised words. Answers
// without are left out of every measure.
func hasSpeech(a answerTranscript) bool { return len(a.Words) > 0 }
