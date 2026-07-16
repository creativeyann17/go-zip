package compress

import "errors"

var (
	// ErrInputRequired is returned when input path is not specified
	ErrInputRequired = errors.New("input path is required")

	// ErrInvalidLevel is returned when compression level is out of range
	ErrInvalidLevel = errors.New("compression level for ZIP (deflate) must be between 1 and 9")

	// ErrNoFiles is returned when no files are found to compress
	ErrNoFiles = errors.New("no regular files found to compress")
)
