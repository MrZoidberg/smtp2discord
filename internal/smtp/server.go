package smtp

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"text/template"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"golang.org/x/crypto/bcrypt"

	"github.com/MrZoidberg/smtp2discord/internal/config"
	"github.com/MrZoidberg/smtp2discord/internal/discord"
	"github.com/MrZoidberg/smtp2discord/internal/logger"
)

// Server wraps the SMTP server configuration and its dependencies.
type Server struct {
	cfg             *config.Config
	discord         *discord.Client
	logger          *logger.Logger
	messageTemplate *template.Template
}

// NewServer creates a new SMTP server with the given configuration and logger.
func NewServer(cfg *config.Config, logger *logger.Logger) *Server {
	messageTemplate := template.Must(template.New("discord-message").Option("missingkey=zero").Parse(cfg.MessageTemplate))

	return &Server{
		cfg:             cfg,
		discord:         discord.NewClient(cfg.Webhook, logger),
		logger:          logger,
		messageTemplate: messageTemplate,
	}
}

// ListenAndServe starts the SMTP server and blocks until it exits.
func (s *Server) ListenAndServe() error {
	smtpServer := gosmtp.NewServer(&backend{server: s})
	smtpServer.Addr = s.cfg.ListenAddr
	smtpServer.Domain = s.cfg.ServerName
	smtpServer.ReadTimeout = s.cfg.ReadTimeout
	smtpServer.WriteTimeout = s.cfg.WriteTimeout
	smtpServer.MaxMessageBytes = int64(s.cfg.MaxMessageSize)

	if s.cfg.Debug {
		smtpServer.Debug = s.logger
	}

	if err := smtpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("start SMTP server: %w", err)
	}

	return nil
}

// handleMessage processes an incoming SMTP message and forwards it to Discord.
// log should be the per-session logger so log entries carry the remote address.
func (s *Server) handleMessage(log *logger.Logger, rawMessage io.Reader, envelopeFrom string) error {
	msg, err := mail.ReadMessage(rawMessage)
	if err != nil {
		return fmt.Errorf("cannot parse message: %w", err)
	}

	from, subject := extractMessageMetadata(msg, envelopeFrom)

	body, err := extractTextBody(msg)
	if err != nil {
		return fmt.Errorf("cannot extract message body: %w", err)
	}

	content, err := s.formatDiscordContent(from, subject, body)
	if err != nil {
		return fmt.Errorf("cannot format discord content: %w", err)
	}

	discordMsg := discord.Message{
		Username:  s.cfg.Author,
		AvatarURL: s.cfg.AvatarURL,
		Content:   content,
	}

	if err := s.discord.Send(discordMsg); err != nil {
		return fmt.Errorf("cannot forward message to Discord: %w", err)
	}

	log.Infof("mail accepted from=%s subject=%q", from, subject)
	return nil
}

type backend struct {
	server *Server
}

func (b *backend) NewSession(conn *gosmtp.Conn) (gosmtp.Session, error) {
	remoteAddr := conn.Conn().RemoteAddr().String()
	log := b.server.logger.With(fmt.Sprintf("[%s]", remoteAddr))
	log.Debugf("new SMTP session")
	return &session{server: b.server, logger: log}, nil
}

type session struct {
	server        *Server
	logger        *logger.Logger
	authenticated bool
	mailFrom      string
}

func (s *session) Mail(from string, _ *gosmtp.MailOptions) error {
	if s.server.cfg.SMTPUsername != "" && !s.authenticated {
		s.logger.Debugf("MAIL FROM rejected: authentication required, from=%s", from)
		return &gosmtp.SMTPError{Code: 530, Message: "Authentication required"}
	}

	s.mailFrom = strings.TrimSpace(from)
	s.logger.Debugf("MAIL FROM accepted, from=%s", s.mailFrom)

	return nil
}

func (s *session) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	s.logger.Debugf("RCPT TO %s", to)
	return nil
}

func (s *session) Data(reader io.Reader) error {
	s.logger.Debugf("DATA command received")
	if err := s.server.handleMessage(s.logger, reader, s.mailFrom); err != nil {
		s.logger.Infof("mail rejected from=%s: %v", s.mailFrom, err)
		return err
	}
	return nil
}

func (s *session) Reset() {
	s.logger.Debugf("RSET command received")
	s.mailFrom = ""
}

func (s *session) Logout() error {
	s.logger.Debugf("QUIT command received")
	return nil
}

func (s *session) AuthMechanisms() []string {
	if s.server.cfg.SMTPUsername == "" {
		return nil
	}

	return []string{sasl.Plain}
}

func (s *session) Auth(mech string) (sasl.Server, error) {
	if s.server.cfg.SMTPUsername == "" {
		return nil, &gosmtp.SMTPError{Code: 502, Message: "Authentication not supported"}
	}

	if mech != sasl.Plain {
		return nil, gosmtp.ErrAuthFailed
	}

	return sasl.NewPlainServer(func(_, username, password string) error {
		if username != s.server.cfg.SMTPUsername {
			s.logger.Debugf("AUTH failed: wrong username=%s", username)
			return gosmtp.ErrAuthFailed
		}
		if err := bcrypt.CompareHashAndPassword([]byte(s.server.cfg.SMTPPassHash), []byte(password)); err != nil {
			s.logger.Debugf("AUTH failed: wrong password for username=%s", username)
			return gosmtp.ErrAuthFailed
		}

		s.authenticated = true
		s.logger.Debugf("AUTH successful, username=%s", username)
		return nil
	}), nil
}

func extractTextBody(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", fmt.Errorf("read message body: %w", err)
		}
		return strings.TrimSpace(string(body)), nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		body, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return "", fmt.Errorf("read message body after content type parse failure: %w", readErr)
		}
		return strings.TrimSpace(string(body)), nil
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		body, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return "", fmt.Errorf("read non-multipart body: %w", readErr)
		}
		return strings.TrimSpace(string(body)), nil
	}

	boundary := params["boundary"]
	if boundary == "" {
		body, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return "", fmt.Errorf("read multipart body without boundary: %w", readErr)
		}
		return strings.TrimSpace(string(body)), nil
	}

	body, err := extractTextFromMultipart(msg.Body, boundary)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(body), nil
}

func extractTextFromMultipart(body io.Reader, boundary string) (string, error) {
	reader := multipart.NewReader(body, boundary)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read next multipart part: %w", err)
		}

		partContentType := part.Header.Get("Content-Type")
		if partContentType == "" {
			partContentType = "text/plain"
		}

		mediaType, params, parseErr := mime.ParseMediaType(partContentType)
		if parseErr != nil {
			mediaType = "text/plain"
		}

		switch {
		case mediaType == "text/plain":
			content, readErr := io.ReadAll(part)
			if readErr != nil {
				return "", fmt.Errorf("read multipart text part: %w", readErr)
			}
			text := strings.TrimSpace(string(content))
			if text != "" {
				return text, nil
			}
		case strings.HasPrefix(mediaType, "multipart/"):
			nestedBoundary := params["boundary"]
			if nestedBoundary == "" {
				continue
			}

			nestedText, nestedErr := extractTextFromMultipart(part, nestedBoundary)
			if nestedErr != nil {
				return "", nestedErr
			}
			if nestedText != "" {
				return nestedText, nil
			}
		}
	}

	return "", nil
}

func extractMessageMetadata(msg *mail.Message, envelopeFrom string) (string, string) {
	fromHeader := decodeMIMEHeader(strings.TrimSpace(msg.Header.Get("From")))
	subject := decodeMIMEHeader(strings.TrimSpace(msg.Header.Get("Subject")))

	if fromHeader == "" {
		return normalizeFrom(envelopeFrom), subject
	}

	addresses, err := mail.ParseAddressList(fromHeader)
	if err != nil || len(addresses) == 0 {
		return fromHeader, subject
	}

	normalized := make([]string, 0, len(addresses))
	for _, address := range addresses {
		normalized = append(normalized, address.String())
	}

	return strings.Join(normalized, ", "), subject
}

func normalizeFrom(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	address, err := mail.ParseAddress(value)
	if err != nil {
		return value
	}

	return address.String()
}

func decodeMIMEHeader(value string) string {
	if value == "" {
		return ""
	}

	decoded, err := new(mime.WordDecoder).DecodeHeader(value)
	if err != nil {
		return value
	}

	return decoded
}

func (s *Server) formatDiscordContent(from, subject, body string) (string, error) {
	type messageTemplateData struct {
		From    string
		Subject string
		Body    string
	}

	var buffer bytes.Buffer
	err := s.messageTemplate.Execute(&buffer, messageTemplateData{
		From:    from,
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return "", fmt.Errorf("execute message template: %w", err)
	}

	return strings.TrimSpace(buffer.String()), nil
}
