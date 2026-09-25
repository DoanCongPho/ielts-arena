package grading

import (
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
	"testing"
)

func TestSummarizePronunciation(t *testing.T) {
	s := summarizePronunciation([]clipAssessment{
		{QuestionID: "p2", Seconds: 90, A: &pronunciation.Assessment{
			Intelligibility: 0.9, Prosody: pronunciation.Prosody{PitchStdST: 3},
			Words: []pronunciation.Word{{Word: "think", Score: 40, Phonemes: []pronunciation.Phoneme{{Expected: "θ", Heard: "t", Score: 20}}}, {Word: "book", Score: 90}},
		}},
		{QuestionID: "p3.q1", Seconds: 30, A: &pronunciation.Assessment{
			Intelligibility: 0.5, Prosody: pronunciation.Prosody{PitchStdST: 1},
			Words: []pronunciation.Word{{Word: "Think", Score: 55, Phonemes: []pronunciation.Phoneme{{Expected: "θ", Heard: "t", Score: 30}}}, {Word: "uh", Score: 0}},
		}},
	})
	if s.Intelligibility != 0.8 { // (0.9*90 + 0.5*30) / 120
		t.Errorf("Intelligibility = %v, want 0.8", s.Intelligibility)
	}
	if s.Prosody.PitchStdST != 2.5 {
		t.Errorf("PitchStdST = %v, want 2.5", s.Prosody.PitchStdST)
	}
	if len(s.Mispronounced) != 1 || s.Mispronounced[0].Count != 2 || s.Mispronounced[0].Expected != "θ" {
		t.Errorf("Mispronounced = %+v", s.Mispronounced)
	}
	if got := s.Words["p2"]; len(got) != 1 || got[0].Index != 0 || got[0].Expected != "θ" {
		t.Errorf("Words[p2] = %+v (book has no phonemes, so only think is kept)", got)
	}
}
