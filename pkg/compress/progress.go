package compress

import (
	"github.com/creativeyann17/go-zip/internal/progress"
	"github.com/vbauerster/mpb/v8"
)

// ProgressBarCallback creates multi-progress bars for compression.
func ProgressBarCallback() (ProgressCallback, *mpb.Progress) {
	return progress.ProgressBarCallback()
}

// FormatSummary formats a compression result.
func FormatSummary(result *Result, opts *Options) string {
	isDryRun := opts != nil && opts.DryRun
	return progress.FormatSummary(result, progress.OperationCompress, isDryRun)
}

// FormatSize formats bytes into a human-readable string.
func FormatSize(bytes uint64) string {
	return progress.FormatSize(bytes)
}
