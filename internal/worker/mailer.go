package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
)

// Mailer is the SMTP-side dependency the notify worker uses to deliver
// queued notifications. The package ships two implementations: SMTPMailer
// (real SMTP, used in prod and against Mailpit in dev) and noopMailer
// (no SMTP_HOST configured — notifications are persisted but never
// shipped over the wire).
type Mailer interface {
	Send(ctx context.Context, n notifydom.Notification) error
}

// MailerConfig is the bundle of SMTP knobs the platform layer reads
// from env vars and threads here.
type MailerConfig struct {
	Logger    *slog.Logger
	Host      string
	Port      int
	Username  string
	Password  string
	From      string
	LookupTo  RecipientLookup
	Subjector Subjector
}

// RecipientLookup resolves a user id into the email address we should
// send to. The default in production is a callback into pgrepo.Users.
type RecipientLookup func(ctx context.Context, userID uuid.UUID) (string, error)

// Subjector turns a notification kind + payload into a (subject, body)
// pair. Callers can swap in a richer renderer later — the default
// implementation in subject.go is intentionally minimal.
type Subjector interface {
	Subject(n notifydom.Notification) string
	Body(n notifydom.Notification) string
}

// NewMailer chooses an implementation based on whether SMTP is
// configured. When Host is empty we return a noopMailer so the
// dispatcher still drains the queue.
func NewMailer(cfg MailerConfig) Mailer {
	if cfg.Subjector == nil {
		cfg.Subjector = defaultSubjector{}
	}
	if cfg.Host == "" {
		return noopMailer{logger: cfg.Logger}
	}
	return &smtpMailer{cfg: cfg}
}

// smtpMailer talks SMTP plain (no TLS) — Mailpit is the local target
// and prod targets are typically a managed relay that handles TLS at
// the network boundary. Add explicit TLS when we move to a hosted
// provider.
type smtpMailer struct{ cfg MailerConfig }

func (m *smtpMailer) Send(ctx context.Context, n notifydom.Notification) error {
	if m.cfg.LookupTo == nil {
		return fmt.Errorf("recipient lookup not configured")
	}
	to, err := m.cfg.LookupTo(ctx, n.UserID)
	if err != nil {
		return fmt.Errorf("recipient lookup: %w", err)
	}
	if to == "" {
		return nil
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	subject := m.cfg.Subjector.Subject(n)
	body := m.cfg.Subjector.Body(n)
	msg := buildEmail(m.cfg.From, to, subject, body)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// noopMailer logs and reports success. Used when SMTP is unconfigured
// so the dispatch loop still marks notifications as sent and the bell
// still surfaces them.
type noopMailer struct{ logger *slog.Logger }

func (m noopMailer) Send(_ context.Context, n notifydom.Notification) error {
	if m.logger != nil {
		m.logger.Debug("notify: SMTP unset, dropping email but marking sent",
			"id", n.ID, "kind", n.Kind, "user", n.UserID)
	}
	return nil
}

// buildEmail composes a minimal RFC 5322 message. Production callers
// should swap in a templated renderer, but for MVP the subject + body
// are enough to wake somebody up.
func buildEmail(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(to)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteString("\r\n")
	b.WriteString("Date: ")
	b.WriteString(time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// defaultSubjector is a tiny canned template generator.
type defaultSubjector struct{}

func (defaultSubjector) Subject(n notifydom.Notification) string {
	switch n.Kind {
	case notifydom.KindUnderAttack:
		return "[Supremacy] Your forces are under attack"
	case notifydom.KindProvinceCaptured:
		return "[Supremacy] You lost a province"
	case notifydom.KindTreatyProposed:
		return "[Supremacy] A new treaty has been proposed"
	case notifydom.KindMatchEnded:
		return "[Supremacy] Match ended"
	case notifydom.KindBotTakeover:
		return "[Supremacy] AI is now controlling your slot"
	default:
		return "[Supremacy] Match update"
	}
}

func (defaultSubjector) Body(n notifydom.Notification) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Notification kind: %s\n", n.Kind)
	fmt.Fprintf(&b, "Match: %s\n", n.MatchID)
	fmt.Fprintf(&b, "Created: %s\n\n", n.CreatedAt.Format(time.RFC3339))
	if len(n.Payload) > 0 {
		b.WriteString("Details:\n")
		for k, v := range n.Payload {
			fmt.Fprintf(&b, "  %s: %v\n", k, v)
		}
	}
	return b.String()
}
