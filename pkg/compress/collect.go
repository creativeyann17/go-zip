package compress

import (
	"fmt"
	"os"
	"path/filepath"
)

type fileTask struct {
	AbsPath  string
	RelPath  string
	Info     os.FileInfo
	OrigSize uint64
}

type folderTask struct {
	FolderPath string
	Files      []fileTask
}

// ProgressCallback is called for various progress events.
type ProgressCallback func(event ProgressEvent)

// ProgressEvent contains progress information.
type ProgressEvent struct {
	Type           EventType
	FilePath       string
	Current        int64
	Total          int64
	CurrentBytes   uint64
	TotalBytes     uint64
	CompressedSize uint64
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

func collectFiles(opts *Options, result *Result) ([]folderTask, int, uint64, error) {
	folderMap := make(map[string][]fileTask)
	seenRelPaths := make(map[string]string)
	var totalOrigSize uint64
	var totalFiles int

	addFile := func(absPath, relPath string, info os.FileInfo, source string) error {
		if existingSource, exists := seenRelPaths[relPath]; exists {
			return fmt.Errorf("path overlap: %q from %q conflicts with %q", relPath, source, existingSource)
		}
		seenRelPaths[relPath] = source

		folderPath := filepath.Dir(relPath)
		if folderPath == "." {
			folderPath = ""
		}

		task := fileTask{
			AbsPath:  absPath,
			RelPath:  relPath,
			Info:     info,
			OrigSize: uint64(info.Size()),
		}
		folderMap[folderPath] = append(folderMap[folderPath], task)
		totalOrigSize += uint64(info.Size())
		totalFiles++
		return nil
	}

	if len(opts.Files) > 0 {
		for _, inputPath := range opts.Files {
			cleanPath := filepath.Clean(inputPath)
			info, err := os.Stat(cleanPath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", inputPath, err))
				continue
			}

			if info.IsDir() {
				var matcher *gitignoreMatcher
				if opts.UseGitignore {
					matcher, _ = newGitignoreMatcher(cleanPath)
				}
				dirBase := filepath.Base(cleanPath)
				err := filepath.Walk(cleanPath, func(path string, finfo os.FileInfo, err error) error {
					if err != nil {
						result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
						return nil
					}
					relToDir, _ := filepath.Rel(cleanPath, path)
					if finfo.IsDir() {
						if path != cleanPath && matcher != nil && matcher.ShouldIgnoreDir(relToDir) {
							return filepath.SkipDir
						}
						return nil
					}
					if !finfo.Mode().IsRegular() {
						return nil
					}
					if matcher != nil && matcher.ShouldIgnore(relToDir) {
						return nil
					}
					relPath := filepath.Join(dirBase, relToDir)
					if err := addFile(path, relPath, finfo, inputPath); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return nil, 0, 0, err
				}
			} else if info.Mode().IsRegular() {
				relPath := filepath.Base(cleanPath)
				if err := addFile(cleanPath, relPath, info, inputPath); err != nil {
					return nil, 0, 0, err
				}
			}
		}
	} else {
		baseDir := opts.InputPath
		var matcher *gitignoreMatcher
		if opts.UseGitignore {
			matcher, _ = newGitignoreMatcher(baseDir)
		}
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
				return nil
			}
			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				relPath = filepath.Base(path)
			}
			if info.IsDir() {
				if path != baseDir && matcher != nil && matcher.ShouldIgnoreDir(relPath) {
					return filepath.SkipDir
				}
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			if matcher != nil && matcher.ShouldIgnore(relPath) {
				return nil
			}
			if err := addFile(path, relPath, info, baseDir); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, 0, 0, fmt.Errorf("directory walk failed: %w", err)
		}
	}

	foldersToCompress := make([]folderTask, 0, len(folderMap))
	for folderPath, files := range folderMap {
		foldersToCompress = append(foldersToCompress, folderTask{
			FolderPath: folderPath,
			Files:      files,
		})
	}
	return foldersToCompress, totalFiles, totalOrigSize, nil
}
