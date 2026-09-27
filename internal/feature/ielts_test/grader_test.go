package ielts_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github/DoanCongPho/game-arena/internal/platform/llm"
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

func TestJudgeSystemPrompt(t *testing.T) {
	ta := judgeSystemPrompt("task1", criterionTaskAchievement, true)
	for _, want := range []string{criterionTaskAchievement, "at least 150 words", "attached", "Vietnamese", `"checks"`, "3: The response does not address"} {
		if !strings.Contains(ta, want) {
			t.Errorf("Task Achievement prompt is missing %q", want)
		}
	}
	if strings.Contains(ta, criterionGrammar) {
		t.Error("a judge's prompt must carry only its own criterion")
	}

	// Word count and chart only matter to the content criterion.
	gra := judgeSystemPrompt("task1", criterionGrammar, false)
	if strings.Contains(gra, "at least 150 words") || strings.Contains(gra, "attached") {
		t.Error("the grammar judge must not be told about length or the chart")
	}

	tr := judgeSystemPrompt("task2", criterionTaskResponse, false)
	if !strings.Contains(tr, "at least 250 words") {
		t.Error("Task Response prompt is missing the Task 2 length")
	}
}

// fakeCompleter answers each grading job by its system prompt.
type fakeCompleter struct {
	mu     sync.Mutex
	images map[string]string               // job → image URL it was sent
	params map[string]llm.CompletionParams // job → what it was sent with
	fail   string                          // job to fail
	band   int
}

func (f *fakeCompleter) Complete(_ context.Context, system, _, imageURL string, params llm.CompletionParams) (string, error) {
	job := "judge"
	switch {
	case strings.Contains(system, "marking a learner"):
		job = "corrections"
	case strings.Contains(system, "model answer"):
		job = "model answer"
	default:
		for _, c := range []string{criterionTaskAchievement, criterionTaskResponse, criterionCoherence, criterionLexical, criterionGrammar} {
			if strings.Contains(system, "one criterion only: "+c) {
				job = c
			}
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.images[job] = imageURL
	f.params[job] = params
	if job == f.fail {
		return "", errors.New("boom")
	}
	switch job {
	case "corrections":
		return `{"corrections":[{"span":"people believes","issue":"grammar","suggestion":"people believe"}]}`, nil
	case "model answer":
		return `{"model_answer":"A model answer."}`, nil
	}
	return fmt.Sprintf(`{"checks":[],"band":%d,"feedback":"ok","improvements":["do x"]}`, f.band), nil
}

func TestGrade_RunsAJudgePerCriterion(t *testing.T) {
	f := &fakeCompleter{images: map[string]string{}, params: map[string]llm.CompletionParams{}, band: 6}
	g := NewOpenAIGrader(f, "judge-model", "review-model", nil)
	got, err := g.Grade(context.Background(), GradeInput{
		TaskType: "task1", Prompt: "Describe the chart.", ImageURL: "https://x/chart.png",
		Answer: "Many people believes the chart shows growth.", NeedModelAnswer: true,
	})
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if len(got.Criteria) != 4 || got.Criteria[criterionGrammar].Score != 6 || got.Criteria[criterionLexical].Improvements[0] != "do x" {
		t.Errorf("criteria = %+v", got.Criteria)
	}
	if len(got.Corrections) != 1 || got.ModelAnswer != "A model answer." {
		t.Errorf("corrections = %+v, model answer = %q", got.Corrections, got.ModelAnswer)
	}
	// The chart goes to the content judge and the model answer only.
	for job, want := range map[string]string{
		criterionTaskAchievement: "https://x/chart.png", "model answer": "https://x/chart.png",
		criterionCoherence: "", criterionLexical: "", criterionGrammar: "", "corrections": "",
	} {
		if f.images[job] != want {
			t.Errorf("%s got image %q, want %q", job, f.images[job], want)
		}
	}
	// Judges rate on the judge model; what learners read closely comes from
	// the review model, thinking kept low.
	for job, p := range f.params {
		want := "judge-model"
		if job == "corrections" || job == "model answer" {
			want = "review-model"
			if p.ReasoningEffort != "low" {
				t.Errorf("%s reasoning effort = %q, want low", job, p.ReasoningEffort)
			}
		}
		if p.Model != want {
			t.Errorf("%s used model %q, want %s", job, p.Model, want)
		}
	}
}

func TestGrade_SkipsModelAnswerWhenTheTestHasASample(t *testing.T) {
	f := &fakeCompleter{images: map[string]string{}, params: map[string]llm.CompletionParams{}, band: 7}
	if _, err := NewOpenAIGrader(f, "", "", nil).Grade(context.Background(), GradeInput{TaskType: "task2", Answer: "essay"}); err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if _, ok := f.images["model answer"]; ok {
		t.Error("a model answer was written although the test has a sample")
	}
}

func TestGrade_FailsWhenAnyJobFails(t *testing.T) {
	for _, job := range []string{criterionCoherence, "corrections"} {
		f := &fakeCompleter{images: map[string]string{}, params: map[string]llm.CompletionParams{}, band: 6, fail: job}
		if _, err := NewOpenAIGrader(f, "", "", nil).Grade(context.Background(), GradeInput{TaskType: "task2", Answer: "essay"}); err == nil {
			t.Errorf("expected an error when %s fails", job)
		}
	}
	f := &fakeCompleter{images: map[string]string{}, params: map[string]llm.CompletionParams{}, band: 10}
	if _, err := NewOpenAIGrader(f, "", "", nil).Grade(context.Background(), GradeInput{TaskType: "task2", Answer: "essay"}); err == nil {
		t.Error("expected an error for a band above 9")
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
