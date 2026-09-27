package ielts_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github/DoanCongPho/game-arena/internal/platform/llm"
)

type CriterionScore struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
	// Improvements are concrete steps towards the next band, in Vietnamese.
	Improvements []string `json:"improvements,omitempty"`
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

type completer interface {
	Complete(ctx context.Context, system, user, imageURL string, params llm.CompletionParams) (string, error)
}

// Grading is split into one call per job, run in parallel: a judge per
// criterion, a correction pass and, when the test has no sample, a model
// answer. One prompt doing all of it spreads the model's attention thin,
// and its scores drift towards the middle of the scale. Judging and
// correcting are deliberately low-temperature, to be consistent across
// runs rather than creative.
var (
	judgeParams       = llm.CompletionParams{Temperature: 0.1, MaxTokens: 3000}
	correctionParams  = llm.CompletionParams{Temperature: 0.1, MaxTokens: 5000}
	modelAnswerParams = llm.CompletionParams{Temperature: 0.4, MaxTokens: 1500}
)

// maxWritingCorrections bounds the correction list. It is high enough to
// cover a weak Task 2 essay, where a dozen would leave most errors unmarked.
const maxWritingCorrections = 25

// criterionVerdict is one judge's answer. Checks come first in the schema
// so the model weighs the evidence before it commits to a band.
type criterionVerdict struct {
	Checks       []descriptorCheck `json:"checks"`
	Band         int               `json:"band"`
	Feedback     string            `json:"feedback"`
	Improvements []string          `json:"improvements"`
}

type descriptorCheck struct {
	Band     int    `json:"band"`
	Feature  string `json:"feature"`
	Verdict  string `json:"verdict"` // met | partly | not_met
	Evidence string `json:"evidence"`
}

type openAIGrader struct {
	client       completer
	model        string
	resolveImage ImageResolver
}

// NewOpenAIGrader grades writing answers with model through client; an
// empty model means the client's default. resolveImage may be nil when
// every image_url is already absolute.
func NewOpenAIGrader(client completer, model string, resolveImage ImageResolver) Grader {
	return &openAIGrader{client: client, model: model, resolveImage: resolveImage}
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		result   = GradingResult{Criteria: map[string]CriterionScore{}}
	)
	// run starts one grading job; the first failure cancels the others,
	// since the whole grade is retried anyway.
	run := func(name string, job func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := job(); err != nil {
				mu.Lock()
				defer mu.Unlock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", name, err)
					cancel()
				}
			}
		}()
	}

	criteria := writingCriteria(in.TaskType)
	for i, criterion := range criteria {
		// Only the first criterion (Task Achievement / Response) is about
		// the content, so only its judge needs to see the chart.
		image := ""
		if i == 0 {
			image = imageURL
		}
		run(criterion, func() error {
			score, err := g.judge(ctx, criterion, in, image != "", image)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			result.Criteria[criterion] = *score
			return nil
		})
	}
	run("corrections", func() error {
		corrections, err := g.correct(ctx, in)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		result.Corrections = corrections
		return nil
	})
	if in.NeedModelAnswer {
		run("model answer", func() error {
			answer, err := g.writeModelAnswer(ctx, in, imageURL)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			result.ModelAnswer = answer
			return nil
		})
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return &result, nil
}

func (g *openAIGrader) complete(ctx context.Context, system, user, imageURL string, params llm.CompletionParams, out any) error {
	params.Model = g.model
	raw, err := g.client.Complete(ctx, system, user, imageURL, params)
	if err != nil {
		return fmt.Errorf("llm: %w", err)
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func (g *openAIGrader) judge(ctx context.Context, criterion string, in GradeInput, hasImage bool, imageURL string) (*CriterionScore, error) {
	var v criterionVerdict
	if err := g.complete(ctx, judgeSystemPrompt(in.TaskType, criterion, hasImage), answerPrompt(in), imageURL, judgeParams, &v); err != nil {
		return nil, err
	}
	if v.Band < 0 || v.Band > 9 {
		return nil, fmt.Errorf("band %d is not a whole band from 0 to 9", v.Band)
	}
	return &CriterionScore{Score: float64(v.Band), Feedback: v.Feedback, Improvements: v.Improvements}, nil
}

func (g *openAIGrader) correct(ctx context.Context, in GradeInput) ([]Correction, error) {
	var out struct {
		Corrections []Correction `json:"corrections"`
	}
	if err := g.complete(ctx, correctionSystemPrompt(in.TaskType), answerPrompt(in), "", correctionParams, &out); err != nil {
		return nil, err
	}
	return out.Corrections, nil
}

func (g *openAIGrader) writeModelAnswer(ctx context.Context, in GradeInput, imageURL string) (string, error) {
	var out struct {
		ModelAnswer string `json:"model_answer"`
	}
	user := "Task prompt:\n" + in.Prompt
	if err := g.complete(ctx, modelAnswerSystemPrompt(in.TaskType, imageURL != ""), user, imageURL, modelAnswerParams, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.ModelAnswer) == "" {
		return "", errors.New("empty model answer")
	}
	return out.ModelAnswer, nil
}

// answerPrompt is the user message every judge and the correction pass read.
func answerPrompt(in GradeInput) string {
	return fmt.Sprintf("Task prompt:\n%s\n\nCandidate answer (%d words):\n%s", in.Prompt, countWords(in.Answer), in.Answer)
}

func taskName(taskType string) string {
	if taskType == "task1" {
		return "Writing Task 1 (a report on visual information)"
	}
	return "Writing Task 2 (an essay)"
}

func judgeSystemPrompt(taskType, criterion string, hasImage bool) string {
	var notes strings.Builder
	if criterion == writingCriteria(taskType)[0] {
		if words, ok := minWords[taskType]; ok {
			fmt.Fprintf(&notes, "\nThe task asks for at least %d words. An answer under that length is penalised under this criterion, since it cannot fully cover the task.", words)
		}
		if hasImage {
			notes.WriteString("\nThe task's chart/graph/diagram is attached. Check every figure, trend and comparison the candidate reports against it: inaccurate data counts against this criterion.")
		}
	}
	return fmt.Sprintf(`You are a certified IELTS Writing examiner. You are rating an Academic %s on one criterion only: %s.

%s

Band descriptors for %s:%s
%s
Work in this order:
1. For each band from 9 down to 4 (and lower if needed), check that band's key features against the answer: "met", "partly" or "not_met", each with a short quote from the answer as evidence.
2. Pick the best-fit whole band.
3. Write feedback in Vietnamese, addressed to the learner: two to four sentences specific to this answer, quoting their own English words, saying what earns the band and what holds it back from the next one.
4. Give two or three improvements in Vietnamese: concrete things to do to reach the next band, using an example from their answer where you can.

Respond with ONLY valid JSON:
{
  "checks": [ { "band": <int>, "feature": "<descriptor feature>", "verdict": "met|partly|not_met", "evidence": "<quote>" } ],
  "band": <whole number 0-9>,
  "feedback": "<string>",
  "improvements": ["<string>"]
}`, taskName(taskType), criterion, examinerPrinciples, criterion, bandDescriptors[criterion], notes.String())
}

func correctionSystemPrompt(taskType string) string {
	return fmt.Sprintf(`You are an experienced IELTS Writing teacher marking a learner's Academic %s for language errors.

Mark every real error and every clearly unnatural expression a band 8 writer would not produce. Don't mark stylistic preferences in language that is already correct and natural.
- "span": the shortest stretch of the answer that contains the error, copied character for character, with enough words around it to be unambiguous (e.g. "many people believes", not "believes").
- "issue": one of %s.
- "suggestion": the corrected English text that replaces exactly the span.
- "explanation": one short sentence in Vietnamese saying why it is wrong.
Mark each error once, and don't let spans overlap. List them in the order they appear in the answer. If there are more than %d, keep the %d that matter most: errors that obscure meaning first, then ones the learner repeats.

Respond with ONLY valid JSON:
{
  "corrections": [
    { "span": "<exact text from the answer>", "issue": "<category>", "suggestion": "<corrected text>", "explanation": "<why, in Vietnamese>" }
  ]
}`, taskName(taskType), strings.Join(correctionIssues, ", "), maxWritingCorrections, maxWritingCorrections)
}

func modelAnswerSystemPrompt(taskType string, hasImage bool) string {
	length := "about 280 words"
	if taskType == "task1" {
		length = "about 180 words, with a clear overview of the main trends"
	}
	if hasImage {
		length += ". The chart/graph/diagram is attached: every figure you report must match it"
	}
	return fmt.Sprintf(`You are an IELTS Writing examiner. Write a band 8-9 model answer to this Academic %s, %s.

Respond with ONLY valid JSON:
{ "model_answer": "<the answer in English, paragraphs separated by \\n\\n>" }`, taskName(taskType), length)
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
	dropped := 0
	for _, c := range r.Corrections {
		span := collapseSpace(c.Span)
		if span == "" || !strings.Contains(haystack, span) {
			dropped++
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
	if dropped > 0 {
		// A span the model didn't copy exactly can't be marked in the
		// answer. Many of these mean the model is misquoting the learner.
		log.Printf("ielts_test: dropped %d of %d writing corrections whose span is not in the answer", dropped, len(r.Corrections))
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
