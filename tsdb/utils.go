package tsdb

import (
	//"time"
	"fmt"
	"os"
	"syscall"
)

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
