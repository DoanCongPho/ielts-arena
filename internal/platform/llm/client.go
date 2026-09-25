package llm

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// ErrNoCompletion means the API returned successfully but with no usable
// message — a content filter fired, or generation stopped before emitting
// anything. Callers treat it as a transient failure worth retrying.
var ErrNoCompletion = errors.New("llm returned no completion choices")

type Client struct {
	ai    openai.Client
	model string
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		ai:    openai.NewClient(option.WithAPIKey(apiKey)),
		model: model,
	}
}

// CompletionParams tunes sampling for a single Complete call. Each caller
// (feature package) decides its own values — e.g. grading wants low
// Temperature for consistent scores, while a creative feature might want
// higher. Zero-value fields are left unset and fall back to the API default.
type CompletionParams struct {
	// Model overrides the client's default model for this call.
	Model       string
	Temperature float64
	MaxTokens   int64
	TopP        float64
}

// Complete sends system + user messages and returns the raw text response.
// It knows nothing about domain types — callers handle prompt building and parsing.
func (c *Client) Complete(ctx context.Context, system, user, imageURL string, params CompletionParams) (string, error) {
	model := c.model
	if params.Model != "" {
		model = params.Model
	}
	req := openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			c.userMessage(user, imageURL),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: "json_object",
			},
		},
	}
	if params.Temperature != 0 {
		req.Temperature = openai.Float(params.Temperature)
	}
	if params.MaxTokens != 0 {
		req.MaxTokens = openai.Int(params.MaxTokens)
	}
	if params.TopP != 0 {
		req.TopP = openai.Float(params.TopP)
	}

	resp, err := c.ai.Chat.Completions.New(ctx, req)
	if err != nil {
		return "", err
	}
	// A successful HTTP response does not guarantee a choice: indexing
	// Choices[0] blindly panics whenever the model returns none.
	if len(resp.Choices) == 0 {
		return "", ErrNoCompletion
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *Client) userMessage(text, imageURL string) openai.ChatCompletionMessageParamUnion {
	if imageURL == "" {
		return openai.UserMessage(text)
	}
	return openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(text),
		openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: imageURL}),
	})
}
