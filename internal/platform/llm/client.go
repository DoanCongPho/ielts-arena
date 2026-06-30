package llm

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type Client struct {
	sdk   *openai.Client
	model string
}

func NewClient(apiKey, model string) *Client {
	return &Client{sdk: openai.NewClient(apiKey), model: model}
}

// Complete gọi chat completion, trả về text thô (Content của message đầu tiên).
// Không biết gì về GradingResult, Criteria, hay bất kỳ domain type nào.
func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "a", nil
	// TODO: gọi c.sdk.CreateChatCompletion(...)
	// TODO: return resp.Choices[0].Message.Content
}
