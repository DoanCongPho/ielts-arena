package grading

import (
	"fmt"
	"strings"
)

// Correction limits: enough to show the pattern of errors without
// burying the answer.
const (
	maxCorrectionsPerAnswer = 6
	maxCorrections          = 40
)

// anchoredCorrections keeps only corrections whose original text really
// occurs in the answer they name — a judge that misquotes would otherwise
// mark text the candidate never said — deduplicated and capped. Grammar
// ones come first: grammar and vocabulary judges sometimes flag the same
// span, and the first one wins.
func anchoredCorrections(answers []answerTranscript, grammar []Correction, gType string, vocab []Correction, vType string) []Correction {
	text := map[string]string{}
	for _, a := range answers {
		text[a.QuestionID] = strings.ToLower(a.Text)
	}
	out := []Correction{}
	perAnswer := map[string]int{}
	seen := map[string]bool{}
	add := func(list []Correction, typ string) {
		for _, c := range list {
			c.Type = typ
			c.Original = strings.TrimSpace(c.Original)
			c.Correction = strings.TrimSpace(c.Correction)
			key := c.QuestionID + "\x00" + strings.ToLower(c.Original)
			switch {
			case c.Original == "" || strings.EqualFold(c.Original, c.Correction):
			case !strings.Contains(text[c.QuestionID], strings.ToLower(c.Original)):
			case seen[key], perAnswer[c.QuestionID] >= maxCorrectionsPerAnswer, len(out) >= maxCorrections:
			default:
				seen[key] = true
				perAnswer[c.QuestionID]++
				out = append(out, c)
			}
		}
	}
	add(grammar, gType)
	add(vocab, vType)
	return out
}

// correctionTask asks the grammar and vocabulary judges to list the errors
// they rated, for the review to mark on the transcript.
func correctionTask(criterion string) string {
	kind := map[string]string{
		CriterionGRA: "grammar error (tense, agreement, articles, plurals, word order, prepositions, sentence structure)",
		CriterionLR:  "word-choice error (wrong word, wrong collocation, wrong word form, unnatural phrasing)",
	}[criterion]
	if kind == "" {
		return ""
	}
	return fmt.Sprintf(`4. List each %s in the candidate's answers, up to 6 per answer, most important first:
   - "original": the shortest span that contains the error, copied EXACTLY from that answer's transcript (same words, same spelling), with enough words to be unambiguous;
   - "correction": that span rewritten correctly, changing as little as possible;
   - "explanation": one short sentence on the rule.
   Ignore fillers, hesitations, repetitions and self-corrections (they belong to fluency), and words that are clearly speech-recognition mistakes.
`, kind)
}

func correctionField(criterion string) string {
	if criterion != CriterionGRA && criterion != CriterionLR {
		return ""
	}
	return `,
  "corrections": [ { "question_id": "<id>", "original": "<exact text from the answer>", "correction": "<fixed text>", "explanation": "<rule>" } ]`
}
