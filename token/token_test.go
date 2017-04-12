package token

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
)

func TestReadOrCreateKey(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "drpack-test")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	token := path.Join(dir, "test.token")
	fmt.Printf("Using path: %s\n", token)

	settings1 := DefaultTokenSettings()
	assert.Equal([]byte{}, settings1.key)

	err = settings1.ReadOrCreateKey(token)
	assert.Nil(err)
	assert.NotEqual([]byte{}, settings1.key)
	assert.Equal(32, len(settings1.key))

	settings2 := DefaultTokenSettings()
	err = settings2.ReadOrCreateKey(token)
	assert.Nil(err)
	assert.Equal(settings2.key, settings1.key)
}

func TestSimpleEncodeDecode(t *testing.T) {
	assert := assert.New(t)

	generator, err := NewTokenGenerator(DefaultTokenSettings())
	assert.NotNil(generator)
	assert.Nil(err)

	token, err := generator.Generate("foobar", nil)
	assert.NotNil(token)
	assert.Nil(err)

	username, timestamp, err := generator.IsValid(token, nil)
	assert.Nil(err)
	assert.Equal("foobar", username)

	fmt.Printf("Test: %s, %v, %v\n", token, timestamp, err)
}

func TestExtraSignedData(t *testing.T) {
	assert := assert.New(t)

	generator, err := NewTokenGenerator(DefaultTokenSettings())
	assert.NotNil(generator)
	assert.Nil(err)

	token, err := generator.Generate("foobar", []string{"valid data", "that is also signed"})
	assert.NotNil(token)
	assert.Nil(err)

	username, timestamp, err := generator.IsValid(token, nil)
	assert.NotNil(err)
	assert.Equal("", username)
	assert.Equal(int64(0), timestamp)

	username, timestamp, err = generator.IsValid(token, []string{"vald data", "that is also signed"})
	assert.NotNil(err)
	assert.Equal("", username)
	assert.Equal(int64(0), timestamp)

	username, timestamp, err = generator.IsValid(token, []string{"valid data", "that is also signed"})
	assert.Nil(err)
	assert.Equal("foobar", username)
}

func TestInvalidtokens(t *testing.T) {
	assert := assert.New(t)

	generator, err := NewTokenGenerator(DefaultTokenSettings())
	assert.NotNil(generator)
	assert.Nil(err)

	user, validity, err := generator.IsValid("", nil)
	assert.NotNil(err)
	assert.Equal("", user)
	assert.Equal(int64(0), validity)
	fmt.Printf("error: %s\n", err)

	user, validity, err = generator.IsValid("0:", nil)
	assert.NotNil(err)
	assert.Equal("", user)
	assert.Equal(int64(0), validity)
	fmt.Printf("error: %s\n", err)

	user, validity, err = generator.IsValid("0:,", nil)
	assert.NotNil(err)
	assert.Equal("", user)
	assert.Equal(int64(0), validity)
	fmt.Printf("error: %s\n", err)

	// Generate a valid token now.
	token, err := generator.Generate("foobar", nil)
	assert.NotNil(token)
	assert.Nil(err)

	// What if this is too short by one byte?
	user, validity, err = generator.IsValid(token[:len(token)-1], nil)
	assert.NotNil(err)
	assert.Equal("", user)
	assert.Equal(int64(0), validity)
	fmt.Printf("error: %s\n", err)
}

func TestFuzzyToken(t *testing.T) {
	assert := assert.New(t)

	generator, err := NewTokenGenerator(DefaultTokenSettings())
	assert.NotNil(generator)
	assert.Nil(err)

	// Generate a valid token now.
	token, err := generator.Generate("foobar,fuffa", nil)
	fmt.Printf("TOKEN NOW: %s\n", token)
	assert.NotNil(token)
	assert.Nil(err)

	// Check it is valid before  moving forward.
	username, timestamp, err := generator.IsValid(token, nil)
	assert.Nil(err)
	assert.Equal("foobar,fuffa", username)
	assert.NotEqual(int64(0), timestamp)

	// What if token has a random permutation?
	offset := len("0:Zm9vYmFyLGZ1ZmZh,604800,")
	replacements := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	for j := 0; j < 10000; j++ {
		for i := offset; i < len(token); i++ {

			modified := []byte(token)
			var replacement byte
			for true {
				replacement = replacements[rand.Intn(len(replacements))]
				if replacement != modified[i] {
					break
				}
			}

			modified[i] = replacement

			user, validity, err := generator.IsValid(string(modified), nil)
			assert.NotNil(err)
			assert.Equal("", user)
			assert.Equal(int64(0), validity)
		}
	}
}

func TestFuzzyUsername(t *testing.T) {
	assert := assert.New(t)

	generator, err := NewTokenGenerator(DefaultTokenSettings())
	assert.NotNil(generator)
	assert.Nil(err)

	// Generate a valid token now.
	token, err := generator.Generate("foobar", nil)
	assert.NotNil(token)
	assert.Nil(err)

	// Check it is valid before  moving forward.
	username, timestamp, err := generator.IsValid(token, nil)
	assert.Nil(err)
	assert.Equal("foobar", username)
	assert.NotEqual(int64(0), timestamp)

	// What if username has a random permutation?
	offset := len("0:")
	replacements := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	for j := 0; j < 10000; j++ {
		for i := 0; i < len("foobar"); i++ {
			modified := []byte(token)
			var replacement byte
			for true {
				replacement = replacements[rand.Intn(len(replacements))]
				if replacement != modified[offset+i] {
					break
				}
			}

			modified[offset+i] = replacement

			user, validity, err := generator.IsValid(string(modified), nil)
			assert.NotNil(err)
			assert.Equal("", user)
			assert.Equal(int64(0), validity)
		}
	}
}
