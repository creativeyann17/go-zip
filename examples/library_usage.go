// Example: using go-zip as a library.
//
//	go run ./examples/library_usage.go
//
// Paths below are placeholders — replace with real dirs before running.

package main

import (
	"fmt"
	"log"

	"github.com/creativeyann17/go-zip/pkg/compress"
	"github.com/creativeyann17/go-zip/pkg/decompress"
)

func main() {
	opts := compress.DefaultOptions()
	opts.InputPath = "/path/to/files"
	opts.OutputPath = "backup"
	opts.MaxThreads = 4
	opts.Level = 6
	opts.Quiet = true

	result, err := compress.Compress(opts, nil)
	if err != nil {
		log.Fatalf("compress: %v", err)
	}
	fmt.Printf("Compressed %d files → %s (%.1f%%)\n",
		result.FilesProcessed, compress.FormatSize(result.CompressedSize), result.CompressionRatio())

	// Multi-part: open first part; library discovers siblings.
	dopts := decompress.DefaultOptions()
	dopts.InputPath = "backup_01.zip"
	dopts.OutputPath = "/path/to/restore"
	dopts.Overwrite = true
	dopts.Quiet = true

	dresult, err := decompress.Decompress(dopts, nil)
	if err != nil {
		log.Fatalf("decompress: %v", err)
	}
	fmt.Printf("Extracted %d files\n", dresult.FilesProcessed)
}
