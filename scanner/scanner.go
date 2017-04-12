package scanner

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type ProcessDir func(state interface{}, path string, info os.FileInfo, files []os.FileInfo) (interface{}, error)
type ProcessFile func(state interface{}, path string, file os.FileInfo) error
type ProcessError func(state interface{}, path string, err error)

// Used internally, to keep track of the configuration of each directory.
type scanDirState struct {
	state   interface{}
	path    string
	info    os.FileInfo
	parents map[string]struct{}
}

// Why don't we use fileutil.Walk?
// - we want to handle symlinks
// - fileutil.Walk does a depth first walk, while we need a breadth first walk
// - has logic to create and pass over state that changes by directory
//
// How do we prevent symlink loop? There are two kinds of loops:
// - symlink a points to itself, or points to b which points to a or ...
//   Those are handled at the syscall layer, will return ELOOP.
// - a subdirectory pointing to a parent directory, or a subdirectory
//   with a link to another directory which in turn contains a link pointing
//   back to itself. These are detected by the code.
func ScanTree(base string, state interface{}, pdir ProcessDir, pfile ProcessFile, perr ProcessError) error {
	// Directories to recurse into.
	dirs := NewScanDirStateQueue(128)
	dirs.Push(scanDirState{state, filepath.Clean(base), nil, make(map[string]struct{})})

	/* Pseudo code:
	   - read all files in the directory.
	   - look for a configuration file, if there is one, merge it.

	   - for each file:
	     - is this a directory? accumulate it for later.
	     - is this a file? check the extension, and associate an handler with it.

	   - move on to the subdirectories. */
	for dirs.Len() > 0 {
		dirstate := dirs.Pop()
		state := dirstate.state
		dir := dirstate.path
		info := dirstate.info
		parents := dirstate.parents

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			perr(state, dir, err)
			continue
		}

		state, err = pdir(state, dir, info, files)
		if err != nil {
			perr(state, dir, err)
			continue
		}

		var sub_parents map[string]struct{}
		for _, file := range files {
			fullpath := filepath.Clean(path.Join(dir, file.Name()))

			absolute, err := filepath.Abs(fullpath)
			if err != nil {
				perr(state, fullpath, err)
				continue
			}

			// Check if the path was already walked - prevent symlink loops.
			// Note that the parents map is updated below, only if we find
			// a directory to get into.
			if _, found := parents[absolute]; found {
				perr(state, fullpath, fmt.Errorf("Skipping file: %s - symlink loop", absolute))
				continue
			}

			// Resolve symlinks into regular paths.
			if file.Mode()&os.ModeSymlink != 0 {
				newpath, err := filepath.EvalSymlinks(fullpath)
				if err != nil {
					perr(state, newpath, err)
					continue
				}

				file, err = os.Lstat(newpath)
				if err != nil {
					perr(state, newpath, err)
					continue
				}
			}

			switch mode := file.Mode(); {
			case mode.IsDir():
				if sub_parents == nil {
					// parents contains a set of parent directories. It is shared across all
					// child directories of a given directory. If we change it, we affect all
					// directories, and break symlink loop detection.
					// So, make a copy now, to be used later.
					sub_parents = make(map[string]struct{})
					for key, value := range parents {
						sub_parents[key] = value
					}
					sub_parents[absolute] = struct{}{}

				}
				dirs.Push(scanDirState{state, fullpath, file, sub_parents})

			case mode.IsRegular():
				err = pfile(state, fullpath, file)
				if err != nil {
					perr(state, fullpath, err)
					continue
				}

			default:
				perr(state, fullpath, fmt.Errorf("Invalid file type: %s", mode.String()))
			}
		}
	}

	return nil
}
