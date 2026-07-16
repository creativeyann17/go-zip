package decompress

import "errors"

var (
	ErrInputRequired   = errors.New("input archive path is required")
	ErrInvalidArchive  = errors.New("invalid archive format")
	ErrFileExists      = errors.New("file exists (use --overwrite to replace)")
	ErrUnsafeEntryPath = errors.New("entry path escapes output directory")
	ErrNotZIP          = errors.New("not a ZIP archive (expected PK signature)")
)
