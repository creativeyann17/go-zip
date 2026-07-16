package verify

import "errors"

var (
	ErrInputRequired     = errors.New("input path is required")
	ErrInvalidMagic      = errors.New("invalid archive magic bytes")
	ErrCorruptData       = errors.New("data corruption detected")
	ErrUnsupportedFormat = errors.New("unsupported archive format")
)
