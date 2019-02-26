package tsdb

import (
	//"time"
	"fmt"
	"os"
	"syscall"
)

type Serie struct {
	CollectLabels float32

	*DataStore
	*LabelStore
}

func mmapFile(f *os.File) ([]byte, error) {
	st, err := f.Stat()
	if err != nil {
		return []byte{}, err
	}
	size := st.Size()
	if int64(int(size)) != size {
		return []byte{}, fmt.Errorf("size of %d overflows int", size)
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return []byte{}, err
	}

	err = syscall.Mlock(data)
	return data[:size], err
}

func Open() (*Serie, error) {
	return nil, nil
}

func (s *Serie) Sync() {
}

func (s *Serie) Close() {
}

func (s *Serie) Clean() {
}
