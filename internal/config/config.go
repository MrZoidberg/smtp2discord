package config

import (
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

// Options holds the application configuration parsed from command-line flags.
type Options struct {
	ServerName     string `long:"name"          default:"smtp2discord"           description:"The server banner name"`
	ListenAddr     string `long:"listen"        default:":smtp"                  description:"SMTP address to listen on"`
	Author         string `long:"author"        default:""                       description:"Username shown on Discord messages"`
	AvatarURL      string `long:"avatar-url"    default:""                       description:"Avatar URL of the Discord bot"`
	Webhook        string `long:"webhook"       default:""         required:"true" description:"Discord webhook URL"`
	MaxMessageSize int    `long:"msglimit"      default:"2097152"                description:"Maximum incoming message size in bytes"`
	ReadTimeout    int    `long:"timeout.read"  default:"5"                      description:"Read timeout in seconds"`
	WriteTimeout   int    `long:"timeout.write" default:"5"                      description:"Write timeout in seconds"`
}

// Config holds resolved configuration with typed durations.
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
// On parse error or --help, go-flags prints a message and exits.
func Load() *Config {
	var opts Options
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	return &Config{
		ServerName:     opts.ServerName,
		ListenAddr:     opts.ListenAddr,
		Author:         opts.Author,
		AvatarURL:      opts.AvatarURL,
		Webhook:        opts.Webhook,
		MaxMessageSize: opts.MaxMessageSize,
		ReadTimeout:    time.Duration(opts.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(opts.WriteTimeout) * time.Second,
	}
}
