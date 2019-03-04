package tsdb

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestSerieReaderInvalidPath(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "serie-")
	assert.Nil(t, err)

	// Open the time series.
	s := NewSerieReader(filepath.Join(tempdir, "test"))
	err = s.Open()
	assert.NotNil(t, err)
}

func TestSerieReaderBasics(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "serie-")
	assert.Nil(t, err)

	// Open the time series.
	s := NewSerieWriter(filepath.Join(tempdir, "test"))
	s.MaxEntries = 32
	s.LabelBlock = 128
	err = s.Open()
	assert.Nil(t, err)
	for i := uint64(0); i < 2000; i++ {
		labels := []string{
			fmt.Sprintf("foo-%d.1", i),
			fmt.Sprintf("foo.2"),
			fmt.Sprintf("foo-%d.3", i),
			fmt.Sprintf("foo.4"),
			fmt.Sprintf("foo-%d.5", i),
		}
		err := s.Append(i, i+1024, labels)
		assert.Nil(t, err)
	}
	s.Close()

	r := NewSerieReader(filepath.Join(tempdir, "test"))
	assert.NotNil(t, r)
	err = r.Open()
	assert.Nil(t, err)
	assert.Equal(t, 16, len(r.shard))

	// Whitebox tests, verifying integrity of the internal data structures.
	for i, shard := range r.shard {
		assert.Nil(t, shard.dw)
		assert.Nil(t, shard.ls)
		assert.Equal(t, uint64(i*127), shard.mintime)
		//  The last shard is not full.
		if i != 15 {
			assert.Equal(t, 127, shard.entries)
		} else {
			assert.Equal(t, 95, shard.entries)
		}
	}

	first := r.FirstLocation()
	assert.Equal(t, 0, first.shard.index)
	assert.Equal(t, 0, first.element)
	last := r.LastLocation()
	assert.Equal(t, 15, last.shard.index)
	assert.Equal(t, 95, last.element)

	data, err := r.GetData(first, last, nil)
	assert.Nil(t, err)
	assert.Equal(t, 15*127+95, len(data))

	// Read one element at a time throughout the full range.
	end := last
	cursor := last.Minus(r, 1)
	for ; cursor != first; cursor = cursor.Minus(r, 1) {
		data, err := r.GetData(cursor, end, nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(data))
		end = cursor
	}

	// Read 2, 4, 6, 8 ... elements, starting from the end.
	cursor = last.Offset(r, -2)
	i := 2
	for ; cursor != first; cursor = cursor.Offset(r, -2) {
		data, err := r.GetData(cursor, last, nil)
		assert.Nil(t, err)
		assert.Equal(t, i, len(data))
		i += 2
	}

	// Read a few hundred elements at a time, to test the Minus
	// logic when crossing multiple shards at once.
	cursor = last.Offset(r, -390)
	data, err = r.GetData(cursor, last, nil)
	assert.Nil(t, err)
	assert.Equal(t, 390, len(data))
	cursor = cursor.Offset(r, -10000)
	data, err = r.GetData(cursor, last, nil)
	assert.Nil(t, err)
	assert.Equal(t, 2000, len(data))

	// Test increments now.
	cursor = first
	for cursor != last {
		end := cursor.Plus(r, 1)
		data, err := r.GetData(cursor, end, nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(data))
		cursor = end
	}

	// Read 2, 4, 6, 8 ... elements, starting from the end.
	cursor = first
	i = 0
	for ; cursor != last; cursor = cursor.Offset(r, 2) {
		data, err := r.GetData(cursor, last, nil)
		assert.Nil(t, err)
		assert.Equal(t, 2000-i, len(data))
		i += 2
	}

	// Read a few hundred elements at a time, to test the Plus
	// logic when crossing multiple shards at once.
	cursor = first.Offset(r, 390)
	data, err = r.GetData(first, cursor, nil)
	assert.Nil(t, err)
	assert.Equal(t, 390, len(data))
	cursor = cursor.Offset(r, 10000)
	data, err = r.GetData(first, cursor, nil)
	assert.Nil(t, err)
	assert.Equal(t, 2000, len(data))
}
