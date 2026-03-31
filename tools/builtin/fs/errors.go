package fs

import "errors"

var (
	ErrMissingPath               = errors.New("path is required")
	ErrMissingPathOrContent      = errors.New("path and content are required")
	ErrMissingPathOrInstructions = errors.New("path and instructions are required")
)
