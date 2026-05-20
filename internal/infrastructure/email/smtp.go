package email

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
)

type Client interface {
	Send(ctx context.Context, to, subject, body string) error
}

type smtpClient struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPClient() Client {
	return &smtpClient{
		host:     os.Getenv("SMTP_HOST"),
		port:     os.Getenv("SMTP_PORT"),
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("SMTP_FROM"),
	}
}

func (c *smtpClient) Send(_ context.Context, to, subject, body string) error {
	if c.host == "" || c.port == "" || c.from == "" {
		return fmt.Errorf("email provider not configured")
	}

	addr := c.host + ":" + c.port
	msg := []byte(
		"From: " + c.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
			body + "\r\n",
	)

	var auth smtp.Auth
	if c.username != "" && c.password != "" {
		auth = smtp.PlainAuth("", c.username, c.password, c.host)
	}

	return smtp.SendMail(addr, auth, c.from, []string{to}, msg)
}
