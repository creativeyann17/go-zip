package decompress

import (
	"io"
	"runtime"
)

// Options configures decompression behavior.
type Options struct {
	InputPath      string
	OutputPath     string
	MaxThreads     int
	Verbose        bool
	ProgressWriter io.Writer
	Quiet          bool
	Overwrite      bool
}

// DefaultOptions returns options with sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		MaxThreads: runtime.NumCPU(),
	}
}

// Validate checks if options are valid.
func (o *Options) Validate() error {
	if o.InputPath == "" {
		return ErrInputRequired
	}
	if o.OutputPath == "" {
		o.OutputPath = "."
	}
	if o.MaxThreads <= 0 {
		o.MaxThreads = runtime.NumCPU()
	}
	if o.Quiet {
		o.Verbose = false
	}
	return nil
}
