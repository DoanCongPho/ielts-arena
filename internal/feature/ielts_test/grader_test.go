package ielts_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gradeWith builds a Task 2 grade scoring the four criteria as given.
func gradeWith(tr, cc, lr, gra float64) *GradingResult {
	return &GradingResult{Criteria: map[string]CriterionScore{
		criterionTaskResponse: {Score: tr},
		criterionCoherence:    {Score: cc},
		criterionLexical:      {Score: lr},
		criterionGrammar:      {Score: gra},
	}}
}

func TestNormalizeResult_OverallBandUsesIELTSRounding(t *testing.T) {
	cases := []struct {
		name  string
		grade *GradingResult
		want  float64
	}{
		{"mean 6.125 rounds down", gradeWith(6, 6, 6, 6.5), 6},
		{"mean 6.25 rounds up to the half", gradeWith(6, 6, 6.5, 6.5), 6.5},
		{"mean 6.75 rounds up to the whole", gradeWith(6.5, 7, 7, 6.5), 7},
		{"model's own overall is ignored", &GradingResult{OverallBand: 9, Criteria: gradeWith(5, 5, 5, 5).Criteria}, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := normalizeResult(tc.grade, "task2", "essay"); err != nil {
				t.Fatalf("normalizeResult: %v", err)
			}
			if tc.grade.OverallBand != tc.want {
				t.Errorf("OverallBand = %v, want %v", tc.grade.OverallBand, tc.want)
			}
		})
	}
}

func TestNormalizeResult_ClampsAndSnapsScores(t *testing.T) {
	g := gradeWith(9.8, -1, 6.3, 6.8)
	if err := normalizeResult(g, "task2", "essay"); err != nil {
		t.Fatalf("normalizeResult: %v", err)
	}
	want := map[string]float64{criterionTaskResponse: 9, criterionCoherence: 0, criterionLexical: 6.5, criterionGrammar: 7}
	for name, score := range want {
		if got := g.Criteria[name].Score; got != score {
			t.Errorf("%s = %v, want %v", name, got, score)
		}
	}
}

func TestNormalizeResult_RequiresTheTasksCriteria(t *testing.T) {
	// A Task 2 grade scored under Task 1's first criterion is incomplete.
	g := gradeWith(7, 7, 7, 7)
	if err := normalizeResult(g, "task1", "essay"); err == nil {
		t.Fatal("expected an error for a grade missing Task Achievement")
	}

	// Extra criteria the model invented are dropped.
	g = gradeWith(7, 7, 7, 7)
	g.Criteria["Creativity"] = CriterionScore{Score: 9}
	if err := normalizeResult(g, "task2", "essay"); err != nil {
		t.Fatalf("normalizeResult: %v", err)
	}
	if _, ok := g.Criteria["Creativity"]; ok || len(g.Criteria) != 4 {
		t.Errorf("criteria = %v, want exactly the four Task 2 criteria", g.Criteria)
	}
}

func TestNormalizeResult_KeepsOnlyCorrectionsFoundInTheAnswer(t *testing.T) {
	answer := "Nowadays  many people\nbelieves that cities are crowded."
	g := gradeWith(6, 6, 6, 6)
	g.Corrections = []Correction{
		{Span: "many people believes", Issue: "Grammar", Suggestion: "many people believe"}, // whitespace differs
		{Span: "not in the essay", Issue: "grammar", Suggestion: "x"},
		{Span: "  ", Issue: "grammar", Suggestion: "x"},
		{Span: "crowded", Issue: "style", Suggestion: "congested"},
	}
	if err := normalizeResult(g, "task2", answer); err != nil {
		t.Fatalf("normalizeResult: %v", err)
	}
	if len(g.Corrections) != 2 {
		t.Fatalf("kept %d corrections, want 2: %+v", len(g.Corrections), g.Corrections)
	}
	if g.Corrections[0].Issue != "grammar" {
		t.Errorf("Issue = %q, want it lower-cased to grammar", g.Corrections[0].Issue)
	}
	if g.Corrections[1].Issue != "other" {
		t.Errorf("Issue = %q, want an unknown category mapped to other", g.Corrections[1].Issue)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	t1 := buildSystemPrompt("task1", true, false)
	for _, want := range []string{criterionTaskAchievement, "at least 150 words", "attached", "Vietnamese"} {
		if !strings.Contains(t1, want) {
			t.Errorf("Task 1 prompt is missing %q", want)
		}
	}
	if strings.Contains(t1, criterionTaskResponse) || strings.Contains(t1, "model_answer") {
		t.Error("Task 1 prompt with a stored sample must not ask for Task Response or a model answer")
	}

	t2 := buildSystemPrompt("task2", false, true)
	for _, want := range []string{criterionTaskResponse, "at least 250 words", `"model_answer"`} {
		if !strings.Contains(t2, want) {
			t.Errorf("Task 2 prompt is missing %q", want)
		}
	}
}

func TestAssetImageResolver(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "writing_charts"), 0o755); err != nil {
		t.Fatal(err)
	}
	png := []byte("\x89PNG fake")
	if err := os.WriteFile(filepath.Join(dir, "writing_charts", "a.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	local := NewAssetImageResolver(dir, nil)
	got, err := local(ctx, "/assets/writing_charts/a.png")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got, _ := local(ctx, "https://example.com/c.png"); got != "https://example.com/c.png" {
		t.Errorf("an absolute URL must pass through, got %q", got)
	}
	if _, err := local(ctx, "/assets/../../etc/passwd"); err == nil {
		t.Error("expected a path outside the assets directory to be refused")
	}

	bucket := NewAssetImageResolver(dir, func(key string) (string, error) { return "https://bucket/" + key + "?sig", nil })
	if got, _ := bucket(ctx, "/assets/writing_charts/a.png"); got != "https://bucket/writing_charts/a.png?sig" {
		t.Errorf("with a bucket, got %q", got)
	}
}
