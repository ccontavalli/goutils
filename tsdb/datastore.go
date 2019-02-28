package tsdb

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sync/atomic"
	"unsafe"
	//"log"
)

type DataWriter struct {
	file *os.File
	raw  []byte

	cursor *uint64
	esize  int

	ring []byte
	lpe  uint8
}

type DataWriterOptions struct {
	// Unix mode to open the file as. 0666 by default.
	Mode os.FileMode

	// Number of different labels to keep associated with each time entry. 4 by default.
	LabelsPerEntry uint8
	// Maximum numbers of entries to store in the time database.
	// Note that this is rounded to fill a multiple of the page size.
	MaxEntries uint32
}

func (do DataWriterOptions) GetEntrySize() uint16 {
	return uint16(do.LabelsPerEntry)*4 + 8 + 8
}

func GetHeaderSize() int64 {
	return 16
}

func (do DataWriterOptions) GetEntries() int64 {
	return do.GetRingSize() / int64(do.GetEntrySize())
}
func (do DataWriterOptions) GetRingSize() int64 {
	return int64(do.GetFileSize() - GetHeaderSize())
}

func (do DataWriterOptions) GetFileSize() int64 {
	return MultipleOfPageSize(GetHeaderSize() + int64(do.GetEntrySize())*int64(do.MaxEntries))
}

func (do DataWriterOptions) Valid() error {
	filesize := do.GetFileSize()
	if filesize > math.MaxInt32 || filesize < 0 {
		return fmt.Errorf("Number of entries or labels per entry too large - would overflow uint32")
	}
	return nil
}

func DefaultDataWriterOptions() DataWriterOptions {
	return DataWriterOptions{0666, 4, 604800}
}

// Format of a .data file:
//   [ 0 -  7] - 8 bytes - uint64 - cursor - where the next value should be written.
//   [   8   ] - 1 byte  - uint8 - lpe - number of labels per entry in this datafile.
//   [ 9 - 15] - 7 bytes - unused
//   [16 - ..] - x bytes - entries in the ring.
//
// Format of a ring entry:
//   [ 0 -  7] - 8 bytes - uint64 - timestamp.
//   [ 8 - 15] - 8 bytes - uint64 - value.
//   [16 - ..] - 4 bytes each - uint32 - labels, one uint32 per labelid, up to lpe labels.
//               Unused labels are set to 0
//
// Note that the entire file size is rounded to PAGE_SIZE.
func OpenDataWriter(dbasefile string, options DataWriterOptions) (*DataWriter, error) {
	if unsafe.Sizeof(uint64(0)) != 8 {
		return nil, fmt.Errorf("Unsupported platform - uin64 is not 8 bytes")
	}
	err := options.Valid()
	if err != nil {
		return nil, err
	}

	data := []byte{}
	file := &os.File{}

	// This will either open the existing specified id, or create a file with the correct name.
	for {
		file, err := os.OpenFile(dbasefile, os.O_RDWR, options.Mode)
		if err == nil {
			data, err = mmapFile(file)
			if len(data) <= 0 {
				return nil, err
			}
			break
		}

		if !os.IsNotExist(err) {
			return nil, err
		}

		dir, name := filepath.Split(dbasefile)
		file, err = ioutil.TempFile(dir, name)
		if err != nil {
			return nil, err
		}

		err = file.Truncate(options.GetFileSize())
		if err != nil {
			os.Remove(file.Name())
			file.Close()
			return nil, err
		}

		data, err = mmapFile(file)
		if len(data) <= 0 {
			os.Remove(file.Name())
			file.Close()
			return nil, err
		}
		*(*uint64)(unsafe.Pointer(&data[0])) = uint64(0)
		*(*uint8)(unsafe.Pointer(&data[8])) = options.LabelsPerEntry

		// err = unix.RenameAt2(unix.AT_FDCWD, file.Name(), unix.AT_FDCWD, fullpath, unix.RENAME_NOREPLACE)
		err = unix.Rename(file.Name(), dbasefile)
		if err == nil {
			break
		}

		unix.Munmap(data)
		os.Remove(file.Name())
		file.Close()

		if !os.IsExist(err) {
			return nil, err
		}
	}

	cursor := (*uint64)(unsafe.Pointer(&data[0]))
	lpe := *(*uint8)(unsafe.Pointer(&data[8]))
	ring := data[16:]
	esize := len(ring) / int(options.GetEntrySize())

	return &DataWriter{file, data, cursor, esize, ring, lpe}, nil
}

func (ds *DataWriter) GetEntrySize() uint16 {
	return uint16(8 + 8 + ds.lpe*4)
}

func (ds *DataWriter) Sync() {
	unix.Msync(ds.raw, unix.MS_SYNC|unix.MS_INVALIDATE)
}

func (ds *DataWriter) Close() {
	ds.Sync()
	ds.file.Close()
}

func (ds *DataWriter) GetOne(element int) (time, value uint64, labels []LabelID) {
	if (element > 0 && element >= ds.esize) || (element < 0 && element+1 <= -ds.esize) {
		panic(fmt.Sprintf("invalid index %d, when only %d elements are reachable", element, ds.esize))
	}

	entry := int(ds.GetEntrySize())
	if element < 0 {
		cursor := atomic.LoadUint64(ds.cursor)
		element = int(cursor) + (entry * element)
	} else {
		element *= entry
	}

	time = *(*uint64)(unsafe.Pointer(&ds.ring[element]))
	value = *(*uint64)(unsafe.Pointer(&ds.ring[element+8]))
	for i := 0; i < int(ds.lpe); i++ {
		label := *(*uint32)(unsafe.Pointer(&ds.ring[int(element)+16+i*4]))
		if label == 0 {
			break
		}
		labels = append(labels, LabelID(label))
	}
	return time, value, labels
}

func (ds *DataWriter) Seal() {
	appended, last := ds.Append(0xffffffffffffffff, 0xffffffffffffffff, nil)
	if appended {
		newsize := MultipleOfPageSize(GetHeaderSize() + int64(last))
		atomic.StoreUint64(ds.cursor, uint64(newsize-GetHeaderSize()))
		ds.file.Truncate(newsize)
	}
	ds.Sync()
	ds.Close()
}

func (ds *DataWriter) Peek() bool {
	last := atomic.LoadUint64(ds.cursor)
	if last+uint64(ds.GetEntrySize()) >= uint64(len(ds.ring)) {
		return false
	}
	return true
}

func (ds *DataWriter) Append(time, value uint64, labels []LabelID) (bool, uint64) {
	last := atomic.LoadUint64(ds.cursor)
	if last+uint64(ds.GetEntrySize()) >= uint64(len(ds.ring)) {
		return false, last
	}

	*(*uint64)(unsafe.Pointer(&ds.ring[last])) = time
	last += 8
	*(*uint64)(unsafe.Pointer(&ds.ring[last])) = value
	last += 8

	i := 0
	for ; i < len(labels) && i < int(ds.lpe); i++ {
		*(*uint32)(unsafe.Pointer(&ds.ring[last])) = uint32(labels[i])
		last += 4
	}
	for ; i < int(ds.lpe); i++ {
		*(*uint32)(unsafe.Pointer(&ds.ring[last])) = 0
		last += 4
	}
	atomic.StoreUint64(ds.cursor, last)
	return true, last
}
