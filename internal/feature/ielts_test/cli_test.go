package ielts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestBody(t *testing.T, body any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.json")
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunValidateCmd(t *testing.T) {
	valid := CreateTestRequest{
		Skill: "reading", TaskType: "test1", Series: "cambridge", Volume: 20, TestNumber: 1,
		ContentData: marshalContent(t, readingWithEvidence(Evidence{Paragraph: 0, Quote: "The cat"})),
	}
	if code := RunValidateCmd([]string{writeTestBody(t, valid)}); code != 0 {
		t.Errorf("valid body: exit %d, want 0", code)
	}

	invalid := valid
	invalid.ContentData = marshalContent(t, readingWithEvidence(Evidence{Paragraph: 0, Quote: "not there"}))
	if code := RunValidateCmd([]string{writeTestBody(t, invalid)}); code != 1 {
		t.Errorf("bad evidence: exit %d, want 1", code)
	}

	unknownField := map[string]any{"skill": "reading", "task_type": "test1", "titel": "typo"}
	if code := RunValidateCmd([]string{writeTestBody(t, unknownField)}); code != 1 {
		t.Errorf("unknown field: exit %d, want 1", code)
	}

	if code := RunValidateCmd(nil); code != 2 {
		t.Errorf("no args: exit %d, want 2", code)
	}
}
