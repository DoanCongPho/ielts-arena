package ielts_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion"`
}

type Grader interface {
	Grade(ctx context.Context, skill, taskType, prompt, imageURL, answer string) (*GradingResult, error)
}

// gradingParams is deliberately low-temperature: grading should be
// consistent across runs rather than creative, and the JSON response has a
// bounded, predictable size.
var gradingParams = llm.CompletionParams{
	Temperature: 0.2,
	MaxTokens:   1024,
}

type openAIGrader struct {
	client *llm.Client
}

func NewOpenAIGrader(client *llm.Client) Grader {
	return &openAIGrader{client: client}
}

func (g *openAIGrader) Grade(ctx context.Context, skill, taskType, prompt, imageURL, answer string) (*GradingResult, error) {
	system := buildSystemPrompt(skill, taskType, imageURL != "")
	user := fmt.Sprintf("Task prompt:\n%s\n\nCandidate answer:\n%s", prompt, answer)

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

func buildSystemPrompt(skill, taskType string, hasImage bool) string {
	criteria := writingCriteria
	if skill == "speaking" {
		criteria = speakingCriteria
	}
	imageNote := ""
	if hasImage {
		imageNote = "\nAn image of the chart/graph/diagram is attached — use it to verify the candidate accurately described the data before scoring Task Achievement.\n"
	}
	return fmt.Sprintf(`You are an expert IELTS %s examiner grading a %s response.
Score each criterion on the official IELTS 0–9 band scale.
The overall_band is the mean of all criteria scores, rounded to the nearest 0.5.
%s
Respond with ONLY valid JSON in this exact schema:
{
  "overall_band": <number>,
  "criteria": {
    %s
  },
  "model_answer": "<a model answer for this task>",
  "corrections": [
    { "span": "<exact text from answer>", "issue": "<error type>", "suggestion": "<corrected version>" }
  ]
}`, skill, taskType, imageNote, criteria)
}

const writingCriteria = `
    "Task Achievement":              { "score": <0-9>, "feedback": "<string>" },
    "Coherence and Cohesion":        { "score": <0-9>, "feedback": "<string>" },
    "Lexical Resource":              { "score": <0-9>, "feedback": "<string>" },
    "Grammatical Range and Accuracy":{ "score": <0-9>, "feedback": "<string>" }`

const speakingCriteria = `
    "Fluency and Coherence":         { "score": <0-9>, "feedback": "<string>" },
    "Lexical Resource":              { "score": <0-9>, "feedback": "<string>" },
    "Grammatical Range and Accuracy":{ "score": <0-9>, "feedback": "<string>" },
    "Pronunciation":                 { "score": <0-9>, "feedback": "<string>" }`

var _ Grader = (*openAIGrader)(nil)
