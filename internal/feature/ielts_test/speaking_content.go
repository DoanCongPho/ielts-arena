package ielts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SpeakingContent is a speaking test's content_data. A full test has all
// three parts, as in the real exam; a test with only one part is part
// practice, which is graded the same way but reported as indicative (IELTS
// never scores one part on its own).
type SpeakingContent struct {
	// Title names a custom test in the user's list; official tests are
	// named by their series instead.
	Title string         `json:"title,omitempty"`
	Part1 *SpeakingPart1 `json:"part1,omitempty"`
	Part2 *SpeakingPart2 `json:"part2,omitempty"`
	Part3 *SpeakingPart3 `json:"part3,omitempty"`
}

// SpeakingPart1 is the interview: familiar topics, a few questions each.
type SpeakingPart1 struct {
	Topics []SpeakingTopic `json:"topics"`
}

type SpeakingTopic struct {
	Topic     string             `json:"topic"`
	Questions []SpeakingQuestion `json:"questions"`
}

// SpeakingQuestion's ID is assigned by normalizeSpeakingContent from its
// position ("p1.t2.q3"), so authors never write one and answers can be
// matched back to questions after the content is stored.
type SpeakingQuestion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// SpeakingPart2 is the long turn: a cue card, one minute to prepare, up to
// two minutes to talk, then one or two rounding-off questions.
type SpeakingPart2 struct {
	ID      string   `json:"id"`
	Topic   string   `json:"topic"`   // "Describe a book you enjoyed reading."
	Bullets []string `json:"bullets"` // "You should say: …" points
	Explain string   `json:"explain"` // "and explain why you enjoyed it."
	// FollowUps are the rounding-off questions after the long turn.
	FollowUps []SpeakingQuestion `json:"follow_ups,omitempty"`
}

// SpeakingPart3 is the discussion: abstract questions linked to Part 2's
// topic. Theme names that topic when there is no Part 2 to borrow from.
type SpeakingPart3 struct {
	Theme     string             `json:"theme,omitempty"`
	Questions []SpeakingQuestion `json:"questions"`
}

// Speaking modes, derived from which parts a test has.
const (
	SpeakingModeFull  = "full"
	SpeakingModePart1 = "part1"
	SpeakingModePart2 = "part2"
	SpeakingModePart3 = "part3"
)

// Authoring limits. They follow the real test's shape with some slack, and
// keep a custom card from turning into an essay.
const (
	maxSpeakingTextLen = 300
	minPart1Questions  = 3
	maxPart1Questions  = 15
	minPart2Bullets    = 3
	maxPart2Bullets    = 4
	maxPart2FollowUps  = 2
	minPart3Questions  = 3
	maxPart3Questions  = 8
)

// Mode reports which mode the content supports: full when it has all three
// parts, a single part otherwise. Content with two parts is not a mode the
// real test has, and is rejected by validation.
func (c SpeakingContent) Mode() string {
	switch {
	case c.Part1 != nil && c.Part2 != nil && c.Part3 != nil:
		return SpeakingModeFull
	case c.Part1 != nil && c.Part2 == nil && c.Part3 == nil:
		return SpeakingModePart1
	case c.Part2 != nil && c.Part1 == nil && c.Part3 == nil:
		return SpeakingModePart2
	case c.Part3 != nil && c.Part1 == nil && c.Part2 == nil:
		return SpeakingModePart3
	}
	return ""
}

// NormalizeSpeakingContent trims every text and (re)assigns question IDs
// from their positions. It runs before validation and before storing, so
// IDs are always consistent with the stored order.
func NormalizeSpeakingContent(c *SpeakingContent) {
	c.Title = strings.TrimSpace(c.Title)
	if p := c.Part1; p != nil {
		for ti := range p.Topics {
			t := &p.Topics[ti]
			t.Topic = strings.TrimSpace(t.Topic)
			for qi := range t.Questions {
				t.Questions[qi].ID = fmt.Sprintf("p1.t%d.q%d", ti+1, qi+1)
				t.Questions[qi].Text = strings.TrimSpace(t.Questions[qi].Text)
			}
		}
	}
	if p := c.Part2; p != nil {
		p.ID = "p2"
		p.Topic = strings.TrimSpace(p.Topic)
		p.Explain = strings.TrimSpace(p.Explain)
		for i := range p.Bullets {
			p.Bullets[i] = strings.TrimSpace(p.Bullets[i])
		}
		for i := range p.FollowUps {
			p.FollowUps[i].ID = fmt.Sprintf("p2.f%d", i+1)
			p.FollowUps[i].Text = strings.TrimSpace(p.FollowUps[i].Text)
		}
	}
	if p := c.Part3; p != nil {
		p.Theme = strings.TrimSpace(p.Theme)
		for i := range p.Questions {
			p.Questions[i].ID = fmt.Sprintf("p3.q%d", i+1)
			p.Questions[i].Text = strings.TrimSpace(p.Questions[i].Text)
		}
	}
}

// canonicalContentData returns content_data as it should be stored: for
// speaking, trimmed and with question IDs assigned; other skills unchanged.
func canonicalContentData(skill string, raw []byte) ([]byte, error) {
	if skill != "speaking" {
		return raw, nil
	}
	var c SpeakingContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("invalid content_data for speaking: %w", err)
	}
	NormalizeSpeakingContent(&c)
	return json.Marshal(c)
}

// speakingModeOf reads the mode of already-validated content.
func speakingModeOf(raw []byte) string {
	var c SpeakingContent
	_ = json.Unmarshal(raw, &c)
	return c.Mode()
}

// ValidateSpeakingContent checks normalized content against the test's
// shape and the authoring limits.
func ValidateSpeakingContent(c SpeakingContent) error {
	if c.Mode() == "" {
		return errors.New("a speaking test has either all three parts or exactly one")
	}
	if len([]rune(c.Title)) > 100 {
		return errors.New("title is longer than 100 characters")
	}
	if p := c.Part1; p != nil {
		if len(p.Topics) == 0 || len(p.Topics) > 3 {
			return errors.New("part1 needs 1-3 topics")
		}
		n := 0
		for i, t := range p.Topics {
			if err := checkSpeakingText(fmt.Sprintf("part1 topic %d", i+1), t.Topic); err != nil {
				return err
			}
			if len(t.Questions) == 0 {
				return fmt.Errorf("part1 topic %d has no questions", i+1)
			}
			for _, q := range t.Questions {
				if err := checkSpeakingText("question "+q.ID, q.Text); err != nil {
					return err
				}
			}
			n += len(t.Questions)
		}
		if n < minPart1Questions || n > maxPart1Questions {
			return fmt.Errorf("part1 needs %d-%d questions in total, got %d", minPart1Questions, maxPart1Questions, n)
		}
	}
	if p := c.Part2; p != nil {
		if err := checkSpeakingText("part2 topic", p.Topic); err != nil {
			return err
		}
		if len(p.Bullets) < minPart2Bullets || len(p.Bullets) > maxPart2Bullets {
			return fmt.Errorf("part2 needs %d-%d bullet points", minPart2Bullets, maxPart2Bullets)
		}
		for i, b := range p.Bullets {
			if err := checkSpeakingText(fmt.Sprintf("part2 bullet %d", i+1), b); err != nil {
				return err
			}
		}
		if err := checkSpeakingText("part2 explain line", p.Explain); err != nil {
			return err
		}
		if len(p.FollowUps) > maxPart2FollowUps {
			return fmt.Errorf("part2 has at most %d rounding-off questions", maxPart2FollowUps)
		}
		for _, q := range p.FollowUps {
			if err := checkSpeakingText("question "+q.ID, q.Text); err != nil {
				return err
			}
		}
	}
	if p := c.Part3; p != nil {
		if len(p.Questions) < minPart3Questions || len(p.Questions) > maxPart3Questions {
			return fmt.Errorf("part3 needs %d-%d questions", minPart3Questions, maxPart3Questions)
		}
		if c.Part2 == nil {
			if err := checkSpeakingText("part3 theme", p.Theme); err != nil {
				return err
			}
		}
		for _, q := range p.Questions {
			if err := checkSpeakingText("question "+q.ID, q.Text); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkSpeakingText(field, s string) error {
	if s == "" {
		return fmt.Errorf("%s is empty", field)
	}
	if len([]rune(s)) > maxSpeakingTextLen {
		return fmt.Errorf("%s is longer than %d characters", field, maxSpeakingTextLen)
	}
	return nil
}

// SpeakingQuestionRef is one prompt the candidate answers, flattened in exam
// order with the part it belongs to.
type SpeakingQuestionRef struct {
	ID   string
	Part int
	Text string
}

// Questions lists every answerable prompt in exam order. The Part 2 long
// turn is one question whose text is the whole cue card.
func (c SpeakingContent) Questions() []SpeakingQuestionRef {
	var qs []SpeakingQuestionRef
	if p := c.Part1; p != nil {
		for _, t := range p.Topics {
			for _, q := range t.Questions {
				qs = append(qs, SpeakingQuestionRef{ID: q.ID, Part: 1, Text: q.Text})
			}
		}
	}
	if p := c.Part2; p != nil {
		qs = append(qs, SpeakingQuestionRef{ID: p.ID, Part: 2, Text: p.CueCardText()})
		for _, q := range p.FollowUps {
			qs = append(qs, SpeakingQuestionRef{ID: q.ID, Part: 2, Text: q.Text})
		}
	}
	if p := c.Part3; p != nil {
		for _, q := range p.Questions {
			qs = append(qs, SpeakingQuestionRef{ID: q.ID, Part: 3, Text: q.Text})
		}
	}
	return qs
}

// CueCardText is the card as a candidate reads it.
func (p SpeakingPart2) CueCardText() string {
	var b strings.Builder
	b.WriteString(p.Topic)
	b.WriteString("\nYou should say:")
	for _, bullet := range p.Bullets {
		b.WriteString("\n- ")
		b.WriteString(bullet)
	}
	b.WriteString("\n")
	b.WriteString(p.Explain)
	return b.String()
}

// AllTexts is every author-written string, for moderation.
func (c SpeakingContent) AllTexts() []string {
	var out []string
	if c.Title != "" {
		out = append(out, c.Title)
	}
	if p := c.Part1; p != nil {
		for _, t := range p.Topics {
			out = append(out, t.Topic)
		}
	}
	if p := c.Part2; p != nil {
		out = append(out, p.Topic, p.Explain)
		out = append(out, p.Bullets...)
	}
	if p := c.Part3; p != nil && p.Theme != "" {
		out = append(out, p.Theme)
	}
	for _, q := range c.Questions() {
		if q.ID != "p2" {
			out = append(out, q.Text)
		}
	}
	return out
}
