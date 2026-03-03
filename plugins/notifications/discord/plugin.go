// Package discord implements a Luminarr notification plugin that sends events
// as rich embed messages to a Discord channel via a Discord webhook URL.
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterNotifier("discord", func(settings json.RawMessage) (plugin.Notifier, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("discord: invalid settings: %w", err)
		}
		if cfg.WebhookURL == "" {
			return nil, fmt.Errorf("discord: webhook_url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterNotifierSanitizer("discord", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["webhook_url"]; ok {
			m["webhook_url"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the user-supplied settings for a Discord notifier.
type Config struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`   // override webhook display name
	AvatarURL  string `json:"avatar_url,omitempty"` // override webhook avatar
}

// Notifier is a Discord notifier plugin instance.
type Notifier struct {
	cfg    Config
	client *http.Client
}

// New creates a new Notifier from the given config.
func New(cfg Config) *Notifier {
	if cfg.Username == "" {
		cfg.Username = "Luminarr"
	}
	return &Notifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (n *Notifier) Name() string { return "Discord" }

// discordPayload is the JSON structure Discord's webhook API accepts.
type discordPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color"`               // decimal RGB
	Timestamp   string `json:"timestamp,omitempty"` // ISO 8601
}

// colorForEvent returns a Discord embed color (decimal RGB) for an event type.
func colorForEvent(t plugin.EventType) int {
	switch t {
	case plugin.EventGrabStarted:
		return 0x5865F2 // blurple
	case plugin.EventDownloadDone, plugin.EventImportDone:
		return 0x57F287 // green
	case plugin.EventImportFailed, plugin.EventHealthIssue:
		return 0xED4245 // red
	case plugin.EventHealthOK:
		return 0x57F287 // green
	case plugin.EventMovieAdded:
		return 0xFEE75C // yellow
	default:
		return 0x5865F2
	}
}

// Notify sends the event as a Discord embed message.
func (n *Notifier) Notify(ctx context.Context, event plugin.NotificationEvent) error {
	payload := discordPayload{
		Username:  n.cfg.Username,
		AvatarURL: n.cfg.AvatarURL,
		Embeds: []discordEmbed{
			{
				Title:       fmt.Sprintf("[Luminarr] %s", event.Type),
				Description: event.Message,
				Color:       colorForEvent(event.Type),
				Timestamp:   event.Timestamp.UTC().Format(time.RFC3339),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: sending request: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success.
	if resp.StatusCode != http.StatusNoContent && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		return fmt.Errorf("discord: server returned %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test message to verify the Discord webhook is reachable.
func (n *Notifier) Test(ctx context.Context) error {
	return n.Notify(ctx, plugin.NotificationEvent{
		Type:      plugin.EventType("test"),
		Timestamp: time.Now().UTC(),
		Message:   "Luminarr Discord test — connection successful",
	})
}
