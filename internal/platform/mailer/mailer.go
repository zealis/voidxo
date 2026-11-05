// Package mailer 负责真正的邮件投递。
// 生产使用 SMTP，本地开发在未配置 SMTP 时退化为日志发送器，绝不静默丢信。
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidMessage = errors.New("invalid mail message")

type Message struct {
	To      string
	Subject string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	// Mode 取值 none、starttls、tls；生产建议 starttls 或 tls。
	Mode string
}

// LogSender 只在开发环境使用：把邮件写入日志，便于本地取验证码。
type LogSender struct {
	logger *slog.Logger
}

func NewLogSender(logger *slog.Logger) *LogSender {
	return &LogSender{logger: logger}
}

func (s *LogSender) Send(_ context.Context, message Message) error {
	if err := validate(message); err != nil {
		return err
	}
	s.logger.Info("email delivery (dev log sender)",
		"to", message.To, "subject", message.Subject, "text", message.Text)
	return nil
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	if strings.TrimSpace(cfg.Host) == "" || cfg.Port <= 0 {
		return nil, errors.New("smtp host and port are required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return nil, errors.New("smtp from address is required")
	}
	switch cfg.Mode {
	case "", "none", "starttls", "tls":
	default:
		return nil, fmt.Errorf("unsupported smtp mode %q", cfg.Mode)
	}
	return &SMTPSender{cfg: cfg}, nil
}

func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	if err := validate(message); err != nil {
		return err
	}
	payload, err := s.buildPayload(message)
	if err != nil {
		return err
	}

	address := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}
	defer connection.Close()

	client, err := smtp.NewClient(connection, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if s.cfg.Mode == "tls" {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start smtp tls: %w", err)
		}
	} else if s.cfg.Mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start smtp tls: %w", err)
		}
	}

	if s.cfg.Username != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("smtp server does not support AUTH")
		}
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth failed: %w", err)
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}
	return client.Quit()
}

func (s *SMTPSender) buildPayload(message Message) ([]byte, error) {
	from := s.cfg.From
	if name := strings.TrimSpace(s.cfg.FromName); name != "" {
		from = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", name), s.cfg.From)
	}
	headers := []string{
		"From: " + from,
		"To: " + message.To,
		"Subject: " + mime.QEncoding.Encode("UTF-8", message.Subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + normaliseNewlines(message.Text)), nil
}

// validate 拒绝头注入：收件人与主题里不允许出现换行。
func validate(message Message) error {
	if strings.TrimSpace(message.To) == "" || !strings.Contains(message.To, "@") {
		return fmt.Errorf("%w: recipient is required", ErrInvalidMessage)
	}
	if strings.TrimSpace(message.Subject) == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidMessage)
	}
	for _, header := range []string{message.To, message.Subject} {
		if strings.ContainsAny(header, "\r\n") {
			return fmt.Errorf("%w: header contains newline", ErrInvalidMessage)
		}
	}
	return nil
}

func normaliseNewlines(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\n", "\r\n")
}
