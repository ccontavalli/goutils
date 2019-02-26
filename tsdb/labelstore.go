package tsdb

import (
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type LabelOptions struct {
	Mode  os.FileMode
	Block uint32
}

type Label uint32

type LabelStore struct {
	file *os.File
	raw  []byte

	cache  map[string]Label
	offset int // Initialized by reloadCache

	blocksize uint32
}

func DefaultLabelOptions() LabelOptions {
	return LabelOptions{0666, 4 * 1048576}
}

func (lo LabelOptions) Valid() error {
	if lo.Block >= math.MaxInt32 {
		return fmt.Errorf("Block size is too large - would overflow int32")
	}
	return nil
}

func MultipleOfPageSize(value int64) int64 {
	ps := int64(os.Getpagesize())
	return (value + ps - 1) / ps * ps
}

func OpenLabels(dbbasepath string, options LabelOptions) (*LabelStore, error) {
	err := options.Valid()
	if err != nil {
		return nil, err
	}

	fullpath := dbbasepath + ".labels"
	file, err := os.OpenFile(fullpath, os.O_RDWR, options.Mode)
	if err != nil {
		file, err = os.Create(fullpath)
		if err != nil {
			return nil, err
		}
		err = file.Truncate(MultipleOfPageSize(int64(options.Block)))
		if err != nil {
			return nil, err
		}
	}

	data, err := mmapFile(file)
	if len(data) <= 0 {
		return nil, err
	}

	return &LabelStore{file, data, nil, 0, options.Block}, nil
}

func (ls *LabelStore) reloadCache() error {
	if ls.cache == nil {
		ls.cache = make(map[string]Label)
	}

	for offset := ls.offset; offset < len(ls.raw); {
		name, err := ls.LoadString(Label(offset))
		if err != nil {
			return err
		}
		if name == "" {
			ls.offset = offset
			break
		}
		ls.cache[name] = Label(offset)
		offset += (4 + len(name) + 7) / 8 * 8
	}

	return nil
}

func (ls *LabelStore) LoadString(label Label) (string, error) {
	offset := int(label)
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

func (ls *LabelStore) resizeFile(extrasize int) error {
	newsize := (len(ls.raw) + int(extrasize) + int(ls.blocksize) - 1) / int(ls.blocksize) * int(ls.blocksize)
	if newsize <= len(ls.raw) {
		return fmt.Errorf("Cannot increase file size - would overflow")
	}
	ls.file.Truncate(MultipleOfPageSize(int64(newsize)))

	newraw, err := mmapFile(ls.file)
	if err != nil {
		return err
	}
	syscall.Munmap(ls.raw)
	ls.raw = newraw
	return nil
}

func (ls *LabelStore) GetLabel(name string) (Label, error) {
	if ls.cache == nil {
		err := ls.reloadCache()
		if err != nil {
			return Label(0), err
		}
	}

	label, ok := ls.cache[name]
	if ok {
		return label, nil
	}

	if len(name)+int(ls.offset)+4 >= len(ls.raw) {
		err := ls.resizeFile(len(name))
		if err != nil {
			return Label(0), err
		}
	}

	label = Label(ls.offset)
	copy(ls.raw[ls.offset+4:], []byte(name))
	atomic.StoreUint32((*uint32)(unsafe.Pointer(&ls.raw[ls.offset])), uint32(len(name)))
	ls.offset += (4 + len(name) + 7) / 8 * 8
	ls.cache[name] = label
	return label, nil
}
