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

type DataStore struct {
	// Name of the backing file, so we can Open() it.
	name string
	// Mapping of the entire file content in memory, header + ring.
	// Created with mmap(), cleaned with munmap().
	raw []byte

	// Pointer to the header of the ring containing the offset
	// of the next slot in the ring to write.
	// Read/Write it using atomic operations, update only after
	// writing the new entry in the ring (so readers always observe
	// a consistent state.
	cursor *uint64
	// Number of entries in the ring (eg, size in bytes / size of entry).
	entries int

	// Mapping of the file containing the timestamps and values only.
	// It is a slice over raw.
	ring []byte
	// Labels per entry, number of labels to store for each entry.
	lpe int
}

type DataStoreOptions struct {
	// Unix mode to open the file as. 0666 by default.
	Mode os.FileMode

	// Number of different labels to keep associated with each time entry. 4 by default.
	// Cannot exceed 256.
	LabelsPerEntry int
	// Maximum numbers of entries to store in the time database.
	// Note that this is rounded to fill a multiple of the page size.
	MaxEntries int
}

func GetEntrySize(lpe int) int {
	return lpe*4 + 8 + 8
}

func GetHeaderSize() int {
	return 16
}

func (do DataStoreOptions) GetMaxEntries() int {
	return do.GetRingSize() / GetEntrySize(do.LabelsPerEntry)
}
func (do DataStoreOptions) GetRingSize() int {
	return do.GetFileSize() - GetHeaderSize()
}

func (do DataStoreOptions) GetFileSize() int {
	return int(MultipleOfPageSize(GetHeaderSize() + GetEntrySize(do.LabelsPerEntry)*do.MaxEntries))
}

func (do DataStoreOptions) Valid() error {
	if unsafe.Sizeof(uint64(0)) != 8 {
		return fmt.Errorf("Unsupported platform - uin64 is not 8 bytes")
	}
	if do.LabelsPerEntry < 0 || do.LabelsPerEntry > 256 {
		return fmt.Errorf("LabelsPerEntry too large, must be <= 256")
	}

	filesize := do.GetFileSize()
	if filesize > math.MaxInt32 || filesize < 0 {
		return fmt.Errorf("Number of entries or labels per entry too large - would overflow int32")
	}
	return nil
}

func DefaultDataStoreOptions() DataStoreOptions {
	return DataStoreOptions{0666, 4, 604800}
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

func CreateDataStore(filename string, data []byte) *DataStore {
	cursor := (*uint64)(unsafe.Pointer(&data[0]))
	lpe := int(*(*uint8)(unsafe.Pointer(&data[8])))
	ring := data[GetHeaderSize():]
	entries := len(ring) / GetEntrySize(lpe)

	return &DataStore{filename, data, cursor, entries, ring, lpe}
}

func OpenDataStoreForReading(dbasefile string) (*DataStore, error) {
	data := []byte{}
	file, err := os.OpenFile(dbasefile, os.O_RDONLY, 0666)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	data, err = mmapFile(file, unix.PROT_READ)
	if len(data) <= 0 {
		return nil, err
	}

	return CreateDataStore(dbasefile, data), nil
}

func OpenDataStoreForWriting(dbasefile string, options DataStoreOptions) (*DataStore, error) {
	err := options.Valid()
	if err != nil {
		return nil, err
	}

	// This will either open the existing specified id, or create a file with the correct name.
	data := []byte{}
	for {
		file, err := os.OpenFile(dbasefile, os.O_RDWR, options.Mode)
		defer file.Close()

		if err == nil {
			data, err = mmapFile(file, unix.PROT_WRITE)
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

		err = file.Truncate(int64(options.GetFileSize()))
		if err != nil {
			os.Remove(file.Name())
			return nil, err
		}

		data, err = mmapFile(file, unix.PROT_WRITE)
		if len(data) <= 0 {
			os.Remove(file.Name())
			return nil, err
		}
		*(*uint64)(unsafe.Pointer(&data[0])) = uint64(0)
		*(*uint8)(unsafe.Pointer(&data[8])) = uint8(options.LabelsPerEntry)

		// err = unix.RenameAt2(unix.AT_FDCWD, file.Name(), unix.AT_FDCWD, fullpath, unix.RENAME_NOREPLACE)
		err = unix.Rename(file.Name(), dbasefile)
		if err == nil {
			break
		}

		unix.Munmap(data)
		os.Remove(file.Name())

		if !os.IsExist(err) {
			return nil, err
		}
	}
	return CreateDataStore(dbasefile, data), nil
}

func PeekDataStore(dbasefile string) (Point, int, error) {
	file, err := os.OpenFile(dbasefile, os.O_RDONLY, 0666)
	defer file.Close()

	if err != nil {
		return Point{}, 0, err
	}
	st, err := file.Stat()
	if err != nil {
		return Point{}, 0, err
	}
	size := st.Size()
	if int64(int(size)) != size {
		return Point{}, 0, fmt.Errorf("size of %d overflows int", size)
	}

	buffer := make([]byte, GetEntrySize(0)+GetHeaderSize())
	n, err := file.Read(buffer)
	if err != nil {
		return Point{}, 0, err
	}
	if n != len(buffer) {
		return Point{}, 0, fmt.Errorf("file did not have enough bytes to read - %d", n)
	}

	cursor := (*uint64)(unsafe.Pointer(&buffer[0]))
	lpe := int(*(*uint8)(unsafe.Pointer(&buffer[8])))

	time := *(*uint64)(unsafe.Pointer(&buffer[GetHeaderSize()]))
	value := *(*uint64)(unsafe.Pointer(&buffer[GetHeaderSize()+8]))

	last := atomic.LoadUint64(cursor)
	return Point{time, value, nil}, GetEntries(last, int(size)-GetHeaderSize(), lpe), nil
}

func (ds *DataStore) Sync() {
	unix.Msync(ds.raw, unix.MS_SYNC|unix.MS_INVALIDATE)
}

func (ds *DataStore) Close() {
	ds.Sync()
	unix.Munmap(ds.raw)
}

type Offset int

func GetEntries(last uint64, ringlen int, lpe int) int {
	if last >= uint64(ringlen) {
		last = uint64(ringlen)
	}
	return int(last) / GetEntrySize(lpe)
}

func (ds *DataStore) GetEntries() int {
	last := atomic.LoadUint64(ds.cursor)
	return GetEntries(last, len(ds.ring), ds.lpe)
}

func (ds *DataStore) GetOffset(element int) Offset {
	if (element > 0 && element >= ds.entries) || (element < 0 && element+1 <= -ds.entries) {
		panic(fmt.Sprintf("invalid index %d, when only %d elements are reachable", element, ds.entries))
	}

	entry := GetEntrySize(ds.lpe)
	if element < 0 {
		cursor := atomic.LoadUint64(ds.cursor)
		element = int(cursor) + (entry * element)
	} else {
		element *= entry
	}
	return Offset(element)
}

func (ds *DataStore) GetTime(offset Offset) uint64 {
	return *(*uint64)(unsafe.Pointer(&ds.ring[offset]))
}
func (ds *DataStore) GetValue(offset Offset) uint64 {
	return *(*uint64)(unsafe.Pointer(&ds.ring[offset+8]))
}

func (ds *DataStore) GetLabels(offset Offset, labels []LabelID) []LabelID {
	if labels == nil {
		labels = make([]LabelID, 0, ds.lpe)
	}
	for i := 0; i < ds.lpe; i++ {
		label := *(*uint32)(unsafe.Pointer(&ds.ring[int(offset)+GetEntrySize(i)]))
		if label == 0 {
			break
		}
		labels = append(labels, LabelID(label))
	}
	return labels
}

func (ds *DataStore) GetOne(element int) (time, value uint64, labels []LabelID) {
	offset := ds.GetOffset(element)
	time = ds.GetTime(offset)
	value = ds.GetValue(offset)

	labels = ds.GetLabels(offset, nil)

	return time, value, labels
}

func (ds *DataStore) Seal() {
	appended, last := ds.Append(0xffffffffffffffff, 0xffffffffffffffff, nil)
	if appended {
		newsize := MultipleOfPageSize(GetHeaderSize() + int(last))
		// Mode when opening an existing file is ignored.
		file, err := os.OpenFile(ds.name, os.O_RDWR, 0666)
		defer file.Close()
		if err == nil {
			file.Truncate(int64(newsize))
		}
	}
	ds.Sync()
	ds.Close()
}

func (ds *DataStore) PeekAppend() (bool, uint64) {
	last := atomic.LoadUint64(ds.cursor)
	if last+uint64(GetEntrySize(ds.lpe)) >= uint64(len(ds.ring)) {
		return false, last
	}
	return true, last
}

func (ds *DataStore) Append(time, value uint64, labels []LabelID) (bool, uint64) {
	last := atomic.LoadUint64(ds.cursor)
	if last+uint64(GetEntrySize(ds.lpe)) >= uint64(len(ds.ring)) {
		return false, last
	}

	*(*uint64)(unsafe.Pointer(&ds.ring[last])) = time
	last += 8
	*(*uint64)(unsafe.Pointer(&ds.ring[last])) = value
	last += 8

	for i := 0; i < len(labels) && i < ds.lpe; i++ {
		*(*uint32)(unsafe.Pointer(&ds.ring[int(last)+i*4])) = uint32(labels[i])
	}
	last += uint64(ds.lpe * 4)
	atomic.StoreUint64(ds.cursor, last)
	return true, last
}
