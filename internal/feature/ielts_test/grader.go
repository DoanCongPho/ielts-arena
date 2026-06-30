package ielts_test

import (
	"context"
	"github/DoanCongPho/game-arena/internal/platform/llm"
)

type CriterionScore struct {
	Score    float64
	Feedback string
}

type GradingResult struct {
	OverallBand float64
	Criteria    map[string]CriterionScore
	ModelAnswer string
	Corrections []Correction
}

type Correction struct {
	Span       string
	Issue      string
	Suggestion string
}

type Grader interface {
	Grade(ctx context.Context, skill, taskType, prompt, answer string) (*GradingResult, error)
}

type openAIGrader struct {
	client *llm.Client
}

func NewOpenAIGrader(client *llm.Client) Grader {
	return &openAIGrader{client: client}
}

func (g *openAIGrader) Grade(ctx context.Context, skill, taskType, prompt, answer string) (*GradingResult, error) {
	return nil, nil
	// TODO: build systemPrompt (yêu cầu LLM trả JSON đúng schema GradingResult)
	// TODO: text, err := g.client.Complete(ctx, systemPrompt, userPrompt)
	// TODO: json.Unmarshal([]byte(text), &result)
	// return &result, nil
}

var _ Grader = (*openAIGrader)(nil)
