package smtp

import (
	"fmt"
	"log/slog"

	smtpsrv "github.com/alash3al/go-smtpsrv"

	"github.com/donserdal/smtp2discord/internal/config"
	"github.com/donserdal/smtp2discord/internal/discord"
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
	cfg := smtpsrv.ServerConfig{
		ReadTimeout:     s.cfg.ReadTimeout,
		WriteTimeout:    s.cfg.WriteTimeout,
		ListenAddr:      s.cfg.ListenAddr,
		MaxMessageBytes: s.cfg.MaxMessageSize,
		BannerDomain:    s.cfg.ServerName,
		Handler:         smtpsrv.HandlerFunc(s.handleMessage),
	}
	return smtpsrv.ListenAndServe(&cfg)
}

// handleMessage processes an incoming SMTP message and forwards it to Discord.
func (s *Server) handleMessage(c *smtpsrv.Context) error {
	msg, err := c.Parse()
	if err != nil {
		return fmt.Errorf("cannot parse message: %w", err)
	}

	discordMsg := discord.Message{
		Username:  s.cfg.Author,
		AvatarURL: s.cfg.AvatarURL,
		Content:   msg.TextBody,
	}

	if err := s.discord.Send(discordMsg); err != nil {
		s.logger.Error("failed to forward message to Discord", "error", err)
		return fmt.Errorf("cannot forward message to Discord: %w", err)
	}

	s.logger.Info("message forwarded to Discord")
	return nil
}
