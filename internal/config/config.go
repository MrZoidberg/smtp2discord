package config

import (
	_ "embed"
	"fmt"
	"os"
	templatepkg "text/template"
	"time"

	"github.com/jessevdk/go-flags"
	"golang.org/x/crypto/bcrypt"
)

//go:embed default_message.tmpl
var defaultMessageTemplate string

// Options holds the application configuration parsed from command-line flags.
// Every field also accepts a corresponding environment variable (listed in the env tag).
// Flags take precedence over environment variables.
type Options struct {
	ServerName     string `long:"name"                  default:"smtp2discord" env:"SMTP2DISCORD_NAME"                  description:"The server banner name"`
	ListenAddr     string `long:"listen"                default:":smtp"        env:"SMTP2DISCORD_LISTEN"                description:"SMTP address to listen on"`
	SMTPUsername   string `long:"smtp-user"             default:""             env:"SMTP2DISCORD_SMTP_USER"             description:"SMTP AUTH PLAIN username"`
	SMTPPassHash   string `long:"smtp-pass-hash"        default:""             env:"SMTP2DISCORD_SMTP_PASS_HASH"        description:"SMTP AUTH PLAIN password hash (bcrypt)"`
	TemplateFile   string `long:"message-template-file" default:""             env:"SMTP2DISCORD_MESSAGE_TEMPLATE_FILE" description:"Path to Go template file for Discord message formatting"`
	Author         string `long:"author"                default:""             env:"SMTP2DISCORD_AUTHOR"                description:"Username shown on Discord messages"`
	AvatarURL      string `long:"avatar-url"            default:""             env:"SMTP2DISCORD_AVATAR_URL"            description:"Avatar URL of the Discord bot"`
	Webhook        string `long:"webhook"               default:""             env:"SMTP2DISCORD_WEBHOOK" required:"true" description:"Discord webhook URL"`
	MaxMessageSize int    `long:"msglimit"              default:"2097152"      env:"SMTP2DISCORD_MSG_LIMIT"             description:"Maximum incoming message size in bytes"`
	ReadTimeout    int    `long:"timeout.read"          default:"5"            env:"SMTP2DISCORD_TIMEOUT_READ"          description:"Read timeout in seconds"`
	WriteTimeout   int    `long:"timeout.write"         default:"5"            env:"SMTP2DISCORD_TIMEOUT_WRITE"         description:"Write timeout in seconds"`
	Debug          bool   `long:"debug"                 default:"false"        env:"SMTP2DISCORD_DEBUG"                 description:"Enable debug logging (verbose SMTP protocol and Discord webhook logs)"`
}

// Config holds resolved configuration with typed durations.
type Config struct {
	ServerName      string
	ListenAddr      string
	SMTPUsername    string
	SMTPPassHash    string
	MessageTemplate string
	Author          string
	AvatarURL       string
	Webhook         string
	MaxMessageSize  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	Debug           bool
}

// Load parses command-line flags and returns a populated Config.
// On parse error or --help, go-flags prints a message and exits.
func Load() *Config {
	var opts Options
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	if (opts.SMTPUsername == "") != (opts.SMTPPassHash == "") {
		fmt.Fprintln(os.Stderr, "--smtp-user and --smtp-pass-hash must be provided together")
		os.Exit(1)
	}

	if opts.SMTPPassHash != "" {
		if _, err := bcrypt.Cost([]byte(opts.SMTPPassHash)); err != nil {
			fmt.Fprintf(os.Stderr, "invalid bcrypt hash in --smtp-pass-hash/SMTP2DISCORD_SMTP_PASS_HASH: %v\n", err)
			os.Exit(1)
		}
	}

	messageTemplate := defaultMessageTemplate
	if opts.TemplateFile != "" {
		templateBytes, err := os.ReadFile(opts.TemplateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read --message-template-file: %v\n", err)
			os.Exit(1)
		}
		messageTemplate = string(templateBytes)
	}

	if _, err := templatepkg.New("discord-message").Option("missingkey=zero").Parse(messageTemplate); err != nil {
		fmt.Fprintf(os.Stderr, "invalid message template: %v\n", err)
		os.Exit(1)
	}

	return &Config{
		ServerName:      opts.ServerName,
		ListenAddr:      opts.ListenAddr,
		SMTPUsername:    opts.SMTPUsername,
		SMTPPassHash:    opts.SMTPPassHash,
		MessageTemplate: messageTemplate,
		Author:          opts.Author,
		AvatarURL:       opts.AvatarURL,
		Webhook:         opts.Webhook,
		MaxMessageSize:  opts.MaxMessageSize,
		ReadTimeout:     time.Duration(opts.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(opts.WriteTimeout) * time.Second,
		Debug:           opts.Debug,
	}
}
