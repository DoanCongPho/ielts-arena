package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github/DoanCongPho/game-arena/internal/feature/ielts_test"
	"github/DoanCongPho/game-arena/internal/feature/speaking/examiner"
	"github/DoanCongPho/game-arena/internal/platform/httpx"
	"log"
	"time"
)

func (s *Service) ListCustom(ctx context.Context, userID uint64, page int) (*ielts_test.ListTestResponse, error) {
	req := ielts_test.ListTestRequest{Page: page}
	tests, total, err := s.repo.ListOwnedTests(ctx, userID, "speaking", req.Limit(), req.Offset())
	if err != nil {
		return nil, err
	}
	resp := &ielts_test.ListTestResponse{Data: []ielts_test.TestResponse{}, Pagination: httpx.NewPagination(total, req.Page, req.Limit())}
	for i := range tests {
		resp.Data = append(resp.Data, ielts_test.NewTestResponse(&tests[i], tests[i].ContentData))
	}
	return resp, nil
}

// ComposeRequest builds a custom test part by part: each part
// present comes either from an official test (BankTestID) or is written by
// the user (Custom). The parts present decide the mode — all three for a
// full test, one for part practice.
type ComposeRequest struct {
	Title string      `json:"title"`
	Part1 *PartSource `json:"part1,omitempty"`
	Part2 *PartSource `json:"part2,omitempty"`
	Part3 *PartSource `json:"part3,omitempty"`
}

type PartSource struct {
	BankTestID uint64          `json:"bank_test_id,omitempty"`
	Custom     json.RawMessage `json:"custom,omitempty"`
}

// ComposeResult is the saved test plus non-blocking advice about the
// user's own wording.
type ComposeResult struct {
	Test     ielts_test.TestResponse `json:"test"`
	Warnings []string                `json:"warnings"`
}

func (s *Service) CreateCustom(ctx context.Context, userID uint64, req ComposeRequest) (*ComposeResult, error) {
	if _, total, err := s.repo.ListOwnedTests(ctx, userID, "speaking", 1, 0); err != nil {
		return nil, err
	} else if total >= maxCustomTests {
		return nil, ErrTooManyCustomTests
	}
	content, custom, err := s.compose(ctx, req)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	// Custom tests earn no XP: users could otherwise mint XP by writing
	// cards as fast as they can answer them.
	t, err := s.repo.CreateTest(ctx, &ielts_test.Test{
		OwnerID: userID, Skill: "speaking", TaskType: content.Mode(),
		ContentData: raw, Source: "custom", CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, err
	}
	s.examiner.Prepare(examiner.Build(*content, s.examiner.Voice()))
	return &ComposeResult{Test: ielts_test.NewTestResponse(t, t.ContentData), Warnings: s.styleWarnings(ctx, custom)}, nil
}

func (s *Service) UpdateCustom(ctx context.Context, userID, testID uint64, req ComposeRequest) (*ComposeResult, error) {
	t, err := s.ownedTest(ctx, userID, testID)
	if err != nil {
		return nil, err
	}
	// Past scores refer to the questions as they were; editing would
	// silently change what those answers were answers to.
	if has, err := s.repo.HasSubmissions(ctx, t.ID); err != nil {
		return nil, err
	} else if has {
		return nil, ErrTestHasAttempts
	}
	content, custom, err := s.compose(ctx, req)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	t.TaskType, t.ContentData = content.Mode(), raw
	if err := s.repo.ReplaceTest(ctx, t); err != nil {
		return nil, err
	}
	s.examiner.Prepare(examiner.Build(*content, s.examiner.Voice()))
	return &ComposeResult{Test: ielts_test.NewTestResponse(t, t.ContentData), Warnings: s.styleWarnings(ctx, custom)}, nil
}

func (s *Service) DeleteCustom(ctx context.Context, userID, testID uint64) error {
	t, err := s.ownedTest(ctx, userID, testID)
	if err != nil {
		return err
	}
	if has, err := s.repo.HasSubmissions(ctx, t.ID); err != nil {
		return err
	} else if has {
		return ErrTestHasAttempts
	}
	return s.repo.DeleteTest(ctx, t.ID)
}

// compose assembles, validates and moderates a custom test. It also
// returns just the user-written parts, for the style check.
func (s *Service) compose(ctx context.Context, req ComposeRequest) (*ielts_test.SpeakingContent, ielts_test.SpeakingContent, error) {
	c := &ielts_test.SpeakingContent{Title: req.Title}
	var custom ielts_test.SpeakingContent
	banks := map[uint64]*ielts_test.SpeakingContent{}
	bank := func(id uint64) (*ielts_test.SpeakingContent, error) {
		if b, ok := banks[id]; ok {
			return b, nil
		}
		t, err := s.repo.GetTestByID(ctx, id)
		if err != nil || t.OwnerID != 0 || t.Skill != "speaking" {
			return nil, fmt.Errorf("%w: official speaking test %d not found", ErrInvalidContent, id)
		}
		var b ielts_test.SpeakingContent
		if err := json.Unmarshal(t.ContentData, &b); err != nil {
			return nil, err
		}
		banks[id] = &b
		return &b, nil
	}
	pick := func(name string, src *PartSource, fromBank func(*ielts_test.SpeakingContent) bool, fromCustom func(json.RawMessage) error) error {
		if src == nil {
			return nil
		}
		switch {
		case src.BankTestID != 0 && len(src.Custom) == 0:
			b, err := bank(src.BankTestID)
			if err != nil {
				return err
			}
			if !fromBank(b) {
				return fmt.Errorf("%w: test %d has no %s", ErrInvalidContent, src.BankTestID, name)
			}
		case src.BankTestID == 0 && len(src.Custom) > 0:
			if err := fromCustom(src.Custom); err != nil {
				return fmt.Errorf("%w: %s: %v", ErrInvalidContent, name, err)
			}
		default:
			return fmt.Errorf("%w: %s needs either bank_test_id or custom", ErrInvalidContent, name)
		}
		return nil
	}

	err := errors.Join(
		pick("part1", req.Part1,
			func(b *ielts_test.SpeakingContent) bool { c.Part1 = b.Part1; return b.Part1 != nil },
			func(raw json.RawMessage) error {
				c.Part1 = new(ielts_test.SpeakingPart1)
				custom.Part1 = c.Part1
				return json.Unmarshal(raw, c.Part1)
			}),
		pick("part2", req.Part2,
			func(b *ielts_test.SpeakingContent) bool { c.Part2 = b.Part2; return b.Part2 != nil },
			func(raw json.RawMessage) error {
				c.Part2 = new(ielts_test.SpeakingPart2)
				custom.Part2 = c.Part2
				return json.Unmarshal(raw, c.Part2)
			}),
		pick("part3", req.Part3,
			func(b *ielts_test.SpeakingContent) bool {
				if b.Part3 == nil {
					return false
				}
				p := *b.Part3
				// A bank Part 3 follows on from its own Part 2; keep that
				// topic as the theme when the Part 2 is not coming along.
				if p.Theme == "" && b.Part2 != nil {
					p.Theme = examiner.Part2Subject(b.Part2.Topic)
				}
				c.Part3 = &p
				return true
			},
			func(raw json.RawMessage) error {
				c.Part3 = new(ielts_test.SpeakingPart3)
				custom.Part3 = c.Part3
				return json.Unmarshal(raw, c.Part3)
			}),
	)
	if err != nil {
		return nil, custom, err
	}

	ielts_test.NormalizeSpeakingContent(c)
	if err := ielts_test.ValidateSpeakingContent(*c); err != nil {
		return nil, custom, fmt.Errorf("%w: %v", ErrInvalidContent, err)
	}
	if s.moderator != nil {
		texts := custom.AllTexts()
		if c.Title != "" {
			texts = append(texts, c.Title)
		}
		// A moderation outage lets content through (logged) rather than
		// locking users out: a custom test is only ever seen by its author.
		flagged, err := s.moderator.Flagged(ctx, texts)
		if err != nil {
			log.Printf("speaking: moderation unavailable, saving unchecked: %v", err)
		}
		if flagged {
			return nil, custom, ErrContentFlagged
		}
	}
	return c, custom, nil
}
