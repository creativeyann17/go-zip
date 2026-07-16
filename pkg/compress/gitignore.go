package compress

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// gitignoreMatcher handles .gitignore pattern matching with hierarchy support.
type gitignoreMatcher struct {
	baseDir  string
	matchers map[string]*ignore.GitIgnore
}

func newGitignoreMatcher(baseDir string) (*gitignoreMatcher, error) {
	baseDir = filepath.Clean(baseDir)
	gm := &gitignoreMatcher{
		baseDir:  baseDir,
		matchers: make(map[string]*ignore.GitIgnore),
	}

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) != ".gitignore" {
			return nil
		}

		dir := filepath.Dir(path)
		relDir, err := filepath.Rel(baseDir, dir)
		if err != nil {
			return nil
		}
		if relDir == "." {
			relDir = ""
		}

		matcher, err := ignore.CompileIgnoreFile(path)
		if err != nil {
			return nil
		}
		gm.matchers[relDir] = matcher
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(gm.matchers) == 0 {
		return nil, nil
	}
	return gm, nil
}

func (gm *gitignoreMatcher) ShouldIgnore(relPath string) bool {
	if gm == nil || len(gm.matchers) == 0 {
		return false
	}
	relPath = filepath.ToSlash(relPath)
	for _, dirPath := range gm.buildHierarchy(relPath) {
		matcher, exists := gm.matchers[dirPath]
		if !exists {
			continue
		}
		var pathToCheck string
		if dirPath == "" {
			pathToCheck = relPath
		} else {
			pathToCheck = strings.TrimPrefix(relPath, dirPath+"/")
		}
		if matcher.MatchesPath(pathToCheck) {
			return true
		}
	}
	return false
}

func (gm *gitignoreMatcher) ShouldIgnoreDir(relPath string) bool {
	if gm == nil || len(gm.matchers) == 0 {
		return false
	}
	matchesWithSlash := gm.ShouldIgnore(relPath + "/")
	matchesWithoutSlash := gm.ShouldIgnore(relPath)
	return matchesWithSlash && !matchesWithoutSlash
}

func (gm *gitignoreMatcher) buildHierarchy(relPath string) []string {
	relPath = filepath.ToSlash(relPath)
	parentDir := filepath.Dir(relPath)
	if parentDir == "." {
		parentDir = ""
	}
	parentDir = filepath.ToSlash(parentDir)

	hierarchy := []string{""}
	if parentDir == "" {
		return hierarchy
	}

	parts := strings.Split(parentDir, "/")
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}
		hierarchy = append(hierarchy, current)
	}
	sort.Slice(hierarchy, func(i, j int) bool {
		return len(hierarchy[i]) < len(hierarchy[j])
	})
	return hierarchy
}
