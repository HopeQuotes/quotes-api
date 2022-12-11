package smtp

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"javlonrahimov/quotes-api/assets"
	"javlonrahimov/quotes-api/internal/funcs"

	"github.com/go-mail/mail/v2"
)

type Mailer interface {
	Send(recipient, plainOTP string, data any, patterns ...string) error
}

type EmailSender struct {
	dialer *mail.Dialer
	from   string
}

type TelegramSender struct {
	botToken  string
	channelID string
}

func NewEmailSender(host string, port int, username, password, from string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return &EmailSender{
		dialer: dialer,
		from:   from,
	}
}

func NewTelegramSender(botToken, channelID string) Mailer {
	return &TelegramSender{
		botToken:  botToken,
		channelID: channelID,
	}
}

func (m *EmailSender) Send(recipient, plainOTP string, data any, patterns ...string) error {
	for i := range patterns {
		patterns[i] = "emails/" + patterns[i]
	}

	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.from)

	ts, err := template.New("").Funcs(funcs.HelperFuncs).ParseFS(assets.EmbeddedFiles, patterns...)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = ts.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	msg.SetHeader("Subject", subject.String())

	plainBody := new(bytes.Buffer)
	err = ts.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	msg.SetBody("text/plain", plainBody.String())

	if ts.Lookup("htmlBody") != nil {
		htmlBody := new(bytes.Buffer)
		err = ts.ExecuteTemplate(htmlBody, "htmlBody", data)
		if err != nil {
			return err
		}

		msg.AddAlternative("text/html", htmlBody.String())
	}

	for i := 1; i <= 3; i++ {
		err = m.dialer.DialAndSend(msg)

		if nil == err {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return err
}

func (m *TelegramSender) Send(recipient, plainOTP string, data any, patterns ...string) error {
	str := fmt.Sprintf(`{"chat_id": "%s", "text": "<code>%s</code> for %s", "parse_mode": "html"}`, m.channelID, plainOTP, recipient)
	jsonBody := []byte(str)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.botToken), bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return nil
}
