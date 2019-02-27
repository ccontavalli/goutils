package tsdb

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
	"fmt"
)

func TestBasics(t *testing.T) {
	options := DefaultLabelOptions()
	options.Block = 64

	tempdir, err := ioutil.TempDir("", "labelstore-")
	assert.Nil(t, err)

	db1, err := OpenLabels(filepath.Join(tempdir, "test"), options)
	assert.Nil(t, err)
	assert.NotNil(t, db1)

	label, err := db1.GetLabel("some")
	assert.Nil(t, err)
	assert.Equal(t, 1, int(label))

	label, err = db1.GetLabel("animals")
	assert.Nil(t, err)
	assert.Equal(t, 9, int(label))

	label, err = db1.GetLabel("some")
	assert.Nil(t, err)
	assert.Equal(t, 1, int(label))

	label, err = db1.GetLabel("are")
	assert.Nil(t, err)
	assert.Equal(t, 25, int(label))

	for i := 0; i < 10000; i++ {
		stored := fmt.Sprintf("%d-more-equal", i)
		label, err = db1.GetLabel(stored)
		assert.Nil(t, err)
		readback, err := db1.LoadString(label)
		assert.Nil(t, err)
		assert.Equal(t, readback, stored)
	}

	db2, err := OpenLabels(filepath.Join(tempdir, "test"), options)
	label2, err := db2.GetLabel("8756-more-equal")
	assert.Nil(t, err)
	label1, err := db1.GetLabel("8756-more-equal")
	assert.Equal(t, label1, label2)
}
