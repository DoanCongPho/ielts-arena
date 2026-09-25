package ielts_test

import (
	"context"
	"log"
	"sync"
	"time"
)

type (
	speechSynth interface {
		Speak(ctx context.Context, model, voice, instructions, text string) ([]byte, error)
	}
	examinerStore interface {
		DownloadURL(key string, expires time.Duration) (string, error)
		Put(ctx context.Context, key, contentType string, body []byte) error
		Exists(ctx context.Context, key string) (bool, error)
	}
)

// ExaminerAudio gives script lines a recorded examiner voice. Recordings
// are made on demand: the first time a line is served and has none, it is
// queued for text-to-speech and the runner falls back to browser speech
// until the recording exists. Every line's key is a hash of its text and
// voice, so each distinct line is synthesised once, ever, and shared by
// every test that uses it — the standard examiner wording costs nothing
// after its first use.
type ExaminerAudio struct {
	store examinerStore
	tts   speechSynth // nil: never synthesise; browser speech only
	voice ExaminerVoice

	known   sync.Map // audio key -> struct{}: recording exists
	pending sync.Map // audio key -> struct{}: queued or being made
	queue   chan ScriptLine
}

// examinerQueueSize bounds lines waiting for synthesis; beyond it new
// lines are dropped and retried the next time they're served.
const examinerQueueSize = 256

func NewExaminerAudio(store examinerStore, tts speechSynth, voice ExaminerVoice) *ExaminerAudio {
	return &ExaminerAudio{store: store, tts: tts, voice: voice, queue: make(chan ScriptLine, examinerQueueSize)}
}

func (e *ExaminerAudio) Voice() ExaminerVoice { return e.voice }

// Start runs the synthesis worker until ctx is cancelled. One worker is
// enough: lines are short and only ever made once.
func (e *ExaminerAudio) Start(ctx context.Context) {
	if e.tts == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case line := <-e.queue:
				e.synthesise(ctx, line)
			}
		}
	}()
}

func (e *ExaminerAudio) synthesise(ctx context.Context, line ScriptLine) {
	defer e.pending.Delete(line.AudioKey)
	jobCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	// Another replica may have made it since this one queued it.
	if ok, err := e.store.Exists(jobCtx, line.AudioKey); err == nil && ok {
		e.known.Store(line.AudioKey, struct{}{})
		return
	}
	mp3, err := e.tts.Speak(jobCtx, e.voice.Model, e.voice.Voice, e.voice.Instructions, line.Text)
	if err != nil {
		log.Printf("examiner audio: synthesise %s: %v", line.AudioKey, err)
		return
	}
	if err := e.store.Put(jobCtx, line.AudioKey, "audio/mpeg", mp3); err != nil {
		log.Printf("examiner audio: store %s: %v", line.AudioKey, err)
		return
	}
	e.known.Store(line.AudioKey, struct{}{})
}

// Resolve fills in AudioURL for lines whose recording exists and queues
// the rest for synthesis.
func (e *ExaminerAudio) Resolve(ctx context.Context, lines []ScriptLine) []ScriptLine {
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := range lines {
		key := lines[i].AudioKey
		if _, ok := e.known.Load(key); ok {
			continue
		}
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if ok, err := e.store.Exists(checkCtx, key); err == nil && ok {
				e.known.Store(key, struct{}{})
			}
		}(key)
	}
	wg.Wait()

	for i := range lines {
		if _, ok := e.known.Load(lines[i].AudioKey); ok {
			if u, err := e.store.DownloadURL(lines[i].AudioKey, 2*time.Hour); err == nil {
				lines[i].AudioURL = u
			}
			continue
		}
		e.enqueue(lines[i])
	}
	return lines
}

// Prepare queues synthesis for every line not yet recorded, without
// waiting — called when a test is created so its audio is usually ready
// before anyone takes it.
func (e *ExaminerAudio) Prepare(lines []ScriptLine) {
	go e.Resolve(context.Background(), lines)
}

func (e *ExaminerAudio) enqueue(line ScriptLine) {
	if e.tts == nil {
		return
	}
	if _, dup := e.pending.LoadOrStore(line.AudioKey, struct{}{}); dup {
		return
	}
	select {
	case e.queue <- line:
	default:
		e.pending.Delete(line.AudioKey)
	}
}
