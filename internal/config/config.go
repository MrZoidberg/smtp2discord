package config

import (
	"flag"
	"time"
)

// Config holds the application configuration loaded from command-line flags.
type Config struct {
	ServerName     string
	ListenAddr     string
	Author         string
	AvatarURL      string
	Webhook        string
	MaxMessageSize int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

// Load parses command-line flags and returns a populated Config.
func Load() *Config {
	serverName := flag.String("name", "smtp2discord", "the server name")
	listenAddr := flag.String("listen", ":smtp", "the smtp address to listen on")
	author := flag.String("author", "", "the username for the discord webhook")
	avatarURL := flag.String("avatar-url", "", "the avatar URL of the bot")
	webhook := flag.String("webhook", "", "the discord webhook URL")
	maxMessageSize := flag.Int64("msglimit", 1024*1024*2, "maximum incoming message size in bytes")
	readTimeout := flag.Int("timeout.read", 5, "the read timeout in seconds")
	writeTimeout := flag.Int("timeout.write", 5, "the write timeout in seconds")

	flag.Parse()

	return &Config{
		ServerName:     *serverName,
		ListenAddr:     *listenAddr,
		Author:         *author,
		AvatarURL:      *avatarURL,
		Webhook:        *webhook,
		MaxMessageSize: int(*maxMessageSize),
		ReadTimeout:    time.Duration(*readTimeout) * time.Second,
		WriteTimeout:   time.Duration(*writeTimeout) * time.Second,
	}
}
