package compress

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/creativeyann17/go-zip/internal/multipart"
	"github.com/klauspost/compress/flate"
)

// progressReportStep is the minimum bytes between EventFileProgress emissions.
const progressReportStep = 1 << 20

// Compress compresses files into ZIP archive(s).
// MaxThreads == 1 → single base.zip; MaxThreads > 1 → base_01.zip, base_02.zip, …
func Compress(opts *Options, progressCb ProgressCallback) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	result := &Result{}
	foldersToCompress, totalFiles, totalOrigSize, err := collectFiles(opts, result)
	if err != nil {
		return nil, err
	}
	if totalFiles == 0 {
		return nil, ErrNoFiles
	}

	result.FilesTotal = totalFiles
	result.OriginalSize = totalOrigSize

	if progressCb != nil {
		progressCb(ProgressEvent{
			Type:       EventStart,
			Total:      int64(totalFiles),
			TotalBytes: totalOrigSize,
		})
	}

	if err := compressToZip(opts, progressCb, foldersToCompress, totalFiles, result); err != nil {
		return result, err
	}
	return result, nil
}

func compressToZip(opts *Options, progressCb ProgressCallback, foldersToCompress []folderTask, totalFiles int, result *Result) error {
	if opts.DisableGC {
		runtime.GC()
		oldGCPercent := debug.SetGCPercent(-1)
		defer debug.SetGCPercent(oldGCPercent)
	}

	baseOutputPath := multipart.NormalizeBase(opts.OutputPath)
	multiPart := opts.MaxThreads > 1

	var totalCompSize atomic.Uint64
	var processedCount atomic.Uint32
	var errorsMu sync.Mutex

	var wg sync.WaitGroup

	allTasks := make([]fileTask, 0, totalFiles)
	for _, folder := range foldersToCompress {
		allTasks = append(allTasks, folder.Files...)
	}
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].OrigSize > allTasks[j].OrigSize
	})
	taskCh := make(chan fileTask, opts.MaxThreads*16)

	type zipFileInfo struct {
		path string
		size uint64
	}
	zipFiles := make([]zipFileInfo, opts.MaxThreads)
	var zipFilesMu sync.Mutex
	var partCounter atomic.Int32

	for i := 0; i < opts.MaxThreads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var workerZipWriter *zip.Writer
			var workerZipFile *os.File
			var workerZipPath string

			ensureArchive := func() error {
				if workerZipFile != nil {
					return nil
				}
				var partNum int
				if multiPart {
					partNum = int(partCounter.Add(1))
				} else {
					partNum = 1
				}
				workerZipPath = multipart.OutputPath(baseOutputPath, multiPart, partNum)

				outputDir := filepath.Dir(workerZipPath)
				if err := os.MkdirAll(outputDir, 0750); err != nil {
					return fmt.Errorf("worker %d: create output directory: %w", workerID, err)
				}

				var err error
				workerZipFile, err = os.Create(workerZipPath)
				if err != nil {
					return fmt.Errorf("worker %d: create zip: %w", workerID, err)
				}

				workerZipWriter = zip.NewWriter(workerZipFile)
				workerZipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
					if opts.Level <= 1 {
						return flate.NewWriter(out, flate.NoCompression)
					}
					flateLevel := opts.Level - 1
					if flateLevel > flate.BestCompression {
						flateLevel = flate.BestCompression
					}
					return flate.NewWriter(out, flateLevel)
				})

				zipFilesMu.Lock()
				zipFiles[workerID].path = workerZipPath
				zipFilesMu.Unlock()
				return nil
			}

			for task := range taskCh {
				if !opts.DryRun {
					if err := ensureArchive(); err != nil {
						errorsMu.Lock()
						result.Errors = append(result.Errors, err)
						errorsMu.Unlock()
						return
					}
				}

				if progressCb != nil && task.OrigSize > 0 {
					progressCb(ProgressEvent{
						Type:     EventFileStart,
						FilePath: task.RelPath,
						Total:    int64(task.OrigSize),
					})
				}

				file, err := os.Open(task.AbsPath)
				if err != nil {
					errorsMu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("%s: open: %w", task.RelPath, err))
					errorsMu.Unlock()
					if progressCb != nil {
						progressCb(ProgressEvent{Type: EventError, FilePath: task.RelPath})
					}
					continue
				}

				if !opts.DryRun && workerZipWriter != nil {
					header := &zip.FileHeader{
						Name:   task.RelPath,
						Method: zip.Deflate,
					}
					if opts.Level == 1 {
						header.Method = zip.Store
					}

					w, err := workerZipWriter.CreateHeader(header)
					if err != nil {
						_ = file.Close()
						errorsMu.Lock()
						result.Errors = append(result.Errors, fmt.Errorf("%s: create header: %w", task.RelPath, err))
						errorsMu.Unlock()
						continue
					}

					buf := getReadBuffer()
					var written, lastReported int64
					for {
						nr, errRead := file.Read(buf)
						if nr > 0 {
							nw, errWrite := w.Write(buf[0:nr])
							if errWrite != nil {
								_ = file.Close()
								errorsMu.Lock()
								result.Errors = append(result.Errors, fmt.Errorf("%s: write: %w", task.RelPath, errWrite))
								errorsMu.Unlock()
								break
							}
							written += int64(nw)
							if progressCb != nil && written-lastReported >= progressReportStep {
								lastReported = written
								progressCb(ProgressEvent{
									Type:     EventFileProgress,
									FilePath: task.RelPath,
									Current:  written,
									Total:    int64(task.OrigSize),
								})
							}
						}
						if errRead == io.EOF {
							break
						}
						if errRead != nil {
							_ = file.Close()
							errorsMu.Lock()
							result.Errors = append(result.Errors, fmt.Errorf("%s: read: %w", task.RelPath, errRead))
							errorsMu.Unlock()
							break
						}
					}
					putReadBuffer(buf)
				} else if opts.DryRun {
					totalCompSize.Add(task.OrigSize / 2)
				}

				_ = file.Close()
				processedCount.Add(1)
				if progressCb != nil {
					progressCb(ProgressEvent{
						Type:     EventFileComplete,
						FilePath: task.RelPath,
						Current:  int64(task.OrigSize),
						Total:    int64(task.OrigSize),
					})
				}
			}

			if !opts.DryRun && workerZipFile != nil {
				if err := workerZipWriter.Close(); err != nil {
					errorsMu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("worker %d: close zip: %w", workerID, err))
					errorsMu.Unlock()
					return
				}
				if err := workerZipFile.Close(); err != nil {
					errorsMu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("worker %d: close file: %w", workerID, err))
					errorsMu.Unlock()
					return
				}
				if stat, err := os.Stat(workerZipPath); err == nil {
					zipFilesMu.Lock()
					zipFiles[workerID].size = uint64(stat.Size())
					zipFilesMu.Unlock()
				}
			}
		}(i)
	}

	go func() {
		for _, task := range allTasks {
			taskCh <- task
		}
		close(taskCh)
	}()

	wg.Wait()
	result.FilesProcessed = int(processedCount.Load())

	if !opts.DryRun {
		var totalSize uint64
		var partCount int
		for _, info := range zipFiles {
			if info.size > 0 {
				totalSize += info.size
				partCount++
			}
		}
		result.CompressedSize = totalSize

		if opts.Verbose && !opts.Quiet {
			if multiPart {
				fmt.Printf("\nCreated %d ZIP files:\n", partCount)
			} else {
				fmt.Printf("\nCreated ZIP file:\n")
			}
			for _, info := range zipFiles {
				if info.size > 0 {
					fmt.Printf("  %s (%.2f MB)\n",
						filepath.Base(info.path), float64(info.size)/(1024*1024))
				}
			}
		}
	} else {
		result.CompressedSize = totalCompSize.Load()
	}

	if progressCb != nil {
		progressCb(ProgressEvent{
			Type:           EventComplete,
			Current:        int64(result.FilesProcessed),
			Total:          int64(totalFiles),
			CompressedSize: result.CompressedSize,
		})
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("completed with %d errors (see result.Errors)", len(result.Errors))
	}
	return nil
}
