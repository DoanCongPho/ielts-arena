// Package pronunciation calls the self-hosted pronunciation service
// (services/pronunciation, a wav2vec2 phoneme model on a free CPU VM). It
// knows the wire format only — what the numbers mean for an
// IELTS band is decided by the caller.
package pronunciation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Assessment is the service's verdict on one recording.
type Assessment struct {
	// Intelligibility is the share of words (0-1) whose phonemes the model
	// heard as the expected ones — words a listener gets without effort.
	Intelligibility float64 `json:"intelligibility"`
	// GOPMean is the mean goodness-of-pronunciation over all phonemes,
	// scaled 0-100.
	GOPMean float64 `json:"gop_mean"`
	// PER is the phoneme error rate of free decoding against the expected
	// phonemes (0 = every phoneme as expected).
	PER float64 `json:"per"`
	// AccentVariant is the reference ("en-us" or "en-gb") the speech
	// matched better. Neither is preferred: it is chosen per word.
	AccentVariant string  `json:"accent_variant"`
	Words         []Word  `json:"words"`
	Prosody       Prosody `json:"prosody"`
}

type Word struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Score float64 `json:"score"` // 0-100
	// Heard is what the model heard in the word's span, as space-separated
	// IPA phonemes, unconstrained by what was expected.
	Heard    string    `json:"heard,omitempty"`
	Phonemes []Phoneme `json:"phonemes,omitempty"`
}

type Phoneme struct {
	Expected string  `json:"expected"`
	Heard    string  `json:"heard,omitempty"`
	Score    float64 `json:"score"` // 0-100
}

// Prosody holds the suprasegmental measures the upper Pronunciation band
// descriptors talk about (intonation, stress, rhythm).
type Prosody struct {
	// PitchRangeST is the 10th-90th percentile pitch span in semitones;
	// PitchStdST its spread. Flat, monotone speech scores low on both.
	PitchRangeST float64 `json:"pitch_range_st"`
	PitchStdST   float64 `json:"pitch_std_st"`
	// StressMatchRate is the share of polysyllabic words whose most
	// prominent syllable is the dictionary's primary stress.
	StressMatchRate float64 `json:"stress_match_rate"`
	// NPVIVowel is the normalised pairwise variability of vowel durations:
	// higher for stress-timed English rhythm, lower for syllable-timed.
	NPVIVowel float64 `json:"npvi_vowel"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient returns nil when baseURL is empty, so callers can treat "no
// service configured" the same as "service down" and fall back.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	if baseURL == "" {
		return nil
	}
	return &Client{baseURL: baseURL, token: token, http: &http.Client{Timeout: timeout}}
}

// Assess sends one recording with its transcript. words are the
// recogniser's per-word timings, which the service uses to cut the audio
// per word instead of aligning the whole clip at once.
func (c *Client) Assess(ctx context.Context, audio []byte, filename, transcript string, words any) (*Assessment, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("audio", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(audio); err != nil {
		return nil, err
	}
	_ = mw.WriteField("transcript", transcript)
	if words != nil {
		wj, err := json.Marshal(words)
		if err != nil {
			return nil, err
		}
		_ = mw.WriteField("words", string(wj))
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/assess", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pronunciation service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("pronunciation service: status %d: %s", resp.StatusCode, msg)
	}
	var a Assessment
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, fmt.Errorf("pronunciation service: decode: %w", err)
	}
	return &a, nil
}
