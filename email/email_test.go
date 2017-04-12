package email

import (
	"github.com/ccontavalli/goutils/templates"
	"github.com/jordan-wright/email"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
	//    "fmt"
)

func TestNewMailSenderFromConfig(t *testing.T) {
	assert := assert.New(t)

	ms, err := NewMailSenderFromConfig(MailSenderConfig{}, nil)
	assert.Nil(ms)
	assert.Error(err)

	ms, err = NewMailSenderFromConfig(MailSenderConfig{&SmtpTransport{}, &MailgunTransport{}, nil}, nil)
	assert.Nil(ms)
	assert.Error(err)

	ms, err = NewMailSenderFromConfig(MailSenderConfig{&SmtpTransport{}, nil, nil}, nil)
	assert.Nil(err)
	assert.NotNil(ms.transport)

	ms, err = NewMailSenderFromConfig(MailSenderConfig{nil, &MailgunTransport{}, nil}, nil)
	assert.Nil(err)
	assert.NotNil(ms.transport)
}

func TestNewMailSenderFromConfigFile(t *testing.T) {
	assert := assert.New(t)

	ms, err := NewMailSenderFromConfigFile("test/doesnotexist.yaml", yaml.Unmarshal, nil)
	assert.NotNil(err)
	assert.Nil(ms)

	ms, err = NewMailSenderFromConfigFile("test/config.yaml", yaml.Unmarshal, nil)
	assert.Nil(err)
	assert.NotNil(ms)

	smtp, ok := ms.transport.(*SmtpTransport)
	assert.True(ok)
	assert.NotNil(smtp)
	assert.Equal(smtp.Username, "foo")
	assert.Equal(smtp.Password, "bar")
	assert.Equal(smtp.Server, "127.0.0.1")
	assert.Equal(smtp.Port, 587)
}

func TestNewMailComposition(t *testing.T) {
	assert := assert.New(t)

	mytemplates := map[string][]byte{
		"test_head": []byte(`{{ define "start" }}{ "From": "foo <validation@test.org>", "Subject": "Yo!" }{{ end }}`),
		"test_text": []byte(`{{ define "start" }}Welcome, {{ .Darling }}{{ end }}`),
		"test_html": []byte(`{{ define "start" }}Ahoh, {{ .Darling }}{{ end }}`),
	}

	st, err := templates.NewStaticTemplatesFromMap(mytemplates, nil)
	assert.Nil(err)
	assert.NotNil(st)

	ms, err := NewMailSenderFromConfigFile("test/config.yaml", yaml.Unmarshal, st)
	assert.Nil(err)
	assert.NotNil(ms)

	mail, err := ms.Get("foo", struct{ Darling string }{"Mine"}, "test.org")
	assert.Error(err)
	assert.Nil(mail)

	mail, err = ms.Get("test", struct{ Darling string }{"Mine"}, "test.org")
	assert.Nil(err)
	assert.NotNil(mail)

	expected := email.Email{}
	expected.From = "foo <validation@test.org>"
	expected.To = []string{"test.org"}
	expected.Subject = "Yo!"
	expected.Text = []byte(`Welcome, Mine`)
	expected.HTML = []byte(`Ahoh, Mine`)

	assert.Equal(expected, *mail)
}
