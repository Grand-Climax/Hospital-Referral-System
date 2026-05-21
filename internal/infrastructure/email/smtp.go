package email

import (
	"context"
	"fmt"
	"log"
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

type devEmailClient struct{}

func (c *devEmailClient) Send(ctx context.Context, to, subject, body string) error {
	log.Printf("[DEV EMAIL] Sending email to %s | Subject: %s | Body: %s", to, subject, body)
	return nil
}

func NewSMTPClient() Client {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	from := os.Getenv("SMTP_FROM")

	if host == "" || port == "" || from == "" {
		log.Println("Warning: SMTP is not fully configured (SMTP_HOST, SMTP_PORT, SMTP_FROM). Falling back to log-based devEmailClient.")
		return &devEmailClient{}
	}

	return &smtpClient{
		host:     host,
		port:     port,
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     from,
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

