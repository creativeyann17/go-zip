package compress

import (
	"io"
	"runtime"
)

// Options configures ZIP compression behavior.
type Options struct {
	// Input path (file or directory). Ignored if Files is provided.
	InputPath string

	// Files allows library users to provide a custom list of files/folders.
	// When set, InputPath is ignored. Library use only.
	Files []string

	// Output archive path base (".zip" suffix optional).
	// MaxThreads == 1 → base.zip; MaxThreads > 1 → base_01.zip, base_02.zip, …
	OutputPath string

	// Maximum number of concurrent compression threads. Default: runtime.NumCPU()
	MaxThreads int

	// Compression level 1-9 (deflate). Level 1 uses Store (no compression).
	// Default: 5
	Level int

	// DryRun simulates compression without writing
	DryRun bool

	// Verbose enables detailed logging
	Verbose bool

	// ProgressWriter receives progress updates (optional)
	ProgressWriter io.Writer

	// Quiet suppresses all output except errors
	Quiet bool

	// UseGitignore respects .gitignore files to exclude matching paths
	UseGitignore bool

	// DisableGC disables garbage collection during compression.
	// Uses pooled buffers to minimize allocations. Default: false
	DisableGC bool
}

// DefaultOptions returns options with sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		MaxThreads: runtime.NumCPU(),
		Level:      5,
	}
}

// Validate checks if options are valid and applies defaults.
func (o *Options) Validate() error {
	if o.InputPath == "" && len(o.Files) == 0 {
		return ErrInputRequired
	}
	if o.OutputPath == "" {
		o.OutputPath = "archive"
	}
	if o.MaxThreads <= 0 {
		o.MaxThreads = runtime.NumCPU()
	}
	if o.Level == 0 {
		o.Level = 5
	}
	if o.Level < 1 || o.Level > 9 {
		return ErrInvalidLevel
	}
	if o.Quiet {
		o.Verbose = false
	}
	return nil
}
