package domain

import "context"

type EmailJob struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(ctx context.Context, job EmailJob) error
}
