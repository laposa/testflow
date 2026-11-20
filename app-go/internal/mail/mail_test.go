package mail

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"io"
	"log/slog"
	"math/big"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testCert returns a self-signed certificate for 127.0.0.1 and a client
// config that trusts it.
func testCert(t *testing.T) (tls.Certificate, *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}
}

// received is what the fake server saw.
type received struct {
	auth, from, to string
	hello          string // the EHLO name
	data           string
	tls            bool
}

// fakeServer answers one SMTP conversation. Without starttls it does not
// offer STARTTLS.
func fakeServer(t *testing.T, starttls bool) (port int, client *tls.Config, got chan received) {
	t.Helper()
	cert, client := testCert(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	got = make(chan received, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var r received
		defer func() { got <- r }()
		tp := textproto.NewConn(conn)
		_ = tp.PrintfLine("220 fake ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			cmd, arg, _ := strings.Cut(line, " ")
			switch strings.ToUpper(cmd) {
			case "EHLO":
				r.hello = arg
				if starttls && !r.tls {
					_ = tp.PrintfLine("250-fake\r\n250 STARTTLS")
				} else {
					_ = tp.PrintfLine("250-fake\r\n250 AUTH PLAIN")
				}
			case "STARTTLS":
				_ = tp.PrintfLine("220 go ahead")
				tc := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
				if err := tc.Handshake(); err != nil {
					return
				}
				conn, r.tls = tc, true
				tp = textproto.NewConn(tc)
			case "AUTH":
				_, b64, _ := strings.Cut(arg, " ")
				dec, _ := base64.StdEncoding.DecodeString(b64)
				r.auth = string(dec)
				_ = tp.PrintfLine("235 ok")
			case "MAIL":
				r.from = arg
				_ = tp.PrintfLine("250 ok")
			case "RCPT":
				r.to = arg
				_ = tp.PrintfLine("250 ok")
			case "DATA":
				_ = tp.PrintfLine("354 go")
				b, _ := tp.ReadDotBytes()
				r.data = string(b)
				_ = tp.PrintfLine("250 queued")
			case "QUIT":
				_ = tp.PrintfLine("221 bye")
				return
			default:
				_ = tp.PrintfLine("502 no")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, client, got
}

func TestSMTPSend(t *testing.T) {
	port, client, got := fakeServer(t, true)
	s := &SMTP{Host: "127.0.0.1", Port: port, Username: "user", Password: "secret",
		From: mail.Address{Name: "TestFlow", Address: "testflow@example.com"}, TLSConfig: client, Timeout: 5 * time.Second}
	err := s.Send(t.Context(), Message{
		To:      mail.Address{Name: "Ada \"QA\" Lovelace\r\nBcc: evil@example.com", Address: "ada@example.com"},
		Subject: "Review Request ✓",
		Text:    "Hi Ada,\n\nBob has requested a review from you.\nhttps://testflow.example.com/sessions/3\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	r := <-got
	if r.hello != helloName() || r.hello == "" {
		t.Errorf("EHLO %q, want %q", r.hello, helloName())
	}
	if !r.tls || r.auth != "\x00user\x00secret" || r.from != "FROM:<testflow@example.com>" || r.to != "TO:<ada@example.com>" {
		t.Errorf("received = %+v", r)
	}
	head, body, _ := strings.Cut(r.data, "\n\n")
	msg, err := mail.ReadMessage(strings.NewReader(r.data))
	if err != nil {
		t.Fatalf("message does not parse: %v\n%s", err, r.data)
	}
	if strings.Contains(head, "\nBcc:") {
		t.Errorf("a name injected a header:\n%s", head)
	}
	to, err := msg.Header.AddressList("To")
	if err != nil || len(to) != 1 || to[0].Address != "ada@example.com" {
		t.Errorf("To = %v, %v", to, err)
	}
	if subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject")); err != nil || subject != "Review Request ✓" {
		t.Errorf("Subject = %q, %v", subject, err)
	}
	if msg.Header.Get("Content-Type") != "text/plain; charset=utf-8" || !strings.HasSuffix(msg.Header.Get("Message-ID"), "@example.com>") {
		t.Errorf("headers:\n%s", head)
	}
	text, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
	if err != nil || !bytes.Contains(text, []byte("Bob has requested a review from you.")) {
		t.Errorf("body = %q, %v", text, err)
	}
}

func TestSMTPRequiresStartTLS(t *testing.T) {
	port, client, got := fakeServer(t, false)
	s := &SMTP{Host: "127.0.0.1", Port: port, Username: "user", Password: "secret",
		From: mail.Address{Address: "testflow@example.com"}, TLSConfig: client, Timeout: 5 * time.Second}
	err := s.Send(t.Context(), Message{To: mail.Address{Address: "ada@example.com"}, Subject: "x", Text: "y"})
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Errorf("err = %v, want a STARTTLS error", err)
	}
	if r := <-got; r.auth != "" {
		t.Errorf("credentials sent without TLS: %+v", r)
	}
}

func TestSMTPNoServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	s := &SMTP{Host: "127.0.0.1", Port: port, From: mail.Address{Address: "a@b.c"}, Timeout: time.Second}
	if err := s.Send(t.Context(), Message{To: mail.Address{Address: "x@y.z"}}); err == nil || !strings.Contains(err.Error(), strconv.Itoa(port)) {
		t.Errorf("err = %v", err)
	}
	if err := s.Send(t.Context(), Message{}); err == nil {
		t.Error("message without recipient sent")
	}
}

func TestLog(t *testing.T) {
	var buf bytes.Buffer
	l := Log{Logger: slog.New(slog.NewTextHandler(&buf, nil))}
	if err := l.Send(t.Context(), Message{To: mail.Address{Address: "ada@example.com"}, Subject: "Review Request", Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, "email.to=ada@example.com") || !strings.Contains(out, "Review Request") {
		t.Errorf("log = %q", out)
	}
}
