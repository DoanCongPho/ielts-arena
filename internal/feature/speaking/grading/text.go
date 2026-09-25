package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"math"
	"strings"
	"unicode"
)

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
