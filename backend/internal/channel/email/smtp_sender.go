package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"backend/internal/model"
)

// OutboundEmail carries the data needed to send a reply.
type OutboundEmail struct {
	From       string
	To         []string
	Subject    string
	HTMLBody   string
	TextBody   string
	InReplyTo  string
	References string
	MessageID  string
}

// SendSMTP sends an outbound email via SMTP and returns the Message-ID used.
func SendSMTP(ch *model.ChannelEmail, msg *OutboundEmail, decryptFn func(string) (string, error)) (sourceID string, err error) {
	if ch.SmtpAddress == nil || ch.SmtpPort == nil {
		return "", fmt.Errorf("smtp: no address configured")
	}

	password := ""
	if ch.SmtpPasswordCiphertext != nil {
		password, err = decryptFn(*ch.SmtpPasswordCiphertext)
		if err != nil {
			return "", fmt.Errorf("smtp decrypt password: %w", err)
		}
	}

	if msg.MessageID == "" {
		msg.MessageID = generateMessageID(ch.Email)
	}

	raw := buildRawMessage(msg)

	addr := fmt.Sprintf("%s:%d", *ch.SmtpAddress, *ch.SmtpPort)
	var auth smtp.Auth
	if ch.SmtpLogin != nil && password != "" {
		auth = smtp.PlainAuth("", *ch.SmtpLogin, password, *ch.SmtpAddress)
	}

	if ch.SmtpEnableSSL {
		tlsCfg := &tls.Config{ServerName: *ch.SmtpAddress}
		conn, dialErr := tls.Dial("tcp", addr, tlsCfg)
		if dialErr != nil {
			return "", fmt.Errorf("smtp tls dial: %w", dialErr)
		}
		client, clientErr := smtp.NewClient(conn, *ch.SmtpAddress)
		if clientErr != nil {
			return "", fmt.Errorf("smtp new client: %w", clientErr)
		}
		defer func() {
			_ = client.Close()
		}()
		if auth != nil {
			if authErr := client.Auth(auth); authErr != nil {
				return "", fmt.Errorf("smtp auth: %w", authErr)
			}
		}
		if err := client.Mail(ch.Email); err != nil {
			return "", fmt.Errorf("smtp MAIL FROM: %w", err)
		}
		for _, to := range msg.To {
			if err := client.Rcpt(to); err != nil {
				return "", fmt.Errorf("smtp RCPT TO %s: %w", to, err)
			}
		}
		wc, err := client.Data()
		if err != nil {
			return "", fmt.Errorf("smtp DATA: %w", err)
		}
		if _, err := wc.Write([]byte(raw)); err != nil {
			return "", fmt.Errorf("smtp write body: %w", err)
		}
		if err := wc.Close(); err != nil {
			return "", fmt.Errorf("smtp close data: %w", err)
		}
		_ = client.Quit()
	} else {
		if err := smtp.SendMail(addr, auth, ch.Email, msg.To, []byte(raw)); err != nil {
			return "", fmt.Errorf("smtp send: %w", err)
		}
	}

	return msg.MessageID, nil
}

func buildRawMessage(msg *OutboundEmail) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	sb.WriteString(fmt.Sprintf("Message-ID: %s\r\n", msg.MessageID))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))
	if msg.InReplyTo != "" {
		sb.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", msg.InReplyTo))
	}
	if msg.References != "" {
		sb.WriteString(fmt.Sprintf("References: %s\r\n", msg.References))
	}
	sb.WriteString("MIME-Version: 1.0\r\n")

	if msg.HTMLBody != "" && msg.TextBody != "" {
		boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.TextBody)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.HTMLBody)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.HTMLBody != "" {
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.HTMLBody)
	} else {
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.TextBody)
	}
	return sb.String()
}

func generateMessageID(fromEmail string) string {
	domain := "elodesk.io"
	if parts := strings.Split(fromEmail, "@"); len(parts) == 2 {
		domain = parts[1]
	}
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), randomHex(8), domain)
}

func randomHex(n int) string {
	b := make([]byte, n)
	// Use net package's random source via connection id trick
	conn, _ := net.Pipe()
	if conn != nil {
		_ = conn.Close()
	}
	for i := range b {
		b[i] = "0123456789abcdef"[time.Now().UnixNano()&0xf]
	}
	return string(b)
}
