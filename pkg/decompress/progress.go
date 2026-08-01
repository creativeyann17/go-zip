package decompress

import (
	"github.com/creativeyann17/go-zip/internal/progress"
	"github.com/vbauerster/mpb/v8"
)

func ProgressBarCallback() (ProgressCallback, *mpb.Progress) {
	return progress.ProgressBarCallback()
}

func FormatSummary(result *Result) string {
	return progress.FormatSummary(result, progress.OperationDecompress, false)
}

func FormatSize(bytes uint64) string {
	return progress.FormatSize(bytes)
}
