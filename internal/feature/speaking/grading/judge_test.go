package grading

import (
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"strings"
	"testing"
)

func TestJudgePrompts_AskForCorrectionsFromGrammarAndVocabularyOnly(t *testing.T) {
	for _, c := range criteria {
		p := judgeSystemPrompt(c, ielts_test.SpeakingModeFull)
		want := c == CriterionGRA || c == CriterionLR
		if got := strings.Contains(p, `"corrections"`); got != want {
			t.Errorf("%s prompt asks for corrections = %v, want %v", c, got, want)
		}
	}
}
