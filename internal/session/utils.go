package session

import (
	"os"
	"path/filepath"
	"time"
)

// findNewestFile returns the newest file matching a glob pattern along with its modification time.
// Returns empty string and zero time if no files match.
func findNewestFile(pattern string) (string, time.Time) {
	files, _ := filepath.Glob(pattern)
	if len(files) == 0 {
		return "", time.Time{}
	}

	var newestPath string
	var newestTime time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestPath = f
		}
	}
	return newestPath, newestTime
}
