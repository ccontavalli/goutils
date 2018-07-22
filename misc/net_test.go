package misc

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	//    "fmt"
)

func TestLooselyGetHost(t *testing.T) {
	assert := assert.New(t)
	testcases := []struct {
		input, expected string
	}{
		{"", ""},
		{":80", ""},
		{"[::]", "::"},
		{"0", "0"},
		{"0:80", "0"},
		{"fuffa:80", "fuffa"},
		{"[::1]:443", "::1"},
		{"[::1:443", "::1:443"},
		{"127.0.0.1:80:80", "127.0.0.1"},
		{"[::1]]]:443", "::1"},
	}
	for _, testcase := range testcases {
		assert.Equal(testcase.expected, LooselyGetHost(testcase.input))
	}
}

func TestSplitHostPortValid(t *testing.T) {
	assert := assert.New(t)
	inputs := []struct {
		data, host, port string
		compatible       bool
	}{
		{"fuffa", "fuffa", "", false},
		{"[::]", "::", "", false},
		{"[::]:80", "::", "80", true},
		{"[::]:http", "::", "http", true},
		{"0:http", "0", "http", true},
		{":http", "", "http", true},
		{"0.0.0.0:8", "0.0.0.0", "8", true},
		{"0.0.0.0:", "0.0.0.0", "", true},
		{":", "", "", true},
		{"", "", "", false},
	}

	for _, input := range inputs {
		host, port, err := SplitHostPort(input.data)
		assert.Nil(err)
		assert.Equal(input.host, host)
		assert.Equal(input.port, port)

		fhost, fport, err := net.SplitHostPort(input.data)
		if !input.compatible {
			assert.NotNil(err)
			continue
		}
		assert.Nil(err)
		assert.Equal(fhost, host)
		assert.Equal(fport, fport)
	}
}

func TestSplitHostPortInvalid(t *testing.T) {
	assert := assert.New(t)
	inputs := []string{
		"::",
		"[]::",
		"[::]::",
		"[::1]]:80",
		"[::1:80",
		"::1]:80",
		"0.0.0.0::",
		"0::",
		"0:[",
		"0:]",
		"]:80",
		"0[:80",
		"0[]:80",
	}

	for _, input := range inputs {
		_, _, err := SplitHostPort(input)
		assert.NotNil(err, "for input: %s", input)
	}
}
