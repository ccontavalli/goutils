package token

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"
)

func uint64ToByteArray(array []byte, number uint64) {
	for i := 0; i < 8; i++ {
		array[i] = byte((number >> uint(56-i*8)) & 0xff)
	}
}

type TokenGenerator struct {
	settings TokenSettings
	cipher   cipher.AEAD
}

type TokenSettings struct {
	validity_seconds uint32

	// Key to use to generate the tokens. If no key is specified,
	// an ephemeral one will be generated for you.
	key []byte
	// If no key is specified, this parameter determines how long
	// of a key to generate. If not specified (eg, == 0), 256 bits
	// are used as a default.
	key_size_bits uint32
}

func DefaultTokenSettings() TokenSettings {
	return TokenSettings{3600 * 24 * 7, []byte{}, 0}
}
func (settings *TokenSettings) GetKeyLengthInBytes() uint32 {
	if settings.key_size_bits > 0 {
		return settings.key_size_bits / 8
	}
	return 256 / 8
}

// For internal use only: reads a key from the settings, or creates
// a new key.
func (settings *TokenSettings) getOrCreateKey() ([]byte, error) {
	var key []byte
	var err error
	if len(settings.key) <= 0 {
		err = settings.CreateKey()
		if err != nil {
			return nil, err
		}
	}

	key = settings.key
	return key, nil
}

// Creates a new random key and stores it in settings, or return error.
func (settings *TokenSettings) CreateKey() error {
	size := settings.GetKeyLengthInBytes()
	key := make([]byte, size)

	n, err := rand.Read(key)
	if err != nil {
		return err
	}
	if n != int(size) {
		return fmt.Errorf("PRNG could not provide %d bytes of key", size)
	}
	settings.key = key
	return nil
}

// Reads a key from a file, or creates a new one and stores it in a file.
// Returns error if it can't succeed in generating or storing a new key.
func (settings *TokenSettings) ReadOrCreateKey(path string) error {
	key, err := ioutil.ReadFile(path)
	if err != nil || len(key) <= 0 || len(key)%8 != 0 {
		err = settings.CreateKey()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(path, settings.key, 0600)
	} else {
		settings.key = key
	}
	return err
}

func NewTokenGenerator(settings TokenSettings) (*TokenGenerator, error) {
	key, err := settings.getOrCreateKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &TokenGenerator{settings, aead}, nil
}

func generateSeal(username []byte, tosign []string) []byte {
	sealsize := uint64(len(username) + 1)
	for _, value := range tosign {
		sealsize += uint64(len(value)) + 1
	}

	toseal := make([]byte, 0, sealsize)
	toseal = append(toseal, []byte(username)...)
	toseal = append(toseal, byte(0))
	for _, value := range tosign {
		toseal = append(toseal, []byte(value)...)
		toseal = append(toseal, byte(0))
	}

	return toseal
}

func (t *TokenGenerator) Generate(data string, tosign []string) (string, error) {
	nonce := make([]byte, t.cipher.NonceSize())
	n, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	if n != t.cipher.NonceSize() {
		return "", fmt.Errorf("PRNG could not provide %d bytes of nonce", t.cipher.NonceSize())
	}

	// output has:
	//   username + "," + approx_time_left_seconds + ",0:" + mime64encode(nonce + ciphertext)
	//
	// plaintext has:
	//   timestamp
	//
	// output should be large enough to hold:
	//    timestamp + t.cipher.Overhead()

	encoder := base64.URLEncoding

	// 20 is len(max<uint64>()), 2* is to take into account mime64 encoding while adding some slack.
	token := make([]byte, 0, 2*(len(data)+20)+len(",,0:")+2*(20+t.cipher.NonceSize()+t.cipher.Overhead()))
	token = append(token, "0:"...)
	offset := len(token)
	token = token[:offset+encoder.EncodedLen(len(data))]
	encoder.Encode(token[offset:], []byte(data))
	token = append(token, ',')
	token = strconv.AppendUint(token, (uint64)(t.settings.validity_seconds), 10)
	token = append(token, ","...)

	now := time.Now().Unix()
	//fmt.Printf("timestamp %d\n", now)
	plaintext := make([]byte, 8)
	uint64ToByteArray(plaintext, uint64(now))

	// Create data to seal.
	toseal := generateSeal([]byte(data), tosign)
	ciphertext := t.cipher.Seal(nonce, nonce, plaintext, toseal)
	//fmt.Printf("nonce %x\n", nonce)
	//fmt.Printf("ciphertext %x\n", ciphertext[t.cipher.NonceSize():])

	offset = len(token)
	token = token[:offset+encoder.EncodedLen(len(ciphertext))]
	encoder.Encode(token[offset:], ciphertext)

	return string(token), nil
}

// Returns username if validaiton succeeds and no error.
// Returns at least error in all other cases.
func (t *TokenGenerator) IsValid(token string, tosign []string) (string, int64, error) {
	btoken := []byte(token)
	if !bytes.HasPrefix(btoken, []byte("0:")) {
		return "", 0, fmt.Errorf("Unknown token format, does not start with 0:")
	}
	stripped := bytes.TrimPrefix(btoken, []byte("0:"))
	//fmt.Printf("token %s\n", stripped)

	ciphertext_offset := bytes.LastIndex(stripped, []byte(","))
	if ciphertext_offset < 0 {
		return "", 0, fmt.Errorf("Invalid token: no , found for ciphertext")
	}

	data_offset := bytes.Index(stripped, []byte(","))
	if data_offset < 0 {
		return "", 0, fmt.Errorf("Invalid token: no , found for data")
	}

	decoder := base64.URLEncoding
	mime64_data := stripped[:data_offset]
	//fmt.Printf("data %s\n", data)
	data := make([]byte, decoder.DecodedLen(len(mime64_data)))
	data_len, err := decoder.Decode(data, mime64_data)
	data = data[:data_len]

	mime64_ciphertext := stripped[ciphertext_offset+1:]
	//fmt.Printf("mime64 %s\n", mime64)

	ciphertext := make([]byte, decoder.DecodedLen(len(mime64_ciphertext)))
	ciphertext_len, err := decoder.Decode(ciphertext, mime64_ciphertext)
	if err != nil {
		return "", 0, err
	}

	if ciphertext_len < t.cipher.NonceSize() {
		return "", 0, fmt.Errorf("ciphertext too short, canot hold nonce.")
	}

	nonce := ciphertext[:t.cipher.NonceSize()]
	ciphertext = ciphertext[t.cipher.NonceSize():ciphertext_len]
	//fmt.Printf("nonce %x\n", nonce)
	//fmt.Printf("ciphertext %x\n", ciphertext)

	toseal := generateSeal(data, tosign)
	plaintext, err := t.cipher.Open(nil, nonce, ciphertext, toseal)
	if err != nil {
		return "", 0, err
	}

	if len(plaintext) < 8 {
		return "", 0, fmt.Errorf("plaintext too short, can't contain valid token")
	}

	// Extract timestamp from the plaintext received.
	timestamp := int64(0)
	for i := 0; i < 8; i++ {
		timestamp = timestamp<<8 | int64(plaintext[i])
	}

	now := time.Now().Unix()
	if timestamp <= 0 || now > timestamp+int64(t.settings.validity_seconds) {
		return "", 0, fmt.Errorf("Token expired %d seconds ago", now-timestamp)
	}

	return string(data), int64(timestamp), nil
}

func (t *TokenGenerator) Extend(token string) (string, error) {
	return "", nil
}
