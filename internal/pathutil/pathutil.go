// Package pathutil provides path tracking helpers.
package pathutil

// PathTracker tracks seen paths and detects duplicates.
type PathTracker struct {
	seen map[string]bool
}

// NewPathTracker creates a new PathTracker.
func NewPathTracker() *PathTracker {
	return &PathTracker{seen: make(map[string]bool)}
}

// CheckDuplicate returns true if the path was already seen, otherwise marks it as seen.
func (pt *PathTracker) CheckDuplicate(path string) bool {
	if pt.seen[path] {
		return true
	}
	pt.seen[path] = true
	return false
}
