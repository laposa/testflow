// Package mail sends the app's emails (spec §8.10; legacy: Laravel
// notifications over SMTP). Two mailers implement Mailer: Log, which writes
// the message to slog (the default for development), and SMTP.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// Message is a plain-text email to one recipient.
type Message struct {
	To      mail.Address
	Subject string
	Text    string
}

// Mailer sends messages.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// Log writes each message to the logger instead of sending it.
type Log struct{ Logger *slog.Logger }

// Send logs m.
func (l Log) Send(ctx context.Context, m Message) error {
	logger := l.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "email not sent (MAIL_MAILER=log)",
		"email.to", m.To.Address, "email.subject", m.Subject, "email.body", m.Text)
	return nil
}

// SMTP sends messages through an SMTP server with authentication. Port
// 465 uses implicit TLS; any other port must offer STARTTLS, so the
// password never crosses the network in clear text.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     mail.Address
	Timeout  time.Duration // per message, default DefaultTimeout
	// TLSConfig is for tests (a server with its own certificate); nil
	// verifies the server's certificate for Host.
	TLSConfig *tls.Config
}

// DefaultTimeout bounds one SMTP conversation, connecting included.
const DefaultTimeout = 15 * time.Second

// Send delivers m.
func (s *SMTP) Send(ctx context.Context, m Message) error {
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	data, err := s.compose(m)
	if err != nil {
		return err
	}
	if err := s.send(ctx, m.To.Address, data); err != nil {
		return fmt.Errorf("smtp %s: %w", s.Host, err)
	}
	return nil
}

func (s *SMTP) send(ctx context.Context, to string, data []byte) error {
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	tlsConfig := s.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}
	}
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	if s.Port == 465 {
		conn = tls.Client(conn, tlsConfig)
	}
	// The deadline covers the whole conversation; net/smtp has no context.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	// net/smtp would greet as "localhost", which strict servers reject.
	if err := c.Hello(helloName()); err != nil {
		return err
	}
	if s.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("the server does not offer STARTTLS")
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if s.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(s.From.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// helloName is this machine's name for EHLO, or localhost.
func helloName() string {
	if h, err := os.Hostname(); err == nil && h != "" && !strings.ContainsAny(h, " \r\n") {
		return h
	}
	return "localhost"
}

// compose builds the message: headers (encoded, so names and subjects
// cannot inject headers) and a quoted-printable UTF-8 body.
func (s *SMTP) compose(m Message) ([]byte, error) {
	if m.To.Address == "" {
		return nil, errors.New("mail: no recipient")
	}
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	_, domain, _ := strings.Cut(s.From.Address, "@")
	var b bytes.Buffer
	header := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	header("From", s.From.String())
	header("To", m.To.String())
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", "<"+hex.EncodeToString(id)+"@"+domain+">")
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=utf-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	// The writer turns line breaks into CRLF.
	if _, err := qp.Write([]byte(m.Text)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
