package decompress

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/creativeyann17/go-zip/internal/multipart"
	"github.com/klauspost/compress/flate"
)

const progressReportStep = 1 << 20

// ProgressCallback is called for various progress events.
type ProgressCallback func(event ProgressEvent)

// ProgressEvent contains progress information.
type ProgressEvent struct {
	Type             EventType
	FilePath         string
	Current          int64
	Total            int64
	CurrentBytes     uint64
	TotalBytes       uint64
	DecompressedSize uint64
}

// EventType indicates the type of progress event.
type EventType int

const (
	EventStart EventType = iota
	EventFileStart
	EventFileProgress
	EventFileComplete
	EventComplete
	EventError
)

// Decompress extracts a ZIP archive (single or multi-part) to OutputPath.
func Decompress(opts *Options, progressCb ProgressCallback) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	result := &Result{}

	// Reject non-ZIP magic on the primary path
	if err := checkZIPMagic(opts.InputPath); err != nil {
		return nil, err
	}

	zipPaths, err := multipart.DiscoverParts(opts.InputPath)
	if err != nil {
		return nil, err
	}

	var totalFiles int
	if !opts.Quiet && len(zipPaths) > 1 {
		fmt.Printf("Detecting multi-part archive: scanning %d parts...\n", len(zipPaths))
	}
	for _, zipPath := range zipPaths {
		zr, err := zip.OpenReader(zipPath)
		if err != nil {
			return nil, fmt.Errorf("open zip archive %s: %w", zipPath, err)
		}
		totalFiles += len(zr.File)
		_ = zr.Close()
	}
	if !opts.Quiet && len(zipPaths) > 1 {
		fmt.Printf("Found %d files across %d archive parts\n\n", totalFiles, len(zipPaths))
	}

	result.FilesTotal = totalFiles

	if progressCb != nil {
		progressCb(ProgressEvent{
			Type:  EventStart,
			Total: int64(totalFiles),
		})
	}

	workers := opts.MaxThreads
	if workers > len(zipPaths) {
		workers = len(zipPaths)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	pathCh := make(chan string)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for zipPath := range pathCh {
				if err := extractZipFile(zipPath, opts, progressCb, result, &mu); err != nil {
					mu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("extract %s: %w", zipPath, err))
					mu.Unlock()
				}
			}
		}()
	}

	for _, zipPath := range zipPaths {
		pathCh <- zipPath
	}
	close(pathCh)
	wg.Wait()

	if progressCb != nil {
		progressCb(ProgressEvent{
			Type:    EventComplete,
			Current: int64(result.FilesProcessed),
			Total:   int64(totalFiles),
		})
	}

	return result, nil
}

func checkZIPMagic(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("read magic: %w", err)
	}
	if magic[0] != 'P' || magic[1] != 'K' {
		return ErrNotZIP
	}
	return nil
}

func extractZipFile(zipPath string, opts *Options, progressCb ProgressCallback, result *Result, mu *sync.Mutex) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer func() { _ = zipReader.Close() }()

	zipReader.RegisterDecompressor(zip.Deflate, func(r io.Reader) io.ReadCloser {
		return flate.NewReader(r)
	})

	recordError := func(err error) {
		mu.Lock()
		result.Errors = append(result.Errors, err)
		mu.Unlock()
	}

	buf := make([]byte, 256*1024)

	for _, zipFile := range zipReader.File {
		if progressCb != nil {
			progressCb(ProgressEvent{
				Type:     EventFileStart,
				FilePath: zipFile.Name,
				Total:    int64(zipFile.UncompressedSize64),
			})
		}

		outPath, err := safeJoin(opts.OutputPath, zipFile.Name)
		if err != nil {
			recordError(fmt.Errorf("%s: %w", zipFile.Name, err))
			if progressCb != nil {
				progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
			}
			continue
		}

		if zipFile.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, 0750); err != nil {
				recordError(fmt.Errorf("%s: mkdir: %w", zipFile.Name, err))
			}
			mu.Lock()
			result.FilesProcessed++
			mu.Unlock()
			continue
		}

		if !opts.Overwrite {
			if _, err := os.Stat(outPath); err == nil {
				recordError(fmt.Errorf("%s: file exists (use --overwrite to replace)", zipFile.Name))
				if progressCb != nil {
					progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
				}
				continue
			}
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
			recordError(fmt.Errorf("%s: mkdir: %w", zipFile.Name, err))
			if progressCb != nil {
				progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
			}
			continue
		}

		rc, err := zipFile.Open()
		if err != nil {
			recordError(fmt.Errorf("%s: open: %w", zipFile.Name, err))
			if progressCb != nil {
				progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
			}
			continue
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			_ = rc.Close()
			recordError(fmt.Errorf("%s: create: %w", zipFile.Name, err))
			if progressCb != nil {
				progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
			}
			continue
		}

		var written, lastReported int64
		var copyErr error
		for {
			nr, errRead := rc.Read(buf)
			if nr > 0 {
				nw, errWrite := outFile.Write(buf[0:nr])
				if errWrite != nil {
					copyErr = errWrite
					recordError(fmt.Errorf("%s: write: %w", zipFile.Name, errWrite))
					if progressCb != nil {
						progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
					}
					break
				}
				written += int64(nw)
				if progressCb != nil && written-lastReported >= progressReportStep {
					lastReported = written
					progressCb(ProgressEvent{
						Type:     EventFileProgress,
						FilePath: zipFile.Name,
						Current:  written,
						Total:    int64(zipFile.UncompressedSize64),
					})
				}
			}
			if errRead == io.EOF {
				break
			}
			if errRead != nil {
				copyErr = errRead
				recordError(fmt.Errorf("%s: read: %w", zipFile.Name, errRead))
				if progressCb != nil {
					progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
				}
				break
			}
		}

		_ = rc.Close()
		if cerr := outFile.Close(); cerr != nil && copyErr == nil {
			copyErr = cerr
			recordError(fmt.Errorf("%s: close: %w", zipFile.Name, cerr))
			if progressCb != nil {
				progressCb(ProgressEvent{Type: EventError, FilePath: zipFile.Name})
			}
		}
		if copyErr != nil {
			continue
		}

		mu.Lock()
		result.FilesProcessed++
		result.DecompressedSize += zipFile.UncompressedSize64
		result.CompressedSize += zipFile.CompressedSize64
		mu.Unlock()

		if progressCb != nil {
			progressCb(ProgressEvent{
				Type:     EventFileComplete,
				FilePath: zipFile.Name,
				Current:  int64(zipFile.UncompressedSize64),
				Total:    int64(zipFile.UncompressedSize64),
			})
		}
	}

	return nil
}
