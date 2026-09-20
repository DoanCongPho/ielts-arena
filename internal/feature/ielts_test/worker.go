package ielts_test

import (
	"context"
	"log"
	"sync"
	"time"
)

// WorkerConfig tunes the grading worker pool.
type WorkerConfig struct {
	// Count is how many submissions may be graded concurrently. This is
	// also the cost/concurrency ceiling on the OpenAI API: with N workers
	// there are never more than N grading calls in flight, no matter how
	// many submissions arrive at once.
	Count int
	// PollInterval is how long a worker waits before re-checking an empty
	// queue. After doing work it re-polls immediately, so a busy queue
	// drains at full speed and an idle one stays cheap.
	PollInterval time.Duration
	// JobTimeout bounds a single grade, so one hung LLM call can't occupy
	// a worker forever. On timeout the submission is rescheduled and
	// picked up again later.
	JobTimeout time.Duration
}

func (c WorkerConfig) withDefaults() WorkerConfig {
	if c.Count <= 0 {
		c.Count = 2
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 3 * time.Second
	}
	if c.JobTimeout <= 0 {
		c.JobTimeout = 90 * time.Second
	}
	return c
}

// StartGradingWorkers launches the pool and returns a WaitGroup that
// completes once every worker has stopped. Workers stop when ctx is
// cancelled; a grade already in flight is cancelled with it and the
// submission returns to the queue, so shutdown never loses work — at
// worst a submission is graded twice as far as the LLM is concerned, and
// the score row is written once.
func StartGradingWorkers(ctx context.Context, svc Service, cfg WorkerConfig) *sync.WaitGroup {
	cfg = cfg.withDefaults()

	var wg sync.WaitGroup
	for i := 0; i < cfg.Count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runGradingWorker(ctx, svc, cfg, id)
		}(i + 1)
	}
	log.Printf("ielts_test: %d grading worker(s) started (poll=%s, job timeout=%s)",
		cfg.Count, cfg.PollInterval, cfg.JobTimeout)
	return &wg
}

func runGradingWorker(ctx context.Context, svc Service, cfg WorkerConfig, id int) {
	for {
		if ctx.Err() != nil {
			log.Printf("ielts_test: grading worker %d stopped", id)
			return
		}

		worked, err := gradeOne(ctx, svc, cfg.JobTimeout)
		if err != nil {
			// Already recorded on the submission (rescheduled or failed);
			// logged here so the failure is visible in the process output.
			log.Printf("ielts_test: grading worker %d: %v", id, err)
		}
		if worked {
			continue // queue has items — keep draining without waiting
		}

		select {
		case <-ctx.Done():
			log.Printf("ielts_test: grading worker %d stopped", id)
			return
		case <-time.After(cfg.PollInterval):
		}
	}
}

// gradeOne wraps a single unit of work in its own timeout, in a function
// so the cancel is released as soon as the job ends rather than on the
// next loop iteration.
func gradeOne(ctx context.Context, svc Service, timeout time.Duration) (bool, error) {
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return svc.GradeNextPending(jobCtx)
}
