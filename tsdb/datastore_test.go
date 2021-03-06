package tsdb

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestOptions(t *testing.T) {
	options := DefaultDataStoreOptions()
	options.MaxEntries = 32

	assert.Equal(t, 0, options.GetFileSize()%4096)
	assert.Greater(t, options.GetFileSize(), 0)
	assert.Nil(t, options.Valid())
}

func TestStore(t *testing.T) {
	options := DefaultDataStoreOptions()
	options.MaxEntries = 32

	assert.Equal(t, 0, options.GetFileSize()%4096)
	assert.Equal(t, 32, GetEntrySize(options.LabelsPerEntry))
	assert.Equal(t, 127, options.GetMaxEntries())
	assert.Greater(t, options.GetFileSize(), 0)
	assert.Nil(t, options.Valid())

	tempdir, err := ioutil.TempDir("", "datastore-")
	assert.Nil(t, err)

	db, err := OpenDataStoreForWriting(filepath.Join(tempdir, "test"), options)
	assert.Nil(t, err)

	// There are 127 slots in the ring. Up to then, we should be
	// able to fill them up with no issue.
	for i := uint64(0); i < 127; i++ {
		result, last := db.Append(i, i+1024, nil)
		assert.True(t, result)
		assert.Equal(t, (i+1)*32, last)

		// Read the value we just inserted.
		time, value, labels := db.GetOne(-1)
		assert.Equal(t, i, time)
		assert.Equal(t, i+1024, value)
		assert.Equal(t, 0, len(labels))

		entries := db.GetEntries()
		assert.Equal(t, int(i+1), entries)
	}

	// Read back all 127 values in right order.
	for i := int(0); i < 127; i++ {
		time, value, labels := db.GetOne(i)
		assert.Equal(t, uint64(i), time)
		assert.Equal(t, uint64(i+1024), value)
		assert.Equal(t, 0, len(labels))
	}

	// Read back all 127 values in reverse order.
	for i := int(-127); i < 0; i++ {
		time, value, labels := db.GetOne(i)
		assert.Equal(t, uint64(127+i), time)
		assert.Equal(t, uint64(127+i+1024), value)
		assert.Equal(t, 0, len(labels))
	}

}
