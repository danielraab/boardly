// Package mail sends email: over SMTP, or into the log when no SMTP server is configured.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	gomail "github.com/wneessen/go-mail"

	"boardly/internal/config"
)

// Message is a plain-text email with an HTML alternative.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender delivers messages.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// New returns an SMTP sender, or a log sender when no SMTP host is configured.
// In that case it logs a warning that mail delivery is disabled.
func New(cfg config.SMTP) Sender {
	if cfg.Host == "" {
		slog.Warn("mail delivery is disabled (BOARDLY_SMTP_HOST is not set); sign-in links are written to the log")
		return LogSender{}
	}
	return &SMTPSender{cfg: cfg}
}

// LogSender writes messages to the log instead of sending them. Meant for local development.
type LogSender struct{}

// Send logs the message's recipient, subject and plain-text body.
func (LogSender) Send(ctx context.Context, m Message) error {
	slog.InfoContext(ctx, "mail not sent (delivery disabled)", "to", m.To, "subject", m.Subject, "body", m.Text)
	return nil
}

// sendTimeout bounds the whole SMTP exchange of one Send.
const sendTimeout = 10 * time.Second

// SMTPSender sends messages through an SMTP server.
type SMTPSender struct {
	cfg config.SMTP
}

// Send connects to the server, delivers m and disconnects.
func (s *SMTPSender) Send(ctx context.Context, m Message) error {
	msg := gomail.NewMsg()
	if err := msg.From(s.cfg.From); err != nil {
		return fmt.Errorf("mail sender %q: %w", s.cfg.From, err)
	}
	if err := msg.To(m.To); err != nil {
		return fmt.Errorf("mail recipient: %w", err)
	}
	msg.Subject(m.Subject)
	msg.SetDate()
	msg.SetMessageID()
	msg.SetBodyString(gomail.TypeTextPlain, m.Text)
	msg.AddAlternativeString(gomail.TypeTextHTML, m.HTML)

	opts := []gomail.Option{gomail.WithPort(s.cfg.Port), gomail.WithTimeout(sendTimeout)}
	switch s.cfg.TLS {
	case "tls":
		opts = append(opts, gomail.WithSSL())
	case "none":
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	if s.cfg.Username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(s.cfg.Username),
			gomail.WithPassword(s.cfg.Password),
		)
	}
	client, err := gomail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("mail client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("sending mail: %w", err)
	}
	return nil
}
