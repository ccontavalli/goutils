package tsdb

import (
	"fmt"
	"math"
	"os"
	"sync/atomic"
	"unsafe"
)

type DataStore struct {
	file *os.File
	raw  []byte

	cursor *uint64
	end    uint64

	ring []byte
	lpe  uint8
}

type DataOptions struct {
	Mode os.FileMode

	LabelsPerEntry uint8
	MaxEntries     uint32
}

func (do DataOptions) GetEntrySize() uint16 {
	return uint16(do.LabelsPerEntry)*4 + 8 + 8
}

func (do DataOptions) GetEntriesSize() int64 {
	return int64(do.GetEntrySize()) * int64(do.MaxEntries)
}
func (do DataOptions) GetFileSize() int64 {
	return 8 + do.GetEntriesSize()
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
		err = file.Truncate(MultipleOfPageSize(options.GetFileSize()))
		if err != nil {
			return nil, err
		}
	}

	data, err := mmapFile(file)
	if len(data) <= 0 {
		return nil, err
	}

	cursor := (*uint64)(unsafe.Pointer(&data[0]))
	end := uint64(len(data) - 8)
	ring := data[8:]

	return &DataStore{file, data, cursor, end, ring, options.LabelsPerEntry}, nil
}

func (ds *DataStore) GetRecordSize() uint16 {
	return uint16(8 + 8 + ds.lpe*4)
}

func (ds *DataStore) Append(time, value uint64, labels []Label) {
	last := atomic.LoadUint64(ds.cursor)
	if last+uint64(ds.GetRecordSize()) >= ds.end {
		last = 0
	}
	atomic.StoreUint64((*uint64)(unsafe.Pointer(&ds.ring[last])), time)
	last += 8
	atomic.StoreUint64((*uint64)(unsafe.Pointer(&ds.ring[last])), value)
	last += 8

	i := 0
	for ; i <= len(labels) && i <= int(ds.lpe); i++ {
		atomic.StoreUint32((*uint32)(unsafe.Pointer(&ds.ring[last])), uint32(labels[i]))
		last += 4
	}
	for ; i <= int(ds.lpe); i++ {
		atomic.StoreUint32((*uint32)(unsafe.Pointer(&ds.ring[last])), 0)
		last += 4
	}
	atomic.StoreUint64(ds.cursor, last)
}
