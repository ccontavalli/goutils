package templates

import (
	"bytes"
	"github.com/ccontavalli/goutils/misc"
	"github.com/stretchr/testify/assert"
	"testing"
	//    "fmt"
)

func TestNewStaticTemplatesEmpty(t *testing.T) {
	assert := assert.New(t)

	st, err := NewStaticTemplates([]string{}, nil, nil)
	assert.Equal(nil, err)
	assert.Equal(0, len(st.templates))
}

func TestNewStaticTemplatesSimple(t *testing.T) {
	template := []byte(`this is a test template`)

	assert := assert.New(t)
	st, err := NewStaticTemplates([]string{"test.txt"}, nil, func(file string) ([]byte, error) {
		assert.Equal(file, "test.txt")
		return template, nil
	})

	assert.Equal(nil, err)
	assert.Equal(1, len(st.templates))
	assert.NotEqual(nil, st.templates["test"])
}

func TestNewStaticTemplatesCombined(t *testing.T) {
	contents := map[string]string{
		"test.txt":         `{{ define "nested" }}{{ end }}{{ define "start"}}start {{ template "nested" . }}{{ end }}`,
		"foo=test.txt":     `{{ define "nosted" }}{{ end }}{{ define "nested"}}nested {{ template "nosted" . }}{{ end }}`,
		"bar=foo,test.txt": `{{ define "nosted"}}mine{{ end }}`,
		"bus=bar.txt":      `{{ define "start"}}{{ template "nosted" . }} munch{{ end }}`,
	}

	assert := assert.New(t)
	st, err := NewStaticTemplates(misc.StringKeysOrPanic(contents), nil, func(file string) ([]byte, error) {
		content, ok := contents[file]
		assert.True(ok)

		return []byte(content), nil
	})

	assert.Equal(nil, err)
	assert.Equal(4, len(st.templates))
	assert.NotEqual(nil, st.templates["test"])
	assert.NotEqual(nil, st.templates["foo"])
	assert.NotEqual(nil, st.templates["bar"])
	assert.NotEqual(nil, st.templates["bus"])

	buffer := bytes.Buffer{}
	assert.Equal(nil, st.Expand("test", struct{}{}, &buffer))
	assert.Equal("start ", buffer.String())

	buffer.Reset()
	assert.Equal(nil, st.Expand("foo", struct{}{}, &buffer))
	assert.Equal("start nested ", buffer.String())

	buffer.Reset()
	assert.Equal(nil, st.Expand("bar", struct{}{}, &buffer))
	assert.Equal("start nested mine", buffer.String())

	buffer.Reset()
	assert.Equal(nil, st.Expand("bus", struct{}{}, &buffer))
	assert.Equal("mine munch", buffer.String())
}

func TestNewStaticTemplatesFromDir(t *testing.T) {
	assert := assert.New(t)

	st, err := NewStaticTemplatesFromDir("non-existing-dir", nil)
	assert.Error(err)
	assert.Nil(st)

	st, err = NewStaticTemplatesFromDir("test", nil)
	assert.Nil(err)

	assert.Equal(3, len(st.templates))
	assert.NotEqual(nil, st.templates["template0"])
	assert.NotEqual(nil, st.templates["template1"])
	assert.NotEqual(nil, st.templates["template2"])
}

func TestNewStaticTemplatesFromMap(t *testing.T) {
	assert := assert.New(t)
	templates := map[string][]byte{
		"template0": []byte("hello"),
		"template1": []byte("world"),
		"template2": []byte("will"),
	}

	st, err := NewStaticTemplatesFromMap(templates, nil)
	assert.Nil(err)
	assert.NotNil(st)

	assert.Equal(3, len(st.templates))
	assert.NotEqual(nil, st.templates["template0"])
	assert.NotEqual(nil, st.templates["template1"])
	assert.NotEqual(nil, st.templates["template2"])
}
