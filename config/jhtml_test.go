package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplitJhtml(t *testing.T) {
	assert := assert.New(t)

	h, b := SplitJhtml([]byte(""))
	assert.EqualValues("", h)
	assert.EqualValues("", b)

	h, b = SplitJhtml([]byte("foo"))
	assert.EqualValues("foo", h)
	assert.EqualValues("", b)

	h, b = SplitJhtml([]byte("foo bar baz---\n---foo bar---"))
	assert.EqualValues("foo bar baz---\n---foo bar---", h)
	assert.EqualValues("", b)

	h, b = SplitJhtml([]byte("\n---\nfoo"))
	assert.EqualValues("", h)
	assert.EqualValues("foo", b)

	h, b = SplitJhtml([]byte("baz\n---\nfoo"))
	assert.EqualValues("baz", h)
	assert.EqualValues("foo", b)
}
