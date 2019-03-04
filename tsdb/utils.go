package tsdb

import (
	//"time"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

func ParseFileName(dbbasepath, fullname string) uint32 {
	if fullname == "" {
		return uint32(0)
	}
	stringid := strings.TrimPrefix(fullname, dbbasepath+"-")
	stringid = strings.TrimSuffix(stringid, ".data")
	stringid = strings.TrimSuffix(stringid, ".labels")
	id, err := strconv.ParseUint(stringid, 16, 32)
	if err != nil {
		return uint32(0)
	}
	return uint32(id)
}

func GetDataFiles(dbbasepath string) []string {
	pattern := dbbasepath + "-[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]" + ".data"
	matches, _ := filepath.Glob(pattern)
	sort.Strings(matches)
	return matches
}

func GetLastFile(dbbasepath string) string {
	matches := GetDataFiles(dbbasepath)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return ""
}

// Returns the id of the file to open for writes.
// It is either the id of the last file written to, or 1 in case no last file
// can be determined. 0 is a reserved value, which indicates errors / uninitialized.
func GetFileId(dbbasepath string) uint32 {
	id := ParseFileName(dbbasepath, GetLastFile(dbbasepath))
	if id == 0 {
		return 1
	}
	return id
}

func MakeFileName(dbbasepath string, number uint32, extension string) string {
	return fmt.Sprintf("%s-%08x.%s", dbbasepath, number, extension)
}

func MakeDataStoreFileName(dbbasepath string, number uint32) string {
	return MakeFileName(dbbasepath, number, "data")
}

func MakeLabelStoreFileName(dbbasepath string, number uint32) string {
	return MakeFileName(dbbasepath, number, "labels")
}

func mmapFile(f *os.File, flags int) ([]byte, error) {
	st, err := f.Stat()
	if err != nil {
		return []byte{}, err
	}
	size := st.Size()
	if int64(int(size)) != size {
		return []byte{}, fmt.Errorf("size of %d overflows int", size)
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|flags, syscall.MAP_SHARED)
	if err != nil {
		return []byte{}, err
	}

	err = syscall.Mlock(data)
	return data[:size], err
}

func MultipleOfPageSize(value int) int {
	ps := os.Getpagesize()
	return (value + ps - 1) / ps * ps
}
