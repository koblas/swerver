package handler

import (
	"path/filepath"
	"runtime"
	"strings"
)

func pathIsInside(thePath, potentialParent string) bool {
	// For inside-directory checking, we want to allow trailing slashes, so normalize.
	thePath = stripTrailingSep(thePath)
	potentialParent = stripTrailingSep(potentialParent)

	if runtime.GOOS == "windows" {
		thePath = strings.ToLower(thePath)
		potentialParent = strings.ToLower(potentialParent)
	}

	// They are either the same or the path has subdirectories from here
	plen := len(potentialParent)
	return strings.HasPrefix(thePath, potentialParent) && (len(thePath) == plen || thePath[plen] == filepath.Separator)
}

func stripTrailingSep(thePath string) string {
	return strings.TrimRight(thePath, string(filepath.Separator))
}
