package progression

import "errors"

// ErrNotFound means there is no user row to read progression from.
var ErrNotFound = errors.New("user progression not found")
