package gutesting

import (
	"os"
	"time"
)

type MockFileInfo struct {
	NameValue    string
	SizeValue    int64
	ModeValue    os.FileMode
	ModTimeValue time.Time
	IsDirValue   bool
	SysValue     interface{}
}

func (mfi *MockFileInfo) Name() string {
	return mfi.NameValue
}

func (mfi *MockFileInfo) Size() int64 {
	return mfi.SizeValue
}

func (mfi *MockFileInfo) Mode() os.FileMode {
	return mfi.ModeValue
}

func (mfi *MockFileInfo) ModTime() time.Time {
	return mfi.ModTimeValue
}

func (mfi *MockFileInfo) IsDir() bool {
	return mfi.IsDirValue
}

func (mfi *MockFileInfo) Sys() interface{} {
	return mfi.SysValue
}
