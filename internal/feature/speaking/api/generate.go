package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"log"
	"strings"
)

// styleWarnings asks a cheap model whether the user's own parts read like
// real IELTS prompts. It never blocks saving: a failure means no warnings.
func (s *Service) styleWarnings(ctx context.Context, custom ielts_test.SpeakingContent) []string {
	warnings := []string{}
	if s.author == nil || (custom.Part1 == nil && custom.Part2 == nil && custom.Part3 == nil) {
		return warnings
	}
	raw, _ := json.Marshal(custom)
	resp, err := s.author.Complete(ctx, styleCheckPrompt, string(raw), "", llm.CompletionParams{Temperature: 0.2, MaxTokens: 400})
	if err != nil {
		return warnings
	}
	var out struct {
		Warnings []string `json:"warnings"`
	}
	if json.Unmarshal([]byte(resp), &out) == nil && out.Warnings != nil {
		warnings = out.Warnings
	}
	return warnings
}

const styleCheckPrompt = `You review practice material for the IELTS Speaking test. The user wrote the parts below.
Point out anything that doesn't match the real test's style, briefly and only if it matters, for example:
- Part 1 questions that aren't about familiar, everyday topics, or that are yes/no only.
- A Part 2 card whose topic isn't a "Describe…" prompt someone could talk about for two minutes, or whose bullet points don't guide the talk.
- Part 3 questions that are personal rather than abstract and discursive.
If everything is fine, return an empty list.
Respond with ONLY valid JSON: {"warnings": ["<one short sentence each>"]}`

// GeneratePartsRequest asks for the parts of a full test to be written
// around the user's own Part 2 cue card.
type GeneratePartsRequest struct {
	Part2 ielts_test.SpeakingPart2 `json:"part2"`
	Part1 bool                     `json:"part1"`
	Part3 bool                     `json:"part3"`
}

// GenerateParts drafts Part 1 and/or Part 3 for a cue card. The result is
// returned for the user to edit, not saved.
func (s *Service) GenerateParts(ctx context.Context, req GeneratePartsRequest) (*ielts_test.SpeakingContent, error) {
	if s.author == nil {
		return nil, errors.New("content generation is not configured")
	}
	card := ielts_test.SpeakingContent{Part2: &req.Part2}
	ielts_test.NormalizeSpeakingContent(&card)
	if err := ielts_test.ValidateSpeakingContent(card); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidContent, err)
	}
	if !req.Part1 && !req.Part3 {
		return nil, fmt.Errorf("%w: ask for part1, part3 or both", ErrInvalidContent)
	}
	if s.moderator != nil {
		if flagged, err := s.moderator.Flagged(ctx, card.AllTexts()); err != nil {
			log.Printf("speaking: moderation unavailable, generating unchecked: %v", err)
		} else if flagged {
			return nil, ErrContentFlagged
		}
	}

	var want []string
	if req.Part1 {
		want = append(want, `"part1": {"topics": [{"topic": "<familiar topic>", "questions": [{"text": "<question>"}]}]}  — 2 or 3 everyday topics (home, work or study, hobbies…), 3-4 short questions each, NOT about the cue card's topic, as in the real test`)
	}
	if req.Part3 {
		want = append(want, `"part3": {"questions": [{"text": "<question>"}]}  — 4 to 6 abstract, discursive questions that broaden the cue card's topic to society, comparisons, causes and the future`)
	}
	system := "You write IELTS Speaking test material in the exact style of official Cambridge IELTS books. Respond with ONLY valid JSON with these keys:\n" + strings.Join(want, "\n")
	cardJSON, _ := json.Marshal(card.Part2)
	resp, err := s.author.Complete(ctx, system, "Part 2 cue card:\n"+string(cardJSON), "", llm.CompletionParams{Temperature: 0.7, MaxTokens: 1200})
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	var out ielts_test.SpeakingContent
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return nil, fmt.Errorf("generate: parse: %w", err)
	}
	out.Part2 = card.Part2
	ielts_test.NormalizeSpeakingContent(&out)
	return &out, nil
}
