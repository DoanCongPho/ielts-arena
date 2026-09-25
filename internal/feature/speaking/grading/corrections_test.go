package grading

import "testing"

func TestAnchoredCorrections(t *testing.T) {
	answers := []answerTranscript{
		{QuestionID: "p2", Text: "He go to the school of magic. He have two friend."},
		{QuestionID: "p3.q1", Text: "Because people use phone."},
	}
	grammar := []Correction{
		{QuestionID: "p2", Original: "He go", Correction: "He goes"},
		{QuestionID: "p2", Original: "two friend", Correction: "two friends"},
		{QuestionID: "p2", Original: "She were happy", Correction: "She was happy"}, // not said
		{QuestionID: "p3.q1", Original: "He go", Correction: "He goes"},             // wrong answer
		{QuestionID: "p2", Original: "he go", Correction: "he goes"},                // duplicate
		{QuestionID: "p2", Original: "magic", Correction: "magic"},                  // no change
	}
	vocab := []Correction{
		{QuestionID: "p3.q1", Original: "use phone", Correction: "use their phones"},
		{QuestionID: "p2", Original: "two friend", Correction: "a couple of friends"}, // already flagged
	}
	got := anchoredCorrections(answers, grammar, "grammar", vocab, "vocabulary")
	if len(got) != 3 {
		t.Fatalf("got %d corrections, want 3: %+v", len(got), got)
	}
	if got[0].Type != "grammar" || got[2].Type != "vocabulary" || got[2].Correction != "use their phones" {
		t.Errorf("corrections = %+v", got)
	}
}
