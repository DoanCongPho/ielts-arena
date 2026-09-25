package ielts_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Examiner script line kinds, which tell the runner what happens after the
// line is spoken.
const (
	LineSay      = "say"       // examiner speaks; nothing to answer
	LineAsk      = "ask"       // examiner asks; record the answer to QuestionID
	LineCueCard  = "cue_card"  // show the Part 2 card and run the preparation timer
	LineLongTurn = "long_turn" // record the Part 2 talk, stopped at the time limit
)

// Timings from the real test. Part 1 and 3 answer limits are soft: the
// runner shows them, and an examiner would move on around then.
const (
	part1AnswerSeconds   = 30
	part2PrepSeconds     = 60
	part2TalkSeconds     = 120
	part2FollowUpSeconds = 30
	part3AnswerSeconds   = 60
)

// ScriptLine is one step of the examiner's script, in order.
type ScriptLine struct {
	Kind       string `json:"kind"`
	Part       int    `json:"part"`
	Text       string `json:"text"`
	QuestionID string `json:"question_id,omitempty"`
	// Seconds is the answer limit (ask, long_turn) or preparation time
	// (cue_card).
	Seconds int `json:"seconds,omitempty"`
	// AudioKey is where the examiner's recording of Text is stored; see
	// examinerAudioKey. The runner falls back to browser speech when the
	// recording isn't there yet.
	AudioKey string `json:"-"`
	AudioURL string `json:"audio_url,omitempty"`
}

// ExaminerVoice fixes how every examiner line sounds. It is part of each
// line's audio key, so changing it regenerates everything rather than
// mixing two voices in one test.
type ExaminerVoice struct {
	Model        string
	Voice        string
	Instructions string
}

var defaultExaminerVoice = ExaminerVoice{
	Model: "gpt-4o-mini-tts",
	Voice: "sage",
	Instructions: "You are a neutral British IELTS speaking examiner. Speak with a clear southern British accent, " +
		"at a calm, measured pace, with clear articulation and no emotion or emphasis beyond natural questioning intonation.",
}

// NewExaminerVoice is the standard examiner delivery with the given model
// and voice; empty values keep the defaults.
func NewExaminerVoice(model, voice string) ExaminerVoice {
	v := defaultExaminerVoice
	if model != "" {
		v.Model = model
	}
	if voice != "" {
		v.Voice = voice
	}
	return v
}

// examinerAudioKey names a line's recording by a hash of everything that
// shapes the audio, so identical lines across tests share one file.
func examinerAudioKey(v ExaminerVoice, text string) string {
	sum := sha256.Sum256([]byte(v.Model + "\n" + v.Voice + "\n" + v.Instructions + "\n" + text))
	return "examiner/" + hex.EncodeToString(sum[:16]) + ".mp3"
}

// buildSpeakingScript turns content into the examiner's script, using the
// standard examiner wording for openings, hand-overs and the close.
func buildSpeakingScript(c SpeakingContent, v ExaminerVoice) []ScriptLine {
	var lines []ScriptLine
	say := func(part int, text string) {
		lines = append(lines, ScriptLine{Kind: LineSay, Part: part, Text: text})
	}
	ask := func(part int, q SpeakingQuestion, seconds int) {
		lines = append(lines, ScriptLine{Kind: LineAsk, Part: part, Text: q.Text, QuestionID: q.ID, Seconds: seconds})
	}

	full := c.Mode() == SpeakingModeFull
	say(firstPart(c), "Good afternoon. My name is Alex, and I'll be your examiner today.")

	if p := c.Part1; p != nil {
		say(1, "In this first part, I'd like to ask you some questions about yourself.")
		for _, t := range p.Topics {
			say(1, fmt.Sprintf("Let's talk about %s.", lowerFirst(strings.TrimSuffix(t.Topic, "."))))
			for _, q := range t.Questions {
				ask(1, q, part1AnswerSeconds)
			}
		}
	}

	if p := c.Part2; p != nil {
		say(2, "Now, I'm going to give you a topic, and I'd like you to talk about it for one to two minutes. "+
			"Before you talk, you'll have one minute to think about what you're going to say. "+
			"You can make some notes if you wish. Do you understand? Here is your topic.")
		lines = append(lines, ScriptLine{Kind: LineCueCard, Part: 2, Text: p.CueCardText(), Seconds: part2PrepSeconds})
		lines = append(lines, ScriptLine{
			Kind: LineLongTurn, Part: 2, QuestionID: p.ID, Seconds: part2TalkSeconds,
			Text: "All right? Remember, you have one to two minutes for this, so don't worry if I stop you. " +
				"I'll tell you when the time is up. Can you start speaking now, please?",
		})
		say(2, "Thank you.")
		for _, q := range p.FollowUps {
			ask(2, q, part2FollowUpSeconds)
		}
	}

	if p := c.Part3; p != nil {
		if c.Part2 != nil {
			say(3, fmt.Sprintf("We've been talking about %s, and I'd like to discuss with you one or two more general questions related to this.",
				part2Subject(c.Part2.Topic)))
		} else {
			say(3, fmt.Sprintf("In this part, I'd like to discuss with you some general questions about %s.",
				lowerFirst(strings.TrimSuffix(p.Theme, "."))))
		}
		for _, q := range p.Questions {
			ask(3, q, part3AnswerSeconds)
		}
	}

	if full {
		say(3, "Thank you. That is the end of the speaking test.")
	} else {
		say(lines[len(lines)-1].Part, "Thank you. That is the end of this part.")
	}

	for i := range lines {
		lines[i].AudioKey = examinerAudioKey(v, lines[i].Text)
	}
	return lines
}

func firstPart(c SpeakingContent) int {
	switch {
	case c.Part1 != nil:
		return 1
	case c.Part2 != nil:
		return 2
	}
	return 3
}

// part2Subject turns "Describe a book you enjoyed reading." into "a book
// you enjoyed reading" for the Part 3 lead-in.
func part2Subject(topic string) string {
	t := strings.TrimSuffix(strings.TrimSpace(topic), ".")
	for _, prefix := range []string{"Describe ", "Talk about ", "Tell me about "} {
		if rest, ok := strings.CutPrefix(t, prefix); ok {
			return rest
		}
	}
	return lowerFirst(t)
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	// Keep acronyms and names ("TV", "IELTS", "London") as written.
	if len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z' {
		return s
	}
	return strings.ToLower(string(r[0])) + string(r[1:])
}
