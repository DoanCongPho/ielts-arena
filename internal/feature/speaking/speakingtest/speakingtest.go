// Package speakingtest holds fixtures shared by the speaking packages'
// tests. Production code never imports it.
package speakingtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
)

// FullContent is a normalized three-part test: two Part 1 topics, a cue
// card with a rounding-off question, and three Part 3 questions.
func FullContent() ielts_test.SpeakingContent {
	c := ielts_test.SpeakingContent{
		Part1: &ielts_test.SpeakingPart1{Topics: []ielts_test.SpeakingTopic{
			{Topic: "Your hometown", Questions: []ielts_test.SpeakingQuestion{{Text: "Where is your hometown?"}, {Text: "Do you like living there?"}}},
			{Topic: "Music", Questions: []ielts_test.SpeakingQuestion{{Text: "What music do you like?"}}},
		}},
		Part2: &ielts_test.SpeakingPart2{
			Topic:     "Describe a book you enjoyed reading.",
			Bullets:   []string{"what the book was", "when you read it", "what it was about"},
			Explain:   "and explain why you enjoyed it.",
			FollowUps: []ielts_test.SpeakingQuestion{{Text: "Do you often read?"}},
		},
		Part3: &ielts_test.SpeakingPart3{Questions: []ielts_test.SpeakingQuestion{
			{Text: "Why do people read less today?"}, {Text: "Should schools make reading compulsory?"}, {Text: "Will printed books disappear?"},
		}},
	}
	ielts_test.NormalizeSpeakingContent(&c)
	return c
}

// MustJSON marshals v or fails the test.
func MustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// IndentJSON is v as indented JSON with a trailing newline, for golden files.
func IndentJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

// CheckGolden compares got with testdata/golden/NAME, or writes it there
// when UPDATE_GOLDEN=1. Golden files pin output a refactor must not change.
func CheckGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	dir := filepath.Join("testdata", "golden")
	path := filepath.Join(dir, name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (UPDATE_GOLDEN=1 creates it)", path, err)
	}
	if string(want) != string(got) {
		_ = os.WriteFile(path+".got", got, 0o644)
		t.Errorf("%s differs from its golden file; compare %s with %s.got", name, path, path)
	}
}
