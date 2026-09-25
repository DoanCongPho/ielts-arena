package ielts_test

import (
	"math"
	"sort"
	"strings"
	"unicode"

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

// Pause thresholds, in seconds. 0.4 s is the conventional floor for a
// silent pause in L2 fluency research; a pause of a second or more is one a
// listener clearly notices.
const (
	pauseMin     = 0.4
	longPauseMin = 1.0
)

// SpeakingEvidence is everything measured from the recordings, which the
// criterion judges read and the guardrails check. None of it is a band on
// its own: the descriptors are judged as a whole, these are the facts.
type SpeakingEvidence struct {
	Fluency   FluencyMetrics   `json:"fluency"`
	Coherence CoherenceMetrics `json:"coherence"`
	Lexis     LexisMetrics     `json:"lexis"`
	Grammar   GrammarMetrics   `json:"grammar"`
	// Recognition is how confidently speech recognition made out the
	// words — the fallback intelligibility signal when the pronunciation
	// service is unavailable.
	Recognition RecognitionMetrics `json:"recognition"`
}

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

type CoherenceMetrics struct {
	DiscourseMarkers map[string]int `json:"discourse_markers"`
	DistinctMarkers  int            `json:"distinct_markers"`
	// MostUsedMarkerShare is the share of all marker uses taken by the
	// most frequent one — high values are the "overuse" the band 5
	// descriptor talks about.
	MostUsedMarkerShare float64 `json:"most_used_marker_share"`
	// MeanAnswerWords is the mean answer length per part ("1", "2", "3").
	MeanAnswerWords map[string]float64 `json:"mean_answer_words"`
	// MinimalAnswers counts Part 1/3 answers under 12 words — answers that
	// don't develop the topic at all.
	MinimalAnswers int `json:"minimal_answers"`
}

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

var hesitationFillers = map[string]bool{
	"um": true, "umm": true, "uh": true, "uhh": true, "er": true, "erm": true,
	"ah": true, "hmm": true, "mm": true, "eh": true,
}

var discourseMarkers = []string{
	"actually", "anyway", "as a result", "as well as", "at the same time", "basically",
	"because", "but", "consequently", "for example", "for instance", "firstly", "secondly",
	"finally", "however", "i mean", "in addition", "in contrast", "in fact", "in my opinion",
	"moreover", "nevertheless", "on the other hand", "on top of that", "overall", "personally",
	"so", "such as", "that said", "therefore", "to be honest", "what's more", "whereas",
	"although", "while", "apart from that", "besides", "and then", "you know", "well",
}

var subordinators = map[string]bool{
	"because": true, "although": true, "though": true, "whereas": true, "while": true,
	"which": true, "who": true, "whom": true, "whose": true, "when": true, "whenever": true,
	"if": true, "unless": true, "since": true, "until": true, "after": true, "before": true,
	"where": true, "whether": true,
}

// extractEvidence measures a whole test's answers.
func extractEvidence(answers []answerTranscript) SpeakingEvidence {
	var ev SpeakingEvidence
	f := &ev.Fluency
	f.PartSeconds = map[string]float64{}

	var (
		allTokens    []string // lowercased words, fillers excluded
		runs         []int
		pauseTotal   float64
		pauses       int
		longPauses   int
		fillers      int
		midClause    int
		segSeconds   float64
		lowSeconds   float64
		logprobSum   float64
		answerWords  = map[int][]int{}
		sentences    int
		complexSents int
		sentWordSum  int
	)

	for _, a := range answers {
		words := a.Words
		if len(words) == 0 {
			answerWords[a.Part] = append(answerWords[a.Part], 0)
			continue
		}
		span := words[len(words)-1].End - words[0].Start
		f.SpeakingSeconds += span
		f.PartSeconds[partKey(a.Part)] += span
		if a.QuestionID == "p2" {
			f.LongTurnSeconds = span
		}

		boundaries := clauseBoundaries(a.Text, words)
		run, spoken := 0, 0
		var prev string
		for i, w := range words {
			tok := normalizeToken(w.Word)
			if tok == "" {
				continue
			}
			if hesitationFillers[tok] {
				fillers++
			} else {
				allTokens = append(allTokens, tok)
				spoken++
				if tok == prev {
					f.Repairs++
				}
				prev = tok
			}
			run++
			if i+1 < len(words) {
				gap := words[i+1].Start - w.End
				if gap >= pauseMin {
					pauses++
					pauseTotal += gap
					runs = append(runs, run)
					run = 0
					atBoundary := boundaries[i]
					if gap >= longPauseMin {
						longPauses++
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
			}
		}
		if run > 0 {
			runs = append(runs, run)
		}
		f.Repairs += repeatedBigrams(allTokens[len(allTokens)-spoken:])
		answerWords[a.Part] = append(answerWords[a.Part], spoken)

		for _, s := range a.Segments {
			d := s.End - s.Start
			segSeconds += d
			logprobSum += s.AvgLogprob * d
			if s.AvgLogprob < -0.8 {
				lowSeconds += d
			}
		}

		for _, sent := range splitSentences(a.Text) {
			toks := strings.Fields(sent)
			if len(toks) == 0 {
				continue
			}
			sentences++
			sentWordSum += len(toks)
			for _, t := range toks {
				if subordinators[normalizeToken(t)] {
					complexSents++
					break
				}
			}
		}
	}

	f.WordCount = len(allTokens)
	minutes := f.SpeakingSeconds / 60
	if minutes > 0 {
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

	ev.Coherence = coherenceMetrics(answers, answerWords)
	ev.Lexis = LexisMetrics{MATTR: round2(mattr(allTokens, 50)), DistinctWords: distinct(allTokens)}
	if sentences > 0 {
		ev.Grammar = GrammarMetrics{
			Sentences:            sentences,
			MeanSentenceWords:    round1(float64(sentWordSum) / float64(sentences)),
			ComplexSentenceShare: round2(float64(complexSents) / float64(sentences)),
		}
	}
	if segSeconds > 0 {
		ev.Recognition = RecognitionMetrics{
			MeanLogprob:        round2(logprobSum / segSeconds),
			LowConfidenceShare: round2(lowSeconds / segSeconds),
		}
	}
	return ev
}

func coherenceMetrics(answers []answerTranscript, answerWords map[int][]int) CoherenceMetrics {
	c := CoherenceMetrics{DiscourseMarkers: map[string]int{}, MeanAnswerWords: map[string]float64{}}
	total, most := 0, 0
	for _, a := range answers {
		text := " " + strings.Join(strings.Fields(strings.ToLower(stripPunct(a.Text))), " ") + " "
		for _, m := range discourseMarkers {
			if n := strings.Count(text, " "+m+" "); n > 0 {
				c.DiscourseMarkers[m] += n
				total += n
			}
		}
	}
	for _, n := range c.DiscourseMarkers {
		most = max(most, n)
	}
	c.DistinctMarkers = len(c.DiscourseMarkers)
	if total > 0 {
		c.MostUsedMarkerShare = round2(float64(most) / float64(total))
	}
	for part, counts := range answerWords {
		c.MeanAnswerWords[partKey(part)] = round1(meanInts(counts))
		if part == 1 || part == 3 {
			for _, n := range counts {
				if n < 12 {
					c.MinimalAnswers++
				}
			}
		}
	}
	return c
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

func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == '.' || r == '?' || r == '!' })
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

func normalizeToken(s string) string {
	return strings.ToLower(strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\''
	}))
}

func stripPunct(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, s)
}

func joinWords(words []llm.TranscriptWord, from, to int) string {
	from, to = max(from, 0), min(to, len(words))
	parts := make([]string, 0, to-from)
	for _, w := range words[from:to] {
		parts = append(parts, strings.TrimSpace(w.Word))
	}
	return strings.Join(parts, " ")
}

func partKey(part int) string { return string(rune('0' + part)) }

func meanInts(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0
	for _, x := range xs {
		s += x
	}
	return float64(s) / float64(len(xs))
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
func round2(x float64) float64 { return math.Round(x*100) / 100 }

// sortedKeys is for deterministic prompt output.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
