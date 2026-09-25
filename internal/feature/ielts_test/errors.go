package ielts_test

import "errors"

var (
	ErrTestNotFound       = errors.New("test not found")
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrScoreNotFound      = errors.New("score not found")
	// ErrAnswerKeyLocked: the caller hasn't finished (had graded) an attempt
	// at this test, so its answers and explanations stay hidden.
	ErrAnswerKeyLocked = errors.New("the answer key is available once your attempt is graded")
	// ErrNoAnswerKey: the test's skill is graded by the LLM (writing /
	// speaking) and has no answer key.
	ErrNoAnswerKey = errors.New("this test has no answer key")
	// ErrInvalidSubmission: the payload doesn't fit the test (wrong
	// questions, someone else's recordings, …). Wrapped with the reason.
	ErrInvalidSubmission = errors.New("invalid submission")
)
