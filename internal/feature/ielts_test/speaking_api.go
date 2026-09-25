package ielts_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github/DoanCongPho/game-arena/internal/platform/httpx"
	"github/DoanCongPho/game-arena/internal/platform/llm"
)

// Errors of the speaking endpoints, mapped to HTTP statuses by the handler.
var (
	ErrInvalidSpeakingContent = errors.New("invalid speaking content")
	ErrContentFlagged         = errors.New("this content can't be used: it was flagged by moderation")
	ErrTestHasAttempts        = errors.New("this test has attempts and can't be changed or deleted")
	ErrTooManyCustomTests     = errors.New("you have reached the limit of custom speaking tests")
)

const (
	// maxCustomTests bounds how many custom speaking tests one user keeps.
	maxCustomTests = 100
	// maxUploadSlots is how many upload links one request may ask for — a
	// full test has at most 15 + 3 + 8 answers.
	maxUploadSlots = 30
	uploadLinkTTL  = 30 * time.Minute
)

// Recording formats browsers produce with MediaRecorder: Chrome and Firefox
// record WebM/Ogg Opus, Safari MP4/AAC. Whisper accepts all of them.
var recordingExts = map[string]bool{"webm": true, "ogg": true, "mp4": true, "m4a": true}

type (
	uploadStore interface {
		UploadURL(key string, expires time.Duration) (string, error)
		DownloadURL(key string, expires time.Duration) (string, error)
	}
	moderator interface {
		Flagged(ctx context.Context, texts []string) (bool, error)
	}
)

// SpeakingService serves everything speaking needs besides submitting and
// grading, which go through Service like every other skill.
type SpeakingService struct {
	repo      Repository
	store     uploadStore
	examiner  *ExaminerAudio
	author    completer // nil: no content generation or style check
	moderator moderator // nil: no moderation
}

func NewSpeakingService(repo Repository, store uploadStore, examiner *ExaminerAudio, author completer, mod moderator) *SpeakingService {
	return &SpeakingService{repo: repo, store: store, examiner: examiner, author: author, moderator: mod}
}

// UploadSlot is where the browser PUTs one recording, and the key the
// submission then names it by.
type UploadSlot struct {
	Key       string `json:"key"`
	UploadURL string `json:"upload_url"`
}

func (s *SpeakingService) UploadSlots(userID uint64, count int, ext string) ([]UploadSlot, error) {
	if count < 1 || count > maxUploadSlots {
		return nil, fmt.Errorf("%w: count must be 1-%d", ErrInvalidSpeakingContent, maxUploadSlots)
	}
	if !recordingExts[ext] {
		return nil, fmt.Errorf("%w: unsupported recording format %q", ErrInvalidSpeakingContent, ext)
	}
	slots := make([]UploadSlot, count)
	for i := range slots {
		var id [12]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, err
		}
		key := SpeakingAudioPrefix(userID) + hex.EncodeToString(id[:]) + "." + ext
		link, err := s.store.UploadURL(key, uploadLinkTTL)
		if err != nil {
			return nil, err
		}
		slots[i] = UploadSlot{Key: key, UploadURL: link}
	}
	return slots, nil
}

// SpeakingScript is what the exam runner plays through.
type SpeakingScript struct {
	TestID uint64       `json:"test_id"`
	Mode   string       `json:"mode"`
	Lines  []ScriptLine `json:"lines"`
}

func (s *SpeakingService) Script(ctx context.Context, userID, testID uint64) (*SpeakingScript, error) {
	t, content, err := s.speakingTest(ctx, userID, testID)
	if err != nil {
		return nil, err
	}
	lines := buildSpeakingScript(*content, s.examiner.Voice())
	lines = s.examiner.Resolve(ctx, lines)
	return &SpeakingScript{TestID: t.ID, Mode: content.Mode(), Lines: lines}, nil
}

// speakingTest loads a speaking test the user may take.
func (s *SpeakingService) speakingTest(ctx context.Context, userID, testID uint64) (*Test, *SpeakingContent, error) {
	t, err := s.repo.GetTestByID(ctx, testID)
	if err != nil {
		return nil, nil, err
	}
	if t.Skill != "speaking" || (t.OwnerID != 0 && t.OwnerID != userID) {
		return nil, nil, ErrTestNotFound
	}
	var c SpeakingContent
	if err := json.Unmarshal(t.ContentData, &c); err != nil {
		return nil, nil, fmt.Errorf("read test content: %w", err)
	}
	return t, &c, nil
}

func (s *SpeakingService) ListCustom(ctx context.Context, userID uint64, page int) (*ListTestResponse, error) {
	req := ListTestRequest{Page: page}
	tests, total, err := s.repo.ListOwnedTests(ctx, userID, "speaking", req.Limit(), req.Offset())
	if err != nil {
		return nil, err
	}
	resp := &ListTestResponse{Data: []TestResponse{}, Pagination: httpx.NewPagination(total, req.Page, req.Limit())}
	for i := range tests {
		resp.Data = append(resp.Data, NewTestResponse(&tests[i], tests[i].ContentData))
	}
	return resp, nil
}

// ComposeSpeakingRequest builds a custom test part by part: each part
// present comes either from an official test (BankTestID) or is written by
// the user (Custom). The parts present decide the mode — all three for a
// full test, one for part practice.
type ComposeSpeakingRequest struct {
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
	Test     TestResponse `json:"test"`
	Warnings []string     `json:"warnings"`
}

func (s *SpeakingService) CreateCustom(ctx context.Context, userID uint64, req ComposeSpeakingRequest) (*ComposeResult, error) {
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
	t, err := s.repo.CreateTest(ctx, &Test{
		OwnerID: userID, Skill: "speaking", TaskType: content.Mode(),
		ContentData: raw, Source: "custom", CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, err
	}
	s.examiner.Prepare(buildSpeakingScript(*content, s.examiner.Voice()))
	return &ComposeResult{Test: NewTestResponse(t, t.ContentData), Warnings: s.styleWarnings(ctx, custom)}, nil
}

func (s *SpeakingService) UpdateCustom(ctx context.Context, userID, testID uint64, req ComposeSpeakingRequest) (*ComposeResult, error) {
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
	s.examiner.Prepare(buildSpeakingScript(*content, s.examiner.Voice()))
	return &ComposeResult{Test: NewTestResponse(t, t.ContentData), Warnings: s.styleWarnings(ctx, custom)}, nil
}

func (s *SpeakingService) DeleteCustom(ctx context.Context, userID, testID uint64) error {
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

func (s *SpeakingService) ownedTest(ctx context.Context, userID, testID uint64) (*Test, error) {
	t, err := s.repo.GetTestByID(ctx, testID)
	if err != nil {
		return nil, err
	}
	if t.OwnerID != userID || t.Skill != "speaking" {
		return nil, ErrTestNotFound
	}
	return t, nil
}

// compose assembles, validates and moderates a custom test. It also
// returns just the user-written parts, for the style check.
func (s *SpeakingService) compose(ctx context.Context, req ComposeSpeakingRequest) (*SpeakingContent, SpeakingContent, error) {
	c := &SpeakingContent{Title: req.Title}
	var custom SpeakingContent
	banks := map[uint64]*SpeakingContent{}
	bank := func(id uint64) (*SpeakingContent, error) {
		if b, ok := banks[id]; ok {
			return b, nil
		}
		t, err := s.repo.GetTestByID(ctx, id)
		if err != nil || t.OwnerID != 0 || t.Skill != "speaking" {
			return nil, fmt.Errorf("%w: official speaking test %d not found", ErrInvalidSpeakingContent, id)
		}
		var b SpeakingContent
		if err := json.Unmarshal(t.ContentData, &b); err != nil {
			return nil, err
		}
		banks[id] = &b
		return &b, nil
	}
	pick := func(name string, src *PartSource, fromBank func(*SpeakingContent) bool, fromCustom func(json.RawMessage) error) error {
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
				return fmt.Errorf("%w: test %d has no %s", ErrInvalidSpeakingContent, src.BankTestID, name)
			}
		case src.BankTestID == 0 && len(src.Custom) > 0:
			if err := fromCustom(src.Custom); err != nil {
				return fmt.Errorf("%w: %s: %v", ErrInvalidSpeakingContent, name, err)
			}
		default:
			return fmt.Errorf("%w: %s needs either bank_test_id or custom", ErrInvalidSpeakingContent, name)
		}
		return nil
	}

	err := errors.Join(
		pick("part1", req.Part1,
			func(b *SpeakingContent) bool { c.Part1 = b.Part1; return b.Part1 != nil },
			func(raw json.RawMessage) error {
				c.Part1 = new(SpeakingPart1)
				custom.Part1 = c.Part1
				return json.Unmarshal(raw, c.Part1)
			}),
		pick("part2", req.Part2,
			func(b *SpeakingContent) bool { c.Part2 = b.Part2; return b.Part2 != nil },
			func(raw json.RawMessage) error {
				c.Part2 = new(SpeakingPart2)
				custom.Part2 = c.Part2
				return json.Unmarshal(raw, c.Part2)
			}),
		pick("part3", req.Part3,
			func(b *SpeakingContent) bool {
				if b.Part3 == nil {
					return false
				}
				p := *b.Part3
				// A bank Part 3 follows on from its own Part 2; keep that
				// topic as the theme when the Part 2 is not coming along.
				if p.Theme == "" && b.Part2 != nil {
					p.Theme = part2Subject(b.Part2.Topic)
				}
				c.Part3 = &p
				return true
			},
			func(raw json.RawMessage) error {
				c.Part3 = new(SpeakingPart3)
				custom.Part3 = c.Part3
				return json.Unmarshal(raw, c.Part3)
			}),
	)
	if err != nil {
		return nil, custom, err
	}

	NormalizeSpeakingContent(c)
	if err := ValidateSpeakingContent(*c); err != nil {
		return nil, custom, fmt.Errorf("%w: %v", ErrInvalidSpeakingContent, err)
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

// styleWarnings asks a cheap model whether the user's own parts read like
// real IELTS prompts. It never blocks saving: a failure means no warnings.
func (s *SpeakingService) styleWarnings(ctx context.Context, custom SpeakingContent) []string {
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
	Part2 SpeakingPart2 `json:"part2"`
	Part1 bool          `json:"part1"`
	Part3 bool          `json:"part3"`
}

// GenerateParts drafts Part 1 and/or Part 3 for a cue card. The result is
// returned for the user to edit, not saved.
func (s *SpeakingService) GenerateParts(ctx context.Context, req GeneratePartsRequest) (*SpeakingContent, error) {
	if s.author == nil {
		return nil, errors.New("content generation is not configured")
	}
	card := SpeakingContent{Part2: &req.Part2}
	NormalizeSpeakingContent(&card)
	if err := ValidateSpeakingContent(card); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSpeakingContent, err)
	}
	if !req.Part1 && !req.Part3 {
		return nil, fmt.Errorf("%w: ask for part1, part3 or both", ErrInvalidSpeakingContent)
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
	var out SpeakingContent
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return nil, fmt.Errorf("generate: parse: %w", err)
	}
	out.Part2 = card.Part2
	NormalizeSpeakingContent(&out)
	return &out, nil
}

// Recordings returns playback links for a submission's answers, keyed by
// question id — only to the user who made them.
func (s *SpeakingService) Recordings(ctx context.Context, userID, submissionID uint64) (map[string]string, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub.UserID != userID {
		return nil, ErrSubmissionNotFound
	}
	var p SpeakingPayload
	if err := json.Unmarshal(sub.Payload, &p); err != nil {
		return nil, ErrSubmissionNotFound
	}
	out := map[string]string{}
	for _, a := range p.Answers {
		if !strings.HasPrefix(a.AudioKey, SpeakingAudioPrefix(userID)) {
			continue
		}
		if u, err := s.store.DownloadURL(a.AudioKey, time.Hour); err == nil {
			out[a.QuestionID] = u
		}
	}
	return out, nil
}
