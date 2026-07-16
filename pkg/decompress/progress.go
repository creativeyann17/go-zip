package decompress

import (
	"github.com/creativeyann17/go-zip/internal/progress"
	"github.com/vbauerster/mpb/v8"
)

func ProgressBarCallback() (ProgressCallback, *mpb.Progress) {
	genericCb, p := progress.ProgressBarCallback()
	callback := func(event ProgressEvent) {
		genericCb(progress.Event{
			Type:         progress.EventType(event.Type),
			FilePath:     event.FilePath,
			Current:      event.Current,
			Total:        event.Total,
			CurrentBytes: event.CurrentBytes,
			TotalBytes:   event.TotalBytes,
		})
	}
	return callback, p
}

func FormatSummary(result *Result) string {
	return progress.FormatSummary(result, progress.OperationDecompress, false)
}

func FormatSize(bytes uint64) string {
	return progress.FormatSize(bytes)
}
