package api

import "errors"

// Errors of the speaking endpoints, mapped to HTTP statuses by the handler.
var (
	ErrInvalidContent     = errors.New("invalid speaking content")
	ErrContentFlagged     = errors.New("this content can't be used: it was flagged by moderation")
	ErrTestHasAttempts    = errors.New("this test has attempts and can't be changed or deleted")
	ErrTooManyCustomTests = errors.New("you have reached the limit of custom speaking tests")
)
