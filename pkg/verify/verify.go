package verify

import (
	"archive/zip"
	"fmt"
	"io"
	"os"

	"github.com/creativeyann17/go-zip/internal/multipart"
	"github.com/creativeyann17/go-zip/internal/pathutil"
)

// ProgressCallback is called for progress updates during verification.
type ProgressCallback func(event ProgressEvent)

// ProgressEvent contains progress information.
type ProgressEvent struct {
	Type     EventType
	FilePath string
	Current  int
	Total    int
	Message  string
}

// EventType indicates the type of progress event.
type EventType int

const (
	EventStart EventType = iota
	EventFileVerify
	EventComplete
	EventError
)

// maxVerifyEntrySize caps decompressed bytes per entry during --data (bomb guard).
const maxVerifyEntrySize = 10 << 30 // 10 GiB

func copyWithLimit(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	written, err := io.CopyN(dst, src, limit+1)
	if err == io.EOF {
		return written, nil
	}
	if err != nil {
		return written, err
	}
	return written, fmt.Errorf("entry exceeds %d byte verify limit, possible decompression bomb", limit)
}

// Verify verifies a ZIP archive (single or multi-part).
func Verify(opts *Options, progressCb ProgressCallback) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	result := &Result{
		ArchivePath: opts.InputPath,
	}

	// Check magic on primary path
	f, err := os.Open(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("read magic: %w", err)
	}
	_ = f.Close()
	if magic[0] != 'P' || magic[1] != 'K' {
		result.HeaderValid = false
		return result, ErrInvalidMagic
	}

	zipPaths, err := multipart.DiscoverParts(opts.InputPath)
	if err != nil {
		return nil, err
	}

	result.HeaderValid = true
	result.MetadataValid = true
	pathTracker := pathutil.NewPathTracker()

	for _, zipPath := range zipPaths {
		stat, err := os.Stat(zipPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("stat %s: %w", zipPath, err))
			continue
		}
		result.ArchiveSize += uint64(stat.Size())

		if err := verifyZipPart(zipPath, opts, progressCb, result, pathTracker); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("verify %s: %w", zipPath, err))
		}
	}

	result.StructureValid = result.HeaderValid && result.MetadataValid && result.DuplicatePaths == 0
	result.FooterValid = true // ZIP central directory serves as footer

	if progressCb != nil {
		progressCb(ProgressEvent{
			Type:    EventComplete,
			Current: result.FileCount,
			Total:   result.FileCount,
			Message: "Verification complete",
		})
	}

	return result, nil
}

func verifyZipPart(zipPath string, opts *Options, progressCb ProgressCallback, result *Result, pathTracker *pathutil.PathTracker) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		result.HeaderValid = false
		return fmt.Errorf("open zip: %w", err)
	}
	defer func() { _ = zipReader.Close() }()

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fileInfo := FileInfo{
			Path:           file.Name,
			OriginalSize:   file.UncompressedSize64,
			CompressedSize: file.CompressedSize64,
		}

		if pathTracker.CheckDuplicate(file.Name) {
			result.DuplicatePaths++
			result.Errors = append(result.Errors, fmt.Errorf("duplicate path: %s", file.Name))
		}

		result.FileCount++
		result.TotalOrigSize += file.UncompressedSize64
		result.TotalCompSize += file.CompressedSize64
		if file.UncompressedSize64 == 0 {
			result.EmptyFiles++
		}

		if progressCb != nil {
			progressCb(ProgressEvent{
				Type:     EventFileVerify,
				FilePath: file.Name,
				Current:  result.FileCount,
				Total:    result.FileCount,
			})
		}

		if opts.VerifyData {
			rc, err := file.Open()
			if err != nil {
				fileInfo.Error = fmt.Errorf("open: %w", err)
				result.CorruptFiles++
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", file.Name, err))
				result.Files = append(result.Files, fileInfo)
				continue
			}

			written, err := copyWithLimit(io.Discard, rc, maxVerifyEntrySize)
			_ = rc.Close()

			if err != nil {
				fileInfo.Error = fmt.Errorf("decompress: %w", err)
				result.CorruptFiles++
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", file.Name, err))
			} else if uint64(written) != file.UncompressedSize64 {
				fileInfo.Error = fmt.Errorf("size mismatch: expected %d, got %d", file.UncompressedSize64, written)
				result.CorruptFiles++
				result.Errors = append(result.Errors, fmt.Errorf("%s: %v", file.Name, fileInfo.Error))
			} else {
				fileInfo.DataValid = true
				result.FilesVerified++
			}
			result.DataVerified = true
		}

		result.Files = append(result.Files, fileInfo)
	}

	return nil
}
