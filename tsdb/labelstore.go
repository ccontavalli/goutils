package tsdb

import (
	"fmt"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type LabelOptions struct {
	Mode       os.FileMode
	LabelBlock int
}

type LabelID uint32

type LabelStore struct {
	fullpath string
	raw      []byte

	cache  map[string]LabelID
	offset int // Initialized by reloadCache

	blocksize int
}

func DefaultLabelOptions() LabelOptions {
	return LabelOptions{0666, 4 * 1048576}
}

func (lo LabelOptions) Valid() error {
	if lo.LabelBlock >= math.MaxInt32 {
		return fmt.Errorf("LabelBlock size is too large - would overflow int32")
	}
	// In reality, we round to the page size, so any size >= 1 is good enough.
	// Minimum size to store a label is probably 5 bytes (4 uint32, and 1 byte of string),
	// 128 seems like a safe bet.
	if lo.LabelBlock < 128 {
		return fmt.Errorf("LabelBlock size is too small - needs to be >= 128")
	}
	return nil
}

func OpenLabelsForReading(fullpath string) (*LabelStore, error) {
	file, err := os.OpenFile(fullpath, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	data, err := mmapFile(file, syscall.PROT_READ)
	if len(data) <= 0 {
		return nil, err
	}

	return &LabelStore{fullpath, data, nil, 0, 0}, nil
}

func OpenLabelsForWriting(fullpath string, options LabelOptions) (*LabelStore, error) {
	err := options.Valid()
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(fullpath, os.O_RDWR, options.Mode)
	if err != nil {
		file, err = os.Create(fullpath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		err = file.Truncate(int64(MultipleOfPageSize(options.LabelBlock)))
		if err != nil {
			return nil, err
		}
	} else {
		defer file.Close()
	}

	data, err := mmapFile(file, syscall.PROT_WRITE)
	if len(data) <= 0 {
		return nil, err
	}

	return &LabelStore{fullpath, data, nil, 0, options.LabelBlock}, nil
}

func (ls *LabelStore) reloadCache() error {
	if ls.cache == nil {
		ls.cache = make(map[string]LabelID)
	}

	for offset := ls.offset; offset < len(ls.raw); {
		label := LabelID(offset + 1)
		name, err := ls.LoadString(label)
		if err != nil {
			return err
		}
		if name == "" {
			ls.offset = offset
			break
		}
		ls.cache[name] = label
		offset += (4 + len(name) + 7) / 8 * 8
	}

	return nil
}

func (ls *LabelStore) LoadString(label LabelID) (string, error) {
	offset := int(label) - 1
	if offset+4 >= len(ls.raw) {
		return "", fmt.Errorf("Label points outside the file - invalid")
	}
	strsize := atomic.LoadUint32((*uint32)(unsafe.Pointer(&ls.raw[offset])))
	if strsize == 0 {
		return "", nil
	}
	if strsize >= math.MaxInt32 {
		return "", fmt.Errorf("Size of string would overflow - invalid")
	}
	if int(strsize)+offset+4 > len(ls.raw) {
		return "", fmt.Errorf("Label too long ends after the file - invalid")
	}
	return string(ls.raw[offset+4 : offset+4+int(strsize)]), nil
}

func (ds *LabelStore) Sync() {
	unix.Msync(ds.raw, unix.MS_SYNC|unix.MS_INVALIDATE)
}

func (ls *LabelStore) Seal() {
	id, err := ls.CreateLabel("")
	if err != nil {
		file, err := os.OpenFile(ls.fullpath, os.O_RDWR, 0666)
		if err != nil {
			file.Truncate(int64(MultipleOfPageSize(int(id) + 4 - 1)))
			file.Close()
		}
	}
	ls.Close()
}

func (ls *LabelStore) Close() {
	ls.Sync()
	syscall.Munmap(ls.raw)
	ls.cache = nil
}

func (ls *LabelStore) resizeFile(extrasize int) error {
	newsize := (len(ls.raw) + extrasize + ls.blocksize - 1) / ls.blocksize * ls.blocksize
	if newsize <= len(ls.raw) {
		return fmt.Errorf("Cannot increase file size - would overflow")
	}

	file, err := os.OpenFile(ls.fullpath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Truncate(int64(MultipleOfPageSize(newsize)))

	newraw, err := mmapFile(file, syscall.PROT_WRITE)
	if err != nil {
		return err
	}
	syscall.Munmap(ls.raw)
	ls.raw = newraw
	return nil
}

// Creates a new label in the database, and returns its LabelID if successful.
// If the label already exists, the existing id is returned.
func (ls *LabelStore) CreateLabel(name string) (LabelID, error) {
	if ls.cache == nil {
		err := ls.reloadCache()
		if err != nil {
			return LabelID(0), err
		}
	}

	label, ok := ls.cache[name]
	if ok {
		return label, nil
	}

	if len(name)+int(ls.offset)+4 >= len(ls.raw) {
		err := ls.resizeFile(len(name))
		if err != nil {
			return LabelID(0), err
		}
	}

	label = LabelID(ls.offset + 1)
	copy(ls.raw[ls.offset+4:], []byte(name))
	atomic.StoreUint32((*uint32)(unsafe.Pointer(&ls.raw[ls.offset])), uint32(len(name)))
	ls.offset += (4 + len(name) + 7) / 8 * 8
	ls.cache[name] = label
	return label, nil
}
