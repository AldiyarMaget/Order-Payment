package infrastructure

import (
	"context"
	"fmt"
	"net/smtp"

	"order/notification/internal/domain"
)

type SMTPSender struct {
	host     string
	port     string
	user     string
	password string
}

func NewSMTPSender(host, port, user, password string) domain.EmailSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		user:     user,
		password: password,
	}
}

func (s *SMTPSender) Send(ctx context.Context, job domain.EmailJob) error {
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	to := []string{job.To}
	msg := []byte("To: " + job.To + "\r\n" +
		"Subject: " + job.Subject + "\r\n" +
		"\r\n" +
		job.Body + "\r\n")

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	err := smtp.SendMail(addr, auth, s.user, to, msg)
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}
