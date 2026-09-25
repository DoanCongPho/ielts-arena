package examiner

import (
	"github/DoanCongPho/game-arena/internal/feature/speaking/speakingtest"
	"strings"
	"testing"
)

func TestBuildSpeakingScript(t *testing.T) {
	lines := Build(speakingtest.FullContent(), defaultVoice)

	var kinds []string
	for _, l := range lines {
		kinds = append(kinds, l.Kind)
		if l.AudioKey == "" || !strings.HasPrefix(l.AudioKey, "examiner/") {
			t.Errorf("line %q has audio key %q", l.Text, l.AudioKey)
		}
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, "say,cue_card,long_turn,say,ask") {
		t.Errorf("Part 2 sequence wrong: %s", joined)
	}
	var leadIn, closing string
	for _, l := range lines {
		if l.Part == 3 && l.Kind == LineSay && leadIn == "" {
			leadIn = l.Text
		}
		closing = l.Text
	}
	if !strings.Contains(leadIn, "talking about a book you enjoyed reading") {
		t.Errorf("Part 3 lead-in = %q", leadIn)
	}
	if closing != "Thank you. That is the end of the speaking test." {
		t.Errorf("closing = %q", closing)
	}

	// Identical lines share one recording across tests.
	again := Build(speakingtest.FullContent(), defaultVoice)
	if again[0].AudioKey != lines[0].AudioKey {
		t.Error("audio key is not deterministic")
	}
	other := defaultVoice
	other.Voice = "alloy"
	if Build(speakingtest.FullContent(), other)[0].AudioKey == lines[0].AudioKey {
		t.Error("audio key ignores the voice")
	}
}
