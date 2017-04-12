// Various random commodity functions used across projects.
//
// Mostly help prevent retyping the same code over and over again.
package config

import (
	"encoding/json"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// This is a function able to parse a buffer into any kind of struct, returning
// an error if the input format is somehow invalid.
//
// Example of UnmarshalFunction are the standard json.Unmarshal or yaml.Unmarshal.
type UnmarshalFunction func([]byte, interface{}) error

// Reads a config from a file and unmarshals it.
//
// The parameter unmarshal is a function like json.Unmarshal or yaml.Unmarshal.
func ReadMarshaledConfigFromFile(filename string, unmarshal UnmarshalFunction, result interface{}) error {
	config, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return unmarshal(config, result)
}

// Reads a file from disk and parses it as yaml.
func ReadYamlConfigFromFile(filename string, result interface{}) error {
	return ReadMarshaledConfigFromFile(filename, yaml.Unmarshal, result)
}

// Reads a file from disk and parses it as json.
func ReadJsonConfigFromFile(filename string, result interface{}) error {
	return ReadMarshaledConfigFromFile(filename, json.Unmarshal, result)
}
