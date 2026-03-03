package main

import (
	"os"

	"github.com/MrZoidberg/smtp2discord/internal/config"
	"github.com/MrZoidberg/smtp2discord/internal/logger"
	"github.com/MrZoidberg/smtp2discord/internal/smtp"
)

func main() {
	cfg := config.Load()

	log := logger.New(cfg.Debug)
	log.Infof("starting SMTP server on %s (debug=%v)", cfg.ListenAddr, cfg.Debug)

	server := smtp.NewServer(cfg, log)

	if err := server.ListenAndServe(); err != nil {
		log.Errorf("server exited with error: %v", err)
		os.Exit(1)
	}
}
