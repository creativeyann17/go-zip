// Package multipart handles ZIP multi-part archive naming and discovery.
package multipart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizeBase strips a trailing .zip from outputPath to get the archive base name.
func NormalizeBase(outputPath string) string {
	return strings.TrimSuffix(outputPath, ".zip")
}

// OutputPath returns the path for a worker's ZIP archive.
// When multiPart is false (single-thread mode), returns base.zip.
// When multiPart is true, returns base_NN.zip with 1-based partNum.
func OutputPath(base string, multiPart bool, partNum int) string {
	if !multiPart {
		return base + ".zip"
	}
	return fmt.Sprintf("%s_%02d.zip", base, partNum)
}

// IsMultiPartName reports whether name (basename) matches the base_NN.zip pattern.
func IsMultiPartName(name string) bool {
	if !strings.HasSuffix(name, ".zip") {
		return false
	}
	stem := name[:len(name)-4]
	idx := strings.LastIndex(stem, "_")
	if idx < 0 || idx == len(stem)-1 {
		return false
	}
	suffix := stem[idx+1:]
	if len(suffix) != 2 {
		return false
	}
	return suffix[0] >= '0' && suffix[0] <= '9' && suffix[1] >= '0' && suffix[1] <= '9'
}

// MultiPartBase returns the base pattern (without _NN.zip) for a multi-part name.
// name is the basename, e.g. "archive_01.zip" → "archive".
func MultiPartBase(name string) (base string, ok bool) {
	if !IsMultiPartName(name) {
		return "", false
	}
	stem := name[:len(name)-4]
	idx := strings.LastIndex(stem, "_")
	return stem[:idx], true
}

// DiscoverParts returns all ZIP paths for the given input.
//
// Rules:
//   - plain *.zip without _NN suffix → single-part only (no sibling scan)
//   - *_NN.zip → scan base_01.zip … base_99.zip until a gap
func DiscoverParts(inputPath string) ([]string, error) {
	baseName := filepath.Base(inputPath)

	if IsMultiPartName(baseName) {
		pattern, ok := MultiPartBase(baseName)
		if !ok {
			return []string{inputPath}, nil
		}
		dirPath := filepath.Dir(inputPath)
		var paths []string
		for i := 1; i <= 99; i++ {
			partPath := filepath.Join(dirPath, fmt.Sprintf("%s_%02d.zip", pattern, i))
			if _, err := os.Stat(partPath); err == nil {
				paths = append(paths, partPath)
			} else {
				break
			}
		}
		if len(paths) == 0 {
			return nil, fmt.Errorf("no multi-part archive files found matching pattern: %s_XX.zip", pattern)
		}
		return paths, nil
	}

	// Plain single archive — do not merge sibling multi-part sets
	return []string{inputPath}, nil
}

// ResolveInputPath expands a user-supplied path that may omit the .zip extension.
// Prefer base.zip, then base_01.zip.
func ResolveInputPath(inputPath string) string {
	if inputPath == "" {
		return inputPath
	}
	if strings.HasSuffix(inputPath, ".zip") {
		return inputPath
	}
	// Already exists as-is
	if _, err := os.Stat(inputPath); err == nil {
		return inputPath
	}
	single := inputPath + ".zip"
	if _, err := os.Stat(single); err == nil {
		return single
	}
	multi := inputPath + "_01.zip"
	if _, err := os.Stat(multi); err == nil {
		return multi
	}
	// Default to .zip; caller will surface open errors
	return single
}
