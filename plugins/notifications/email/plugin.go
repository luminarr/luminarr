// Package email implements a Luminarr notification plugin that sends events
// as plain-text emails via an SMTP server.
package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterNotifier("email", func(settings json.RawMessage) (plugin.Notifier, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("email: invalid settings: %w", err)
		}
		if cfg.Host == "" {
			return nil, fmt.Errorf("email: host is required")
		}
		if cfg.From == "" {
			return nil, fmt.Errorf("email: from is required")
		}
		if len(cfg.To) == 0 {
			return nil, fmt.Errorf("email: to is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterNotifierSanitizer("email", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["password"]; ok {
			m["password"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the user-supplied settings for an email notifier.
type Config struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"` // default: 587
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	TLS      bool     `json:"tls,omitempty"` // use TLS (port 465 style); false = STARTTLS
}

// Notifier is an email notifier plugin instance.
type Notifier struct {
	cfg Config
}

// New creates a new Notifier from the given config.
func New(cfg Config) *Notifier {
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return &Notifier{cfg: cfg}
}

func (n *Notifier) Name() string { return "Email" }

// Notify sends the event as a plain-text email.
func (n *Notifier) Notify(_ context.Context, event plugin.NotificationEvent) error {
	subject := fmt.Sprintf("[Luminarr] %s", event.Type)
	body := fmt.Sprintf("Event: %s\nTime: %s\n\n%s",
		event.Type,
		event.Timestamp.UTC().Format(time.RFC1123),
		event.Message,
	)

	msg := buildMessage(n.cfg.From, n.cfg.To, subject, body)
	addr := fmt.Sprintf("%s:%d", n.cfg.Host, n.cfg.Port)

	if n.cfg.TLS {
		return n.sendTLS(addr, msg)
	}
	return n.sendSTARTTLS(addr, msg)
}

// Test sends a test email to verify SMTP connectivity.
func (n *Notifier) Test(ctx context.Context) error {
	return n.Notify(ctx, plugin.NotificationEvent{
		Type:      plugin.EventType("test"),
		Timestamp: time.Now().UTC(),
		Message:   "Luminarr email test — connection successful",
	})
}

// sendSTARTTLS connects on a plain port and upgrades to TLS via STARTTLS.
func (n *Notifier) sendSTARTTLS(addr string, msg []byte) error {
	var auth smtp.Auth
	if n.cfg.Username != "" {
		auth = smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.Host)
	}
	return smtp.SendMail(addr, auth, n.cfg.From, n.cfg.To, msg)
}

// sendTLS connects directly over TLS (implicit TLS, port 465).
func (n *Notifier) sendTLS(addr string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: n.cfg.Host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("email: TLS dial: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("email: SMTP client: %w", err)
	}
	defer client.Quit() //nolint:errcheck

	if n.cfg.Username != "" {
		auth := smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email: SMTP auth: %w", err)
		}
	}

	if err := client.Mail(n.cfg.From); err != nil {
		return fmt.Errorf("email: MAIL FROM: %w", err)
	}
	for _, to := range n.cfg.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("email: RCPT TO %q: %w", to, err)
		}
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("email: writing message: %w", err)
	}
	return wc.Close()
}

// buildMessage formats a minimal RFC 5322 email message.
func buildMessage(from string, to []string, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}
