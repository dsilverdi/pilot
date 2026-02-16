package agent

import "errors"

var (
	ErrInvalidMaxTokens    = errors.New("max_tokens must be greater than 0")
	ErrInvalidTemperature  = errors.New("temperature must be between 0 and 1")
	ErrNoToolRegistry      = errors.New("tool registry is required")
	ErrContextCanceled     = errors.New("context canceled")
)
