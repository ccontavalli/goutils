package misc

import (
	"github.com/stretchr/testify/assert"
	"testing"
	//    "fmt"
)

func TestHasString(t *testing.T) {
	assert := assert.New(t)
	input := []string{"b", "d", "f"}
	assert.False(SortedHasString(input, "a"))
	assert.True(SortedHasString(input, "b"))
	assert.False(SortedHasString(input, "c"))
	assert.True(SortedHasString(input, "d"))
	assert.False(SortedHasString(input, "e"))
	assert.True(SortedHasString(input, "f"))
	assert.False(SortedHasString(input, "g"))
}

func TestDedup(t *testing.T) {
	assert := assert.New(t)
	assert.Equal([]string{}, SortedDedup([]string{}))
	assert.Equal([]string{"a"}, SortedDedup([]string{"a"}))
	assert.Equal([]string{"a"}, SortedDedup([]string{"a", "a"}))
	assert.Equal([]string{"a"}, SortedDedup([]string{"a", "a", "a"}))
	assert.Equal([]string{"a", "b"}, SortedDedup([]string{"a", "b"}))
	assert.Equal([]string{"a", "b"}, SortedDedup([]string{"a", "a", "a", "b"}))
	assert.Equal([]string{"a", "b"}, SortedDedup([]string{"a", "b", "b"}))
	assert.Equal([]string{"a", "b"}, SortedDedup([]string{"a", "b", "b", "b"}))
	assert.Equal([]string{"a", "b", "c"}, SortedDedup([]string{"a", "b", "b", "c"}))
	assert.Equal([]string{"a", "b", "c"}, SortedDedup([]string{"a", "b", "b", "c", "c", "c"}))
	assert.Equal([]string{"a", "b", "c"}, SortedDedup([]string{"a", "a", "b", "b", "c", "c", "c"}))
}
