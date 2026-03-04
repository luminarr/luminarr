// Package slack implements a Luminarr notification plugin that sends events
// to a Slack channel via an Incoming Webhook URL.
package slack

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
	registry.Default.RegisterNotifier("slack", func(settings json.RawMessage) (plugin.Notifier, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("slack: invalid settings: %w", err)
		}
		if cfg.WebhookURL == "" {
			return nil, fmt.Errorf("slack: webhook_url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterNotifierSanitizer("slack", func(settings json.RawMessage) json.RawMessage {
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

// Config holds the user-supplied settings for a Slack notifier.
type Config struct {
	WebhookURL string `json:"webhook_url"`
	Username   string `json:"username,omitempty"`   // override the bot display name
	IconEmoji  string `json:"icon_emoji,omitempty"` // e.g. ":clapper:"
}

// Notifier is a Slack notifier plugin instance.
type Notifier struct {
	cfg    Config
	client *http.Client
}

// New creates a new Notifier from the given config.
func New(cfg Config) *Notifier {
	if cfg.Username == "" {
		cfg.Username = "Luminarr"
	}
	if cfg.IconEmoji == "" {
		cfg.IconEmoji = ":clapper:"
	}
	return &Notifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (n *Notifier) Name() string { return "Slack" }

// slackPayload is the JSON structure Slack's Incoming Webhook API accepts.
type slackPayload struct {
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string `json:"color"` // hex "#RRGGBB" or "good"/"warning"/"danger"
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	Footer string `json:"footer,omitempty"`
	Ts     int64  `json:"ts,omitempty"` // Unix timestamp for the footer
}

// colorForEvent returns a Slack attachment color for an event type.
func colorForEvent(t plugin.EventType) string {
	switch t {
	case plugin.EventGrabStarted:
		return "#5865F2" // blue-ish
	case plugin.EventDownloadDone, plugin.EventImportDone, plugin.EventHealthOK:
		return "good" // Slack's built-in green
	case plugin.EventImportFailed, plugin.EventHealthIssue:
		return "danger" // Slack's built-in red
	case plugin.EventMovieAdded:
		return "warning" // Slack's built-in yellow
	default:
		return "#5865F2"
	}
}

// Notify sends the event as a Slack attachment message.
func (n *Notifier) Notify(ctx context.Context, event plugin.NotificationEvent) error {
	payload := slackPayload{
		Username:  n.cfg.Username,
		IconEmoji: n.cfg.IconEmoji,
		Attachments: []slackAttachment{
			{
				Color:  colorForEvent(event.Type),
				Title:  fmt.Sprintf("[Luminarr] %s", event.Type),
				Text:   event.Message,
				Footer: "Luminarr",
				Ts:     event.Timestamp.UTC().Unix(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack: server returned %d", resp.StatusCode)
	}
	return nil
}

// Test sends a test message to verify the Slack webhook is reachable.
func (n *Notifier) Test(ctx context.Context) error {
	return n.Notify(ctx, plugin.NotificationEvent{
		Type:      plugin.EventType("test"),
		Timestamp: time.Now().UTC(),
		Message:   "Luminarr Slack test — connection successful",
	})
}
