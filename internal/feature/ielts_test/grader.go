package ielts_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github/DoanCongPho/game-arena/internal/platform/llm"
)

type CriterionScore struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
}

type GradingResult struct {
	OverallBand float64                   `json:"overall_band"`
	Criteria    map[string]CriterionScore `json:"criteria"`
	ModelAnswer string                    `json:"model_answer"`
	Corrections []Correction              `json:"corrections"`
}

type Correction struct {
	Span       string `json:"span"`
	Issue      string `json:"issue"` // one of correctionIssues, or "other"
	Suggestion string `json:"suggestion"`
	// Explanation says, in Vietnamese, why the span is wrong.
	Explanation string `json:"explanation,omitempty"`
}

// GradeInput is one writing answer to grade and the task it answers.
type GradeInput struct {
	TaskType string // task1 | task2
	Prompt   string
	ImageURL string // the Task 1 chart; empty when there is none
	Answer   string
	// NeedModelAnswer asks the grader to write a model answer, for tests
	// that don't store a sample answer of their own.
	NeedModelAnswer bool
}

type Grader interface {
	Grade(ctx context.Context, in GradeInput) (*GradingResult, error)
}

// ImageResolver turns a test's image_url into one the LLM can fetch.
type ImageResolver func(ctx context.Context, url string) (string, error)

// gradingParams is deliberately low-temperature: grading should be
// consistent across runs rather than creative. MaxTokens leaves room for
// the feedback, a dozen corrections and a model answer — a response cut
// off by the limit is invalid JSON and costs a retry.
var gradingParams = llm.CompletionParams{
	Temperature: 0.2,
	MaxTokens:   3000,
}

// maxWritingCorrections bounds the correction list, so the feedback stays on the
// errors that matter most rather than every slip.
const maxWritingCorrections = 12

type openAIGrader struct {
	client       *llm.Client
	resolveImage ImageResolver
}

// NewOpenAIGrader grades writing answers with the LLM behind client.
// resolveImage may be nil when every image_url is already absolute.
func NewOpenAIGrader(client *llm.Client, resolveImage ImageResolver) Grader {
	return &openAIGrader{client: client, resolveImage: resolveImage}
}

func (g *openAIGrader) Grade(ctx context.Context, in GradeInput) (*GradingResult, error) {
	imageURL := in.ImageURL
	if imageURL != "" && g.resolveImage != nil {
		resolved, err := g.resolveImage(ctx, imageURL)
		if err != nil {
			return nil, fmt.Errorf("resolve task image: %w", err)
		}
		imageURL = resolved
	}

	system := buildSystemPrompt(in.TaskType, imageURL != "", in.NeedModelAnswer)
	user := fmt.Sprintf("Task prompt:\n%s\n\nCandidate answer (%d words):\n%s", in.Prompt, countWords(in.Answer), in.Answer)

	raw, err := g.client.Complete(ctx, system, user, imageURL, gradingParams)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	var result GradingResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse grading response: %w", err)
	}
	return &result, nil
}

func buildSystemPrompt(taskType string, hasImage, needModelAnswer bool) string {
	criteria := writingCriteria(taskType)
	task := "Writing Task 2 (an essay)"
	if taskType == "task1" {
		task = "Writing Task 1 (a report on visual information)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "You are an experienced IELTS examiner grading an Academic %s.\n", task)
	b.WriteString("Score each criterion on the official 0–9 band scale in steps of 0.5, judging strictly against these band descriptors:\n")
	for _, name := range criteria {
		fmt.Fprintf(&b, "\n%s:%s\n", name, bandDescriptors[name])
	}
	if words, ok := minWords[taskType]; ok {
		fmt.Fprintf(&b, "\nThe task asks for at least %d words. An answer under that length cannot score above 5 for %s.\n", words, criteria[0])
	}
	if hasImage {
		fmt.Fprintf(&b, "\nThe task's chart/graph/diagram is attached: check the candidate's figures and trends against it when scoring %s.\n", criteria[0])
	}
	fmt.Fprintf(&b, `
Write every "feedback" and "explanation" in Vietnamese, addressed to the learner, naming concrete strengths and what to fix to reach the next band.
List up to %d corrections, most important first. "span" must be copied exactly from the candidate's answer; "suggestion" is the corrected English text; "issue" is one of: %s.
`, maxWritingCorrections, strings.Join(correctionIssues, ", "))

	modelAnswer := ""
	if needModelAnswer {
		modelAnswer = "\n  \"model_answer\": \"<an English band-8+ answer to this task, paragraphs separated by \\n\\n>\","
		b.WriteString("Also write a model answer of about the required length.\n")
	}

	var schema strings.Builder
	for i, name := range criteria {
		sep := ","
		if i == len(criteria)-1 {
			sep = ""
		}
		fmt.Fprintf(&schema, "\n    %q: { \"score\": <0-9>, \"feedback\": \"<string>\" }%s", name, sep)
	}
	fmt.Fprintf(&b, `
Respond with ONLY valid JSON in this exact schema:
{
  "criteria": {%s
  },%s
  "corrections": [
    { "span": "<exact text from the answer>", "issue": "<category>", "suggestion": "<corrected text>", "explanation": "<why, in Vietnamese>" }
  ]
}`, schema.String(), modelAnswer)
	return b.String()
}

// normalizeResult makes an LLM grade safe to store: it keeps exactly the
// task's four criteria with scores clamped to 0–9 in 0.5 steps, computes
// the overall band itself (the model is unreliable at arithmetic), and
// drops corrections that don't point at text in the answer. A missing
// criterion is an error — worth retrying, since another sample usually
// has it.
func normalizeResult(r *GradingResult, taskType, answer string) error {
	criteria := make(map[string]CriterionScore, 4)
	bands := make([]float64, 0, 4)
	for _, name := range writingCriteria(taskType) {
		c, ok := r.Criteria[name]
		if !ok {
			return fmt.Errorf("grading response is missing criterion %q", name)
		}
		c.Score = math.Round(min(max(c.Score, 0), 9)*2) / 2
		criteria[name] = c
		bands = append(bands, c.Score)
	}
	r.Criteria = criteria
	r.OverallBand = IELTSOverall(bands)

	haystack := collapseSpace(answer)
	kept := r.Corrections[:0]
	for _, c := range r.Corrections {
		span := collapseSpace(c.Span)
		if span == "" || !strings.Contains(haystack, span) {
			continue
		}
		c.Issue = strings.ToLower(strings.TrimSpace(c.Issue))
		if !isCorrectionIssue(c.Issue) {
			c.Issue = "other"
		}
		kept = append(kept, c)
		if len(kept) == maxWritingCorrections {
			break
		}
	}
	r.Corrections = kept
	return nil
}

func isCorrectionIssue(s string) bool {
	for _, issue := range correctionIssues {
		if s == issue {
			return true
		}
	}
	return false
}

// countWords counts whitespace-separated words, as the attempt page does.
func countWords(s string) int {
	return len(strings.Fields(s))
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// NewAssetImageResolver resolves the app's own /assets/... image paths,
// which the LLM can't fetch: with presign set (a bucket in production) it
// hands over a short-lived bucket link, otherwise it inlines the file from
// dir as a data: URL. Absolute http(s) URLs pass through unchanged.
func NewAssetImageResolver(dir string, presign func(key string) (string, error)) ImageResolver {
	return func(_ context.Context, url string) (string, error) {
		key, ok := strings.CutPrefix(url, "/assets/")
		if !ok {
			return url, nil
		}
		if presign != nil {
			return presign(key)
		}
		path := filepath.Join(dir, filepath.FromSlash(key))
		if rel, err := filepath.Rel(dir, path); err != nil || strings.HasPrefix(rel, "..") {
			return "", errors.New("image path escapes the assets directory")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		if mimeType == "" {
			mimeType = "image/png"
		}
		return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
	}
}

var _ Grader = (*openAIGrader)(nil)
