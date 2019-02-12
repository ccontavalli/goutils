package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ccontavalli/goutils/config"
	"github.com/jordan-wright/email"
	"gopkg.in/mailgun/mailgun-go.v1"
	"io"
	"log"
	"net/smtp"
	"net/textproto"
	"os"
	"os/exec"
)

// Expands a template.
//
// Implementations are passed the name of a template to expand, a struct or map
// with variables used for expansion, and return the expanded text in the
// passed buffer or an error.
//
// A few implementations can be found in:
//   github.com/ccontavalli/goutils/templates.
type TemplateExpander interface {
	Expand(string, interface{}, io.Writer) error
}

// Takes care of physically shipping the email.
//
// Implementations are expected to provide a Send method, that will deliver
// the email, or return error.
type MailTransport interface {
	Send(*email.Email) error
}

// A transport able to send emails via Smtp.
type SmtpTransport struct {
	Server             string
	Port               int
	Username, Password string
}

// Sends an email via SMTP.
//
// To use it:
//
//     transport := SmtpTransport{
//       Server: "smtp.gmail.com",
//       Port: 587,
//       Username: "foo",
//       Password: "bar",
//     }
//     transport.Send(email)
//
func (sc *SmtpTransport) Send(mail *email.Email) error {
	mail.Headers = textproto.MIMEHeader{}
	return mail.Send(fmt.Sprintf("%s:%d", sc.Server, sc.Port), smtp.PlainAuth("", sc.Username, sc.Password, sc.Server))
}

// A transport able to send emails via a local command.
type PipeTransport struct {
	Command string
}

func (pt *PipeTransport) Send(mail *email.Email) error {
	mail.Headers = textproto.MIMEHeader{}
	command := pt.Command
	if command == "" {
		command = "/usr/sbin/sendmail -t"
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	bytes, err := mail.Bytes()
	if err != nil {
		return err
	}

	written, err := pipe.Write(bytes)
	if err != nil {
		return err
	}

	if written != len(bytes) {
		return fmt.Errorf("could not write all bytes - %d vs %d", written, len(bytes))
	}

	err1 := pipe.Close()
	err2 := cmd.Wait()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

// A transport able to send emails using Mailgun.
type MailgunTransport struct {
	Domain, ApiKey, PublicApiKey string
}

// Sends an email via Mailgun API.
//
// To use it:
//
//     transport := MailgunTransport{
//       Domain: "mydomain.com"
//       ApiKey: "myapikey",
//       PublicApiKey: "sure",
//     }
//     transport.Send(email)
//
func (mg *MailgunTransport) Send(mail *email.Email) error {
	transport := mailgun.NewMailgun(mg.Domain, mg.ApiKey, mg.PublicApiKey)
	message := mailgun.NewMessage(
		mail.From,
		mail.Subject,
		string(mail.Text),
		mail.To...)

	if len(mail.HTML) > 0 {
		message.SetHtml(string(mail.HTML))
	}

	resp, id, err := transport.Send(message)
	if err != nil {
		return err
	}
	log.Printf("SENT MAIL TO %s - ID: %s Resp: %s\n", mail.To, id, resp)
	return nil
}

// An optional configuration for a MailSender.
type MailSenderConfig struct {
	Smtp    *SmtpTransport
	Mailgun *MailgunTransport
	Pipe    *PipeTransport
}

// A MailSender - an object able to read templates and send emails using any
// transport.
type MailSender struct {
	transport MailTransport
	renderer  TemplateExpander
}

// Creates a MailSender by reading a configuration file.
//
// unmarshal is a function able to turn a []byte into a struct, such as json.Unmarshal.
// renderer is an object able to take the name of a template and well, render it.
// filename is the name of a config file in a format that the unmarshal function can process.
func NewMailSenderFromConfigFile(filename string, unmarshal config.UnmarshalFunction, renderer TemplateExpander) (*MailSender, error) {
	var myconfig MailSenderConfig
	err := config.ReadMarshaledConfigFromFile(filename, unmarshal, &myconfig)
	if err != nil {
		return nil, err
	}

	return NewMailSenderFromConfig(myconfig, renderer)
}

// Creates a MailSender from a provided config struct.
func NewMailSenderFromConfig(config MailSenderConfig, renderer TemplateExpander) (*MailSender, error) {

	transport := MailTransport(nil)
	found := 0
	if config.Smtp != nil {
		found += 1
		transport = config.Smtp
	}
	if config.Mailgun != nil {
		found += 1
		transport = config.Mailgun
	}
	if config.Pipe != nil {
		found += 1
		transport = config.Pipe
	}

	if found == 0 {
		return nil, fmt.Errorf("Must pick at least one transport")
	}
	if found > 1 {
		return nil, fmt.Errorf("Can only use one transport. Pick either SMTP, Mailgun or Pipe - %v - %v", found, config)
	}

	return NewMailSender(transport, renderer), nil
}

// Creates a MailSender manually, by specifying a transport to use, and a template expander.
func NewMailSender(transport MailTransport, renderer TemplateExpander) *MailSender {
	return &MailSender{transport, renderer}
}

func (sender *MailSender) Get(template string, data interface{}, to ...string) (*email.Email, error) {
	var headBuffer bytes.Buffer
	err := sender.renderer.Expand(template+"_head", data, &headBuffer)
	if err != nil {
		return nil, err
	}

	mail := &email.Email{}
	mail.To = to

	err = json.Unmarshal(headBuffer.Bytes(), &mail)
	if err != nil {
		return nil, err
	}
	var textBuffer bytes.Buffer
	textErr := sender.renderer.Expand(template+"_text", data, &textBuffer)
	var htmlBuffer bytes.Buffer
	htmlErr := sender.renderer.Expand(template+"_html", data, &htmlBuffer)

	if htmlErr != nil && textErr != nil {
		return nil, fmt.Errorf("Could not find neither %s_html nor %s_text", template, template)
	}

	mail.Text = textBuffer.Bytes()
	mail.HTML = htmlBuffer.Bytes()
	return mail, nil
}

func (sender *MailSender) Send(template string, data interface{}, to ...string) error {
	mail, err := sender.Get(template, data, to...)
	if err != nil {
		return err
	}

	return sender.transport.Send(mail)
}

func (sender *MailSender) SendEmail(mail *email.Email) error {
	return sender.transport.Send(mail)
}
