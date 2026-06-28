package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// ============================================================
// Email Package - حزمة البريد الإلكتروني
// ============================================================

// EmailConfig تكوين خادم البريد الإلكتروني
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	UseTLS       bool
	FromAddress  string
	FromName     string
}

// EmailMessage رسالة بريد إلكتروني
type EmailMessage struct {
	From        string
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []*Attachment
	Headers     map[string]string
	Priority    string // low, normal, high, urgent
}

// Attachment مرفق بريد إلكتروني
type Attachment struct {
	Filename string
	Content  []byte
	MimeType string
}

// EmailClient عميل البريد الإلكتروني
type EmailClient struct {
	config *EmailConfig
}

// NewEmailClient إنشاء عميل بريد إلكتروني جديد
func NewEmailClient(config *EmailConfig) *EmailClient {
	return &EmailClient{
		config: config,
	}
}

// Send إرسال بريد إلكتروني
func (c *EmailClient) Send(msg *EmailMessage) error {
	// إنشاء عنوان المرسل
	from := mail.Address{
		Name:    c.config.FromName,
		Address: c.config.FromAddress,
	}

	// إنشاء عناوين المستلمين
	var to []mail.Address
	for _, addr := range msg.To {
		to = append(to, mail.Address{Address: addr})
	}

	// إنشاء الرسالة
	message := ""
	message += fmt.Sprintf("From: %s\r\n", from.String())
	message += fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", "))

	if len(msg.CC) > 0 {
		message += fmt.Sprintf("CC: %s\r\n", strings.Join(msg.CC, ", "))
	}

	message += fmt.Sprintf("Subject: %s\r\n", msg.Subject)

	// إضافة الأولوية
	if msg.Priority != "" {
		message += fmt.Sprintf("X-Priority: %s\r\n", msg.Priority)
	}

	// إضافة الرؤوس المخصصة
	for key, value := range msg.Headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	message += "MIME-Version: 1.0\r\n"

	// إذا كان هناك مرفقات، استخدم multipart/mixed
	if len(msg.Attachments) > 0 {
		boundary := fmt.Sprintf("boundary=%s", generateBoundary())
		message += fmt.Sprintf("Content-Type: multipart/mixed; %s\r\n", boundary)
		message += "\r\n"

		// إضافة النص
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/plain; charset=utf-8\r\n"
		message += "\r\n"
		message += msg.Body
		message += "\r\n"

		// إضافة المرفقات
		for _, att := range msg.Attachments {
			message += fmt.Sprintf("--%s\r\n", boundary)
			message += fmt.Sprintf("Content-Type: %s\r\n", att.MimeType)
			message += fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename)
			message += "Content-Transfer-Encoding: base64\r\n"
			message += "\r\n"
			message += string(att.Content)
			message += "\r\n"
		}

		message += fmt.Sprintf("--%s--\r\n", boundary)
	} else {
		// رسالة نصية بسيطة
		message += "Content-Type: text/plain; charset=utf-8\r\n"
		message += "\r\n"
		message += msg.Body
	}

	// الاتصال بخادم SMTP
	var auth smtp.Auth
	if c.config.SMTPUsername != "" && c.config.SMTPPassword != "" {
		auth = smtp.PlainAuth("", c.config.SMTPUsername, c.config.SMTPPassword, c.config.SMTPHost)
	}

	var client *smtp.Client
	var err error

	if c.config.UseTLS {
		// الاتصال مع TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         c.config.SMTPHost,
		}

		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort), tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}

		// إنشاء عميل SMTP من الاتصال TLS
		host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// الاتصال بدون TLS
		client, err = smtp.Dial(fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort))
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}

		// ترقية إلى STARTTLS إذا كان متاحاً
		if err := client.StartTLS(&tls.Config{
			InsecureSkipVerify: false,
			ServerName:         c.config.SMTPHost,
		}); err == nil {
			// تم ترقية الاتصال بنجاح
		}
	}

	defer client.Close()

	// المصادقة
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// إرسال الرسالة
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr.Address); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr.Address, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create data writer: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// SendAsync إرسال بريد إلكتروني بشكل غير متزامن
func (c *EmailClient) SendAsync(msg *EmailMessage, callback func(error)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		err := c.Send(msg)
		if callback != nil {
			callback(err)
		}
	}()
}

// Validate التحقق من صحة رسالة البريد الإلكتروني
func (c *EmailClient) Validate(msg *EmailMessage) error {
	if msg.From == "" {
		return fmt.Errorf("from address is required")
	}

	if len(msg.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if msg.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if msg.Body == "" && msg.HTMLBody == "" {
		return fmt.Errorf("body or HTML body is required")
	}

	// التحقق من صحة عناوين البريد الإلكتروني
	for _, addr := range msg.To {
		if !isValidEmail(addr) {
			return fmt.Errorf("invalid email address: %s", addr)
		}
	}

	for _, addr := range msg.CC {
		if !isValidEmail(addr) {
			return fmt.Errorf("invalid CC email address: %s", addr)
		}
	}

	for _, addr := range msg.BCC {
		if !isValidEmail(addr) {
			return fmt.Errorf("invalid BCC email address: %s", addr)
		}
	}

	return nil
}

// isValidEmail التحقق من صحة عنوان البريد الإلكتروني
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// generateBoundary توليد حدود multipart
func generateBoundary() string {
	return fmt.Sprintf("----=%d", time.Now().UnixNano())
}

// EmailServer خادم بريد إلكتروني لاستقبال الرسائل
type EmailServer struct {
	addr     string
	handler  func(*EmailMessage) error
	listener net.Listener
}

// NewEmailServer إنشاء خادم بريد إلكتروني جديد
func NewEmailServer(addr string, handler func(*EmailMessage) error) *EmailServer {
	return &EmailServer{
		addr:    addr,
		handler: handler,
	}
}

// Start بدء تشغيل خادم البريد الإلكتروني
func (s *EmailServer) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	go s.acceptConnections()

	return nil
}

// Stop إيقاف خادم البريد الإلكتروني
func (s *EmailServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections قبول الاتصالات
func (s *EmailServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection معالجة الاتصال
func (s *EmailServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// قراءة رسالة SMTP
	// (تنفيذ مبسط للإيضاح)
	msg := &EmailMessage{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Email",
		Body:    "Test Body",
	}

	if s.handler != nil {
		s.handler(msg)
	}
}
