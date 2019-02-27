package tsdb

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestOptions(t *testing.T) {
	options := DefaultDataWriterOptions()
	options.MaxEntries = 32

	assert.Equal(t, int64(0), options.GetFileSize()%4096)
	assert.Greater(t, options.GetFileSize(), int64(0))
	assert.Nil(t, options.Valid())
}

func TestStore(t *testing.T) {
	options := DefaultDataWriterOptions()
	options.MaxEntries = 32

	assert.Equal(t, int64(0), options.GetFileSize()%4096)
	assert.Equal(t, uint16(32), options.GetEntrySize())
	assert.Equal(t, int64(127), options.GetEntries())
	assert.Greater(t, options.GetFileSize(), int64(0))
	assert.Nil(t, options.Valid())

	tempdir, err := ioutil.TempDir("", "datastore-")
	assert.Nil(t, err)

	db, err := OpenDataWriter(filepath.Join(tempdir, "test"), options)
	assert.Nil(t, err)

	for i := uint64(0); i < 200; i++ {
		db.Append(i, i+1024, nil)

		// Read the value we just inserted.
		time, value, labels := db.GetOne(-1)
		assert.Equal(t, i, time)
		assert.Equal(t, i+1024, value)
		assert.Equal(t, 0, len(labels))

		// Before now, the ring was not full.
		if i > 127 {
			time, value, labels := db.GetOne(0)
			assert.Equal(t, i-125, time)
			assert.Equal(t, i+1024-125, value)
			assert.Equal(t, 0, len(labels))

			time, value, labels = db.GetOne(1)
			assert.Equal(t, i-124, time)
			assert.Equal(t, i+1024-124, value)
			assert.Equal(t, 0, len(labels))

			time, value, labels = db.GetOne(2)
			assert.Equal(t, i-123, time)
			assert.Equal(t, i+1024-123, value)
			assert.Equal(t, 0, len(labels))
		}
	}
}
