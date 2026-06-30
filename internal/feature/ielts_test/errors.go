package ielts_test

import "errors"

var (
	ErrTestNotFound       = errors.New("test not found")
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrScoreNotFound      = errors.New("score not found")
)
