// Various random commodity functions used across projects.
//
// Mostly help prevent retyping the same code over and over again.
package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

var JSeparator = []byte{'\n', '-', '-', '-', '\n'}

func SplitJhtml(content []byte) ([]byte, []byte) {
	separator := bytes.Index(content, JSeparator)
	if separator < 0 {
		return content, content[0:0]
	}
	return content[0:separator], content[separator+len(JSeparator):]
}

func ParseJhtmlBuffer(buffer []byte, jheader interface{}) ([]byte, error) {
	header, content := SplitJhtml(buffer)
	err := json.Unmarshal(header, jheader)
	return content, err
}

func ParseJhtmlFile(fpath string, jheader interface{}) ([]byte, error) {
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	return ParseJhtmlBuffer(data, jheader)
}
