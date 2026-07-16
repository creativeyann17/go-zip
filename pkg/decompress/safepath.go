package decompress

import (
	"path/filepath"
	"strings"
)

// safeJoin joins an archive entry name onto outputDir and verifies the result
// still resolves inside outputDir (zip-slip protection).
func safeJoin(outputDir, entryName string) (string, error) {
	cleanOutputDir := filepath.Clean(outputDir)
	joined := filepath.Join(cleanOutputDir, entryName)
	if joined != cleanOutputDir &&
		!strings.HasPrefix(joined, cleanOutputDir+string(filepath.Separator)) {
		return "", ErrUnsafeEntryPath
	}
	return joined, nil
}
