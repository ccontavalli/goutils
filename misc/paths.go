package misc

import (
	"path"
	"strings"
)

// Like path.Dir, but returns an empty string "" for an empty
// relative path, instead of ".", and does no path cleaning.
// This is useful when working on fragments of a path, for example,
// and cleaning is left to the caller.
func NaiveDir(fspath string) string {
	if len(fspath) == 1 {
		return fspath
	}

	slash := strings.LastIndexByte(fspath, '/')
	if slash < 0 {
		return ""
	}
	return fspath[:slash]
}

// Like path.Clean, but preserves a final / if there.
func CleanPreserveSlash(fspath string) string {
	result := path.Clean(fspath)
	if fspath[len(fspath)-1] == '/' {
		return result + "/"
	}
	return result
}
