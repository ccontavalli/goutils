package tsdb

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	// "fmt"
)

func TestInvalidPath(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "serie-invalid-")
	assert.Nil(t, err)

	s := NewSerie(filepath.Join(tempdir, "invalid-directory", "more", "test"))
	s.MaxEntries = 32
	s.LabelBlock = 128

	err = s.Open()
	assert.NotNil(t, err)
	c, ok := err.(*os.PathError)
	assert.True(t, ok)
	assert.Equal(t, "open", c.Op)
	assert.Equal(t, syscall.Errno(0x2), c.Err)
}

func TestSerieBasics(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "serie-")
	assert.Nil(t, err)

	// Open the time series.
	s := NewSerie(filepath.Join(tempdir, "test"))
	s.MaxEntries = 32
	s.LabelBlock = 128
	err = s.Open()
	assert.Nil(t, err)
	for i := uint64(0); i < 2000; i++ {
		// Two labels are recycled.
		// Two labels are created new per entry.
		// One label never gets stored (configured for 4 labels).
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
	files1, err := filepath.Glob(filepath.Join(tempdir, "test") + "*")
	assert.Nil(t, err)

	basepath := filepath.Join(tempdir, "test")
	lastfile := GetLastFile(basepath)
	assert.NotEqual(t, "", lastfile)
	id := ParseFileName(basepath, lastfile)
	assert.Equal(t, uint32(16), id)

	// Reopen it again, append more.
	err = s.Open()
	assert.Nil(t, err)
	for i := uint64(0); i < 2000; i++ {
		// Two labels are recycled.
		// Two labels are created new per entry.
		// One label never gets stored (configured for 4 labels).
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

	files2, err := filepath.Glob(filepath.Join(tempdir, "test") + "*")
	assert.Equal(t, 2*len(files1), len(files2))
	assert.Nil(t, err)
}
