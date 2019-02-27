package tsdb

import (
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"unsafe"
	"golang.org/x/sys/unix"
	//"log"
)

type DataStore struct {
	file *os.File
	raw  []byte

	cursor *uint64
	esize  int

	ring []byte
	lpe  uint8
}

type DataOptions struct {
	// Unix mode to open the file as. 0666 by default.
	Mode os.FileMode

	// Number of different labels to keep associated with each time entry. 4 by default.
	LabelsPerEntry uint8
	// Maximum numbers of entries to store in the time database.
	// Note that this is rounded to fill a multiple of the page size.
	MaxEntries uint32
}

func (do DataOptions) GetEntrySize() uint16 {
	return uint16(do.LabelsPerEntry)*4 + 8 + 8
}

func (do DataOptions) GetEntries() int64 {
	return do.GetRingSize() / int64(do.GetEntrySize())
}
func (do DataOptions) GetRingSize() int64 {
	return int64(do.GetFileSize() - 8)
}

func (do DataOptions) GetFileSize() int64 {
	return MultipleOfPageSize(8 + int64(do.GetEntrySize())*int64(do.MaxEntries))
}

func (do DataOptions) Valid() error {
	filesize := do.GetFileSize()
	if filesize > math.MaxInt32 || filesize < 0 {
		return fmt.Errorf("Number of entries or labels per entry too large - would overflow uint32")
	}
	return nil
}

func DefaultDataOptions() DataOptions {
	return DataOptions{0666, 4, 604800}
}

func OpenData(dbbasepath string, options DataOptions) (*DataStore, error) {
	if unsafe.Sizeof(uint64(0)) != 8 {
		return nil, fmt.Errorf("Unsupported platform - uin64 is not 8 bytes")
	}
	err := options.Valid()
	if err != nil {
		return nil, err
	}

	fullpath := dbbasepath + ".data"
	file, err := os.OpenFile(fullpath, os.O_RDWR, options.Mode)
	if err != nil {
		file, err = os.Create(fullpath)
		if err != nil {
			return nil, err
		}
		err = file.Truncate(options.GetFileSize())
		if err != nil {
			return nil, err
		}
	}

	data, err := mmapFile(file)
	if len(data) <= 0 {
		return nil, err
	}

	cursor := (*uint64)(unsafe.Pointer(&data[0]))
	ring := data[8:]
	esize := len(ring) / int(options.GetEntrySize())

	return &DataStore{file, data, cursor, esize, ring, options.LabelsPerEntry}, nil
}

func (ds *DataStore) GetEntrySize() uint16 {
	return uint16(8 + 8 + ds.lpe*4)
}

func (ds *DataStore) Sync() {
	unix.Msync(ds.raw, unix.MS_SYNC | unix.MS_INVALIDATE)
}

func (ds *DataStore) Close() {
	ds.Sync()
	ds.file.Close()
}

/* TODO
type Result struct {
}

func (ds *DataStore) GetUntilGreater(value uint64) Result {
	return Result{}
}

func (ds *DataStore) GetLastN(entries int) Result {
	return Result{}
} */

func (ds *DataStore) GetOne(element int) (time, value uint64, labels []Label) {
	if (element > 0 && element >= ds.esize) || (element < 0 && element <= -ds.esize) {
		panic(fmt.Sprintf("invalid index %d, when only %d elements are reachable", element, ds.esize))
	}

	entry := int(ds.GetEntrySize())
	rsize := uint64(entry * ds.esize)

	cursor := atomic.LoadUint64(ds.cursor)
	offset := element * entry
	if offset < 0 {
		offset += int(rsize)
	} else {
		offset += entry
	}
	cursor += uint64(offset)
	if cursor >= rsize {
		cursor -= rsize
	}

	time = atomic.LoadUint64((*uint64)(unsafe.Pointer(&ds.ring[cursor])))
	value = atomic.LoadUint64((*uint64)(unsafe.Pointer(&ds.ring[cursor+8])))
	for i := 0; i < int(ds.lpe); i++ {
		label := atomic.LoadUint32((*uint32)(unsafe.Pointer(&ds.ring[int(cursor)+16+i*4])))
		if label == 0 {
			break
		}
		labels = append(labels, Label(label))
	}
	return time, value, labels
}

func (ds *DataStore) Append(time, value uint64, labels []Label) {
	last := atomic.LoadUint64(ds.cursor)
	if last+uint64(ds.GetEntrySize()) >= uint64(len(ds.ring)) {
		last = 0
	}
	atomic.StoreUint64((*uint64)(unsafe.Pointer(&ds.ring[last])), time)
	last += 8
	atomic.StoreUint64((*uint64)(unsafe.Pointer(&ds.ring[last])), value)
	last += 8

	i := 0
	for ; i < len(labels) && i < int(ds.lpe); i++ {
		atomic.StoreUint32((*uint32)(unsafe.Pointer(&ds.ring[last])), uint32(labels[i]))
		last += 4
	}
	for ; i < int(ds.lpe); i++ {
		atomic.StoreUint32((*uint32)(unsafe.Pointer(&ds.ring[last])), 0)
		last += 4
	}
	atomic.StoreUint64(ds.cursor, last)
}
