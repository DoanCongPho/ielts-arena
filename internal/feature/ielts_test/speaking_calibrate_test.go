package ielts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCalibrateCmd(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("I think it is a really good question and I would say that it depends ", 6)
	for name, text := range map[string]string{"p1.webm": long, "p2.webm": long + long + long, "p3.webm": long} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `[{"id":"sample-1","source":"test","bands":{"overall":6.5,"Lexical Resource":7},
	  "answers":[{"part":1,"question":"Where do you live?","audio":"p1.webm"},
	             {"part":2,"question":"Describe a book you enjoyed.","audio":"p2.webm"},
	             {"part":3,"question":"Why do people read less?","audio":"p3.webm"}]}]`
	path := filepath.Join(dir, "manifest.json")
	_ = os.WriteFile(path, []byte(manifest), 0o644)

	g := NewSpeakingGrader(nil, fakeASR{}, &fakePron{}, &fakeJudge{band: 7}, SpeakingGraderConfig{})
	var out strings.Builder
	code := RunCalibrateCmd(g, []string{path}, &out)

	if !strings.Contains(out.String(), "sample-1") || !strings.Contains(out.String(), "LR 7/7") {
		t.Errorf("report:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "overall within ±0.5: 1/1") {
		t.Errorf("overall agreement missing:\n%s", out.String())
	}
	if code != 0 {
		t.Errorf("exit code = %d with every sample in agreement", code)
	}
}
