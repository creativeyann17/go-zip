package verify

import (
	"fmt"

	"github.com/creativeyann17/go-zip/internal/progress"
)

// Result contains comprehensive verification results.
type Result struct {
	ArchivePath    string
	ArchiveSize    uint64
	HeaderValid    bool
	FileCount      int
	TotalOrigSize  uint64
	TotalCompSize  uint64
	EmptyFiles     int
	DataVerified   bool
	FilesVerified  int
	CorruptFiles   int
	StructureValid bool
	FooterValid    bool
	MetadataValid  bool
	DuplicatePaths int
	Files          []FileInfo
	Errors         []error
}

// FileInfo contains information about a single file in the archive.
type FileInfo struct {
	Path           string
	OriginalSize   uint64
	CompressedSize uint64
	DataValid      bool
	Error          error
}

func (r *Result) CompressionRatio() float64 {
	if r.TotalOrigSize == 0 {
		return 0
	}
	return float64(r.TotalCompSize) / float64(r.TotalOrigSize) * 100
}

func (r *Result) SpaceSaved() uint64 {
	if r.TotalCompSize >= r.TotalOrigSize {
		return 0
	}
	return r.TotalOrigSize - r.TotalCompSize
}

func (r *Result) SpaceSavedRatio() float64 {
	if r.TotalOrigSize == 0 {
		return 0
	}
	return float64(r.SpaceSaved()) / float64(r.TotalOrigSize) * 100
}

func (r *Result) IsValid() bool {
	return r.HeaderValid && r.StructureValid && r.FooterValid &&
		len(r.Errors) == 0 && r.CorruptFiles == 0
}

func (r *Result) Success() bool {
	return r.IsValid()
}

// Summary returns a human-readable summary of the verification result.
func (r *Result) Summary() string {
	status := "VALID"
	if !r.IsValid() {
		status = "INVALID"
	}

	s := fmt.Sprintf("Archive: %s [%s]\n", r.ArchivePath, status)
	s += "Format:  ZIP\n"
	s += fmt.Sprintf("Size:    %s\n", progress.FormatSize(r.ArchiveSize))
	s += fmt.Sprintf("Files:   %d\n", r.FileCount)

	if r.TotalOrigSize > 0 {
		s += fmt.Sprintf("Original:   %s\n", progress.FormatSize(r.TotalOrigSize))
		s += fmt.Sprintf("Compressed: %s (%.1f%% ratio)\n",
			progress.FormatSize(r.TotalCompSize), r.CompressionRatio())
		s += fmt.Sprintf("Saved:      %s (%.1f%%)\n",
			progress.FormatSize(r.SpaceSaved()), r.SpaceSavedRatio())
	}

	if r.DataVerified {
		s += "\nData Integrity:\n"
		s += fmt.Sprintf("  Files Verified:  %d/%d\n", r.FilesVerified, r.FileCount)
		if r.CorruptFiles > 0 {
			s += fmt.Sprintf("  Corrupt Files:   %d\n", r.CorruptFiles)
		}
	}

	if len(r.Errors) > 0 {
		s += fmt.Sprintf("\nErrors (%d):\n", len(r.Errors))
		for i, err := range r.Errors {
			if i >= 10 {
				s += fmt.Sprintf("  ... and %d more errors\n", len(r.Errors)-10)
				break
			}
			s += fmt.Sprintf("  - %v\n", err)
		}
	}

	return s
}
