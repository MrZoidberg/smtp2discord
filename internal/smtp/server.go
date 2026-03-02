package smtp

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"

	"github.com/MrZoidberg/smtp2discord/internal/config"
	"github.com/MrZoidberg/smtp2discord/internal/discord"
)

// Server wraps the SMTP server configuration and its dependencies.
type Server struct {
	cfg     *config.Config
	discord *discord.Client
	logger  *slog.Logger
}

// NewServer creates a new SMTP server with the given configuration and logger.
func NewServer(cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{
		cfg:     cfg,
		discord: discord.NewClient(cfg.Webhook),
		logger:  logger,
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

	if err := smtpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("start SMTP server: %w", err)
	}

	return nil
}

// handleMessage processes an incoming SMTP message and forwards it to Discord.
func (s *Server) handleMessage(rawMessage io.Reader) error {
	msg, err := mail.ReadMessage(rawMessage)
	if err != nil {
		return fmt.Errorf("cannot parse message: %w", err)
	}

	body, err := extractTextBody(msg)
	if err != nil {
		return fmt.Errorf("cannot extract message body: %w", err)
	}

	discordMsg := discord.Message{
		Username:  s.cfg.Author,
		AvatarURL: s.cfg.AvatarURL,
		Content:   body,
	}

	if err := s.discord.Send(discordMsg); err != nil {
		s.logger.Error("failed to forward message to Discord", "error", err)
		return fmt.Errorf("cannot forward message to Discord: %w", err)
	}

	s.logger.Info("message forwarded to Discord")
	return nil
}

type backend struct {
	server *Server
}

func (b *backend) NewSession(*gosmtp.Conn) (gosmtp.Session, error) {
	return &session{server: b.server}, nil
}

type session struct {
	server        *Server
	authenticated bool
}

func (s *session) Mail(string, *gosmtp.MailOptions) error {
	if s.server.cfg.SMTPUsername != "" && !s.authenticated {
		return &gosmtp.SMTPError{Code: 530, Message: "Authentication required"}
	}

	return nil
}

func (s *session) Rcpt(string, *gosmtp.RcptOptions) error {
	return nil
}

func (s *session) Data(reader io.Reader) error {
	return s.server.handleMessage(reader)
}

func (s *session) Reset() {}

func (s *session) Logout() error {
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
		if username != s.server.cfg.SMTPUsername || password != s.server.cfg.SMTPPassword {
			return gosmtp.ErrAuthFailed
		}

		s.authenticated = true
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
