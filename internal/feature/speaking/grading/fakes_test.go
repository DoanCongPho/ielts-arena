package grading

import (
	"context"
	"errors"
	"fmt"
	"github/DoanCongPho/game-arena/internal/platform/llm"
	"github/DoanCongPho/game-arena/internal/platform/pronunciation"
	"strings"
	"sync"
)

type fakeAudio map[string][]byte

func (f fakeAudio) Get(_ context.Context, key string) ([]byte, error) {
	b, ok := f[key]
	if !ok {
		return nil, errors.New("no such recording")
	}
	return b, nil
}

// fakeASR "transcribes" a recording by reading its bytes as the text.
type fakeASR struct{}

func (fakeASR) Transcribe(_ context.Context, _ string, audio []byte, _, _ string) (*llm.Transcript, error) {
	w := words(string(audio), nil)
	return &llm.Transcript{Text: string(audio), Duration: w[len(w)-1].End + 0.5, Words: w,
		Segments: []llm.TranscriptSegment{{Start: 0, End: w[len(w)-1].End, AvgLogprob: -0.2}}}, nil
}

type fakePron struct {
	err   error
	calls int
}

func (f *fakePron) Assess(context.Context, []byte, string, string, any) (*pronunciation.Assessment, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &pronunciation.Assessment{Intelligibility: 0.95, GOPMean: 85, Prosody: pronunciation.Prosody{PitchStdST: 3}}, nil
}

// fakeJudge gives every criterion the same band and records prompts. The
// criteria and the Part 2 rewrite run concurrently, hence the lock.
type fakeJudge struct {
	band    int
	mu      sync.Mutex
	prompts []string
	// rewrites are the user prompts of part review calls, kept apart
	// from the criterion judges.
	rewrites []string
}

func (f *fakeJudge) Complete(_ context.Context, system, user, _ string, _ llm.CompletionParams) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if isRewrite(system) {
		f.rewrites = append(f.rewrites, user)
		return `{"markdown":"### Mục 5 — Bài sửa theo 4 tầng\n**TIME** ..."}`, nil
	}
	f.prompts = append(f.prompts, system+"\n"+user)
	return fmt.Sprintf(`{"checks":[{"band":%d,"feature":"x","verdict":"met","evidence":"y"}],"band":%d,"feedback":"ok","improvements":["z"]}`, f.band, f.band), nil
}

// isRewrite tells a teacher's part review call from a criterion judge.
func isRewrite(system string) bool { return strings.HasSuffix(system, rewriteOutputFormat) }
