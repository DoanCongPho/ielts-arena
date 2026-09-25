package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go/v3"
)

// Transcript is a verbatim transcription with per-word timing — what
// fluency measures (pauses, speech rate) are computed from.
type Transcript struct {
	Text     string
	Duration float64
	Words    []TranscriptWord
	Segments []TranscriptSegment
}

type TranscriptWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// TranscriptSegment carries Whisper's confidence for a stretch of speech:
// a low AvgLogprob means the recogniser struggled to make out the words.
type TranscriptSegment struct {
	Text       string  `json:"text"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	AvgLogprob float64 `json:"avg_logprob"`
}

// Transcribe runs speech-to-text with word and segment timestamps. The
// model must support verbose_json (whisper-1 does; the gpt-4o transcribe
// models return no timestamps). prompt biases the style of the output —
// a prompt written with fillers keeps the speaker's "um"s and restarts.
func (c *Client) Transcribe(ctx context.Context, model string, audio []byte, filename, prompt string) (*Transcript, error) {
	params := openai.AudioTranscriptionNewParams{
		File:                   openai.File(bytes.NewReader(audio), filename, ""),
		Model:                  model,
		Language:               openai.String("en"),
		Temperature:            openai.Float(0),
		ResponseFormat:         openai.AudioResponseFormatVerboseJSON,
		TimestampGranularities: []string{"word", "segment"},
	}
	if prompt != "" {
		params.Prompt = openai.String(prompt)
	}
	resp, err := c.ai.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	t := &Transcript{Text: resp.Text, Duration: resp.Duration}
	for _, w := range resp.Words {
		t.Words = append(t.Words, TranscriptWord{Word: w.Word, Start: w.Start, End: w.End})
	}
	for _, s := range resp.Segments {
		t.Segments = append(t.Segments, TranscriptSegment{Text: s.Text, Start: s.Start, End: s.End, AvgLogprob: s.AvgLogprob})
	}
	return t, nil
}

// Speak renders text to MP3 with a text-to-speech model. instructions
// steer delivery (accent, pace, tone) on models that support them.
func (c *Client) Speak(ctx context.Context, model, voice, instructions, text string) ([]byte, error) {
	params := openai.AudioSpeechNewParams{
		Input:          text,
		Model:          model,
		Voice:          openai.AudioSpeechNewParamsVoiceUnion{OfString: openai.String(voice)},
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	}
	if instructions != "" {
		params.Instructions = openai.String(instructions)
	}
	resp, err := c.ai.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read speech: %w", err)
	}
	return b, nil
}

// Flagged reports whether OpenAI's moderation model flags any of texts.
// The moderation endpoint is free to call.
func (c *Client) Flagged(ctx context.Context, texts []string) (bool, error) {
	if len(texts) == 0 {
		return false, nil
	}
	resp, err := c.ai.Moderations.New(ctx, openai.ModerationNewParams{
		Input: openai.ModerationNewParamsInputUnion{OfStringArray: texts},
		Model: openai.ModerationModelOmniModerationLatest,
	})
	if err != nil {
		return false, err
	}
	for _, r := range resp.Results {
		if r.Flagged {
			return true, nil
		}
	}
	return false, nil
}
