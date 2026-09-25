package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"strings"
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

// extractEvidence measures a whole test's answers.
func extractEvidence(answers []answerTranscript) Evidence {
	var ev Evidence
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
