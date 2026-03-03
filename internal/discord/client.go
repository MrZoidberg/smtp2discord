package discord

import (
	"fmt"

	"github.com/go-resty/resty/v2"

	"github.com/MrZoidberg/smtp2discord/internal/logger"
)

// Message represents the payload sent to a Discord webhook.
type Message struct {
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Content   string `json:"content,omitempty"`
}

// Client sends messages to a Discord webhook.
type Client struct {
	webhookURL string
	http       *resty.Client
	logger     *logger.Logger
}

// NewClient creates a new Discord webhook client.
func NewClient(webhookURL string, log *logger.Logger) *Client {
	return &Client{
		webhookURL: webhookURL,
		http:       resty.New(),
		logger:     log,
	}
}

// Send posts a message to the configured Discord webhook.
// It returns an error if the HTTP request fails or Discord responds with a non-204 status.
func (c *Client) Send(msg Message) error {
	c.logger.Debugf("sending Discord webhook username=%q content_len=%d", msg.Username, len(msg.Content))

	resp, err := c.http.R().
		SetHeader("Content-Type", "application/json").
		SetBody(msg).
		Post(c.webhookURL)
	if err != nil {
		return fmt.Errorf("discord webhook request failed: %w", err)
	}

	c.logger.Debugf("Discord webhook response status=%d body=%s", resp.StatusCode(), resp.Body())

	if resp.StatusCode() != 204 {
		return fmt.Errorf("discord webhook returned unexpected status: %s", resp.Status())
	}
	return nil
}
