package ielts_test

import "testing"

func fullSpeakingContent() SpeakingContent {
	c := SpeakingContent{
		Part1: &SpeakingPart1{Topics: []SpeakingTopic{
			{Topic: "Your hometown", Questions: []SpeakingQuestion{{Text: "Where is your hometown?"}, {Text: "Do you like living there?"}}},
			{Topic: "Music", Questions: []SpeakingQuestion{{Text: "What music do you like?"}}},
		}},
		Part2: &SpeakingPart2{
			Topic:     "Describe a book you enjoyed reading.",
			Bullets:   []string{"what the book was", "when you read it", "what it was about"},
			Explain:   "and explain why you enjoyed it.",
			FollowUps: []SpeakingQuestion{{Text: "Do you often read?"}},
		},
		Part3: &SpeakingPart3{Questions: []SpeakingQuestion{
			{Text: "Why do people read less today?"}, {Text: "Should schools make reading compulsory?"}, {Text: "Will printed books disappear?"},
		}},
	}
	NormalizeSpeakingContent(&c)
	return c
}

func TestSpeakingContent_ModesAndValidation(t *testing.T) {
	c := fullSpeakingContent()
	if err := ValidateSpeakingContent(c); err != nil {
		t.Fatalf("full test rejected: %v", err)
	}
	if c.Mode() != SpeakingModeFull {
		t.Errorf("Mode() = %q, want full", c.Mode())
	}
	if got := c.Part1.Topics[1].Questions[0].ID; got != "p1.t2.q1" {
		t.Errorf("question id = %q, want p1.t2.q1", got)
	}

	twoParts := SpeakingContent{Part1: c.Part1, Part2: c.Part2}
	if err := ValidateSpeakingContent(twoParts); err == nil {
		t.Error("two-part content accepted; the real test has one or three")
	}

	part3Only := SpeakingContent{Part3: &SpeakingPart3{Questions: c.Part3.Questions}}
	if err := ValidateSpeakingContent(part3Only); err == nil {
		t.Error("Part 3 on its own without a theme accepted")
	}
	part3Only.Part3.Theme = "reading"
	if err := ValidateSpeakingContent(part3Only); err != nil {
		t.Errorf("Part 3 with a theme rejected: %v", err)
	}

	bad := fullSpeakingContent()
	bad.Part2.Bullets = bad.Part2.Bullets[:2]
	if err := ValidateSpeakingContent(bad); err == nil {
		t.Error("cue card with two bullet points accepted")
	}
}
