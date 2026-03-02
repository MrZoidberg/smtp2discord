package main

import (
	"log/slog"
	"os"

	"github.com/donserdal/smtp2discord/internal/config"
	"github.com/donserdal/smtp2discord/internal/smtp"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.Load()

	if cfg.Webhook == "" {
		logger.Error("webhook URL is required; set -webhook flag")
		os.Exit(1)
	}

	server := smtp.NewServer(cfg, logger)

	logger.Info("starting SMTP server", "addr", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
