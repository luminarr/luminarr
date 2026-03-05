// Package transmission implements the plugin.DownloadClient interface for the
// Transmission RPC API. Tested against Transmission 3.x and 4.x.
package transmission

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/safedialer"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterDownloader("transmission", func(s json.RawMessage) (plugin.DownloadClient, error) {
		var cfg Config
		if err := json.Unmarshal(s, &cfg); err != nil {
			return nil, fmt.Errorf("transmission: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, errors.New("transmission: url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterDownloaderSanitizer("transmission", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["password"]; ok {
			m["password"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the connection settings for a Transmission instance.
type Config struct {
	URL      string `json:"url"` // e.g. "http://localhost:9091"
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Client implements plugin.DownloadClient against the Transmission RPC API.
type Client struct {
	cfg       Config
	http      *http.Client
	mu        sync.Mutex
	sessionID string // X-Transmission-Session-Id CSRF token
	rpcURL    string // computed once: cfg.URL + "/transmission/rpc"
}

// New creates a new Transmission client.
func New(cfg Config) *Client {
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
		rpcURL: strings.TrimRight(cfg.URL, "/") + "/transmission/rpc",
	}
}

// NewWithHTTPClient creates a Client with a caller-supplied http.Client.
// Intended for unit tests that need to bypass the safe dialer.
func NewWithHTTPClient(cfg Config, client *http.Client) *Client {
	return &Client{
		cfg:    cfg,
		http:   client,
		rpcURL: strings.TrimRight(cfg.URL, "/") + "/transmission/rpc",
	}
}

func (c *Client) Name() string              { return "Transmission" }
func (c *Client) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

// Test verifies connectivity and authentication by fetching the session info.
func (c *Client) Test(ctx context.Context) error {
	var result json.RawMessage
	if err := c.rpc(ctx, "session-get", nil, &result); err != nil {
		return fmt.Errorf("transmission: connectivity check failed: %w", err)
	}
	return nil
}

// Add submits a torrent to Transmission. Magnet links and .torrent URLs are
// both supported. Returns the torrent's hashString as the client item ID.
func (c *Client) Add(ctx context.Context, r plugin.Release) (string, error) {
	args := map[string]any{}

	if strings.HasPrefix(r.DownloadURL, "magnet:") {
		args["filename"] = r.DownloadURL
	} else {
		// Fetch the .torrent file and send as base64-encoded metainfo.
		data, err := c.fetchTorrentFile(ctx, r.DownloadURL)
		if err != nil {
			// Handle magnet redirect from indexers like Prowlarr.
			var mag errMagnetRedirect
			if errors.As(err, &mag) {
				args["filename"] = mag.magnetURL
			} else {
				return "", fmt.Errorf("transmission: downloading torrent file: %w", err)
			}
		}
		if _, ok := args["filename"]; !ok {
			args["metainfo"] = base64.StdEncoding.EncodeToString(data)
		}
	}

	var resp addResponse
	if err := c.rpc(ctx, "torrent-add", args, &resp); err != nil {
		return "", fmt.Errorf("transmission: torrent-add: %w", err)
	}

	added := resp.TorrentAdded
	if added == nil {
		added = resp.TorrentDuplicate
	}
	if added == nil {
		return "", errors.New("transmission: torrent-add returned no torrent info")
	}
	return strings.ToLower(added.HashString), nil
}

// Status returns the current state of a single torrent by its hash string.
func (c *Client) Status(ctx context.Context, clientItemID string) (plugin.QueueItem, error) {
	torrents, err := c.getTorrents(ctx, []string{clientItemID})
	if err != nil {
		return plugin.QueueItem{}, err
	}
	if len(torrents) == 0 {
		return plugin.QueueItem{}, fmt.Errorf("transmission: torrent %q not found", clientItemID)
	}
	return torrents[0].toQueueItem(), nil
}

// GetQueue returns all torrents currently tracked by Transmission.
func (c *Client) GetQueue(ctx context.Context) ([]plugin.QueueItem, error) {
	torrents, err := c.getTorrents(ctx, nil)
	if err != nil {
		return nil, err
	}
	items := make([]plugin.QueueItem, len(torrents))
	for i, t := range torrents {
		items[i] = t.toQueueItem()
	}
	return items, nil
}

// Remove deletes a torrent from Transmission. If deleteFiles is true the
// downloaded data is also removed from disk.
func (c *Client) Remove(ctx context.Context, clientItemID string, deleteFiles bool) error {
	// torrent-remove accepts integer IDs, so we first look up the torrent by
	// hash to get its numeric ID.
	torrents, err := c.getTorrents(ctx, []string{clientItemID})
	if err != nil {
		return err
	}
	if len(torrents) == 0 {
		return fmt.Errorf("transmission: torrent %q not found for removal", clientItemID)
	}

	args := map[string]any{
		"ids":               []int64{torrents[0].ID},
		"delete-local-data": deleteFiles,
	}
	var result json.RawMessage
	if err := c.rpc(ctx, "torrent-remove", args, &result); err != nil {
		return fmt.Errorf("transmission: torrent-remove: %w", err)
	}
	return nil
}

// ── RPC helper ───────────────────────────────────────────────────────────────

// rpc performs a Transmission RPC call, automatically handling the session-ID
// CSRF dance (409 → retry with new X-Transmission-Session-Id header).
func (c *Client) rpc(ctx context.Context, method string, arguments map[string]any, out any) error {
	reqBody := rpcRequest{Method: method, Arguments: arguments}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, bytes.NewReader(body))
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Content-Type", "application/json")

		c.mu.Lock()
		sid := c.sessionID
		c.mu.Unlock()
		if sid != "" {
			req.Header.Set("X-Transmission-Session-Id", sid)
		}
		if c.cfg.Username != "" {
			req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
		}

		resp, doErr := c.http.Do(req)
		if doErr != nil {
			return doErr
		}

		if resp.StatusCode == http.StatusConflict {
			// 409 — extract new session ID and retry.
			newSID := resp.Header.Get("X-Transmission-Session-Id")
			resp.Body.Close()
			if newSID == "" {
				return errors.New("transmission: 409 without session ID header")
			}
			c.mu.Lock()
			c.sessionID = newSID
			c.mu.Unlock()
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			return errors.New("transmission: authentication failed — check username and password")
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("transmission: unexpected HTTP %d", resp.StatusCode)
		}

		var rpcResp rpcResponse
		if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
			return fmt.Errorf("transmission: decoding response: %w", err)
		}
		if rpcResp.Result != "success" {
			return fmt.Errorf("transmission: RPC error: %s", rpcResp.Result)
		}
		if out != nil && rpcResp.Arguments != nil {
			if err := json.Unmarshal(rpcResp.Arguments, out); err != nil {
				return fmt.Errorf("transmission: decoding arguments: %w", err)
			}
		}
		return nil
	}
	return errors.New("transmission: too many 409 retries")
}

// ── Torrent fetch ────────────────────────────────────────────────────────────

// errMagnetRedirect signals that a torrent URL redirected to a magnet link.
type errMagnetRedirect struct{ magnetURL string }

func (e errMagnetRedirect) Error() string { return "magnet redirect: " + e.magnetURL }

func (c *Client) fetchTorrentFile(ctx context.Context, torrentURL string) ([]byte, error) {
	client := &http.Client{
		Transport: safedialer.LANTransport(),
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if strings.HasPrefix(req.URL.String(), "magnet:") {
				return errMagnetRedirect{magnetURL: req.URL.String()}
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			var mag errMagnetRedirect
			if errors.As(urlErr.Err, &mag) {
				return nil, mag
			}
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching torrent", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB limit
}

// ── Torrent get helper ───────────────────────────────────────────────────────

// torrentFields is the list of fields requested from torrent-get.
var torrentFields = []string{
	"id", "hashString", "name", "status", "percentDone",
	"sizeWhenDone", "leftUntilDone", "downloadDir",
	"error", "errorString", "uploadRatio", "addedDate",
}

func (c *Client) getTorrents(ctx context.Context, hashes []string) ([]torrentInfo, error) {
	args := map[string]any{"fields": torrentFields}
	if len(hashes) > 0 {
		args["ids"] = hashes
	}

	var resp getResponse
	if err := c.rpc(ctx, "torrent-get", args, &resp); err != nil {
		return nil, fmt.Errorf("transmission: torrent-get: %w", err)
	}
	return resp.Torrents, nil
}

// ── RPC types ────────────────────────────────────────────────────────────────

type rpcRequest struct {
	Method    string         `json:"method"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type rpcResponse struct {
	Result    string          `json:"result"`
	Arguments json.RawMessage `json:"arguments"`
}

type getResponse struct {
	Torrents []torrentInfo `json:"torrents"`
}

type addResponse struct {
	TorrentAdded     *addedTorrent `json:"torrent-added"`
	TorrentDuplicate *addedTorrent `json:"torrent-duplicate"`
}

type addedTorrent struct {
	ID         int64  `json:"id"`
	HashString string `json:"hashString"`
	Name       string `json:"name"`
}

// ── Torrent state types ──────────────────────────────────────────────────────

type torrentInfo struct {
	ID            int64   `json:"id"`
	HashString    string  `json:"hashString"`
	Name          string  `json:"name"`
	Status        int     `json:"status"`
	PercentDone   float64 `json:"percentDone"`
	SizeWhenDone  int64   `json:"sizeWhenDone"`
	LeftUntilDone int64   `json:"leftUntilDone"`
	DownloadDir   string  `json:"downloadDir"`
	Error         int     `json:"error"`
	ErrorString   string  `json:"errorString"`
	UploadRatio   float64 `json:"uploadRatio"`
	AddedDate     int64   `json:"addedDate"`
}

func (t torrentInfo) toQueueItem() plugin.QueueItem {
	contentPath := ""
	if t.DownloadDir != "" && t.Name != "" {
		contentPath = strings.TrimRight(t.DownloadDir, "/") + "/" + t.Name
	}

	status := mapStatus(t.Status)
	errMsg := ""
	// error > 0 means tracker warning (1), tracker error (2), or local error (3)
	if t.Error > 0 {
		status = plugin.StatusFailed
		errMsg = t.ErrorString
	}

	return plugin.QueueItem{
		ClientItemID: strings.ToLower(t.HashString),
		Title:        t.Name,
		Status:       status,
		Size:         t.SizeWhenDone,
		Downloaded:   t.SizeWhenDone - t.LeftUntilDone,
		SeedRatio:    t.UploadRatio,
		Error:        errMsg,
		ContentPath:  contentPath,
		AddedAt:      t.AddedDate,
	}
}

// Transmission status integer constants.
const (
	trStopped        = 0
	trQueuedToVerify = 1
	trVerifying      = 2
	trQueuedToDown   = 3
	trDownloading    = 4
	trQueuedToSeed   = 5
	trSeeding        = 6
)

// mapStatus converts Transmission's integer status to plugin.DownloadStatus.
func mapStatus(status int) plugin.DownloadStatus {
	switch status {
	case trStopped:
		return plugin.StatusPaused
	case trQueuedToVerify, trVerifying, trQueuedToDown:
		return plugin.StatusQueued
	case trDownloading:
		return plugin.StatusDownloading
	case trQueuedToSeed, trSeeding:
		return plugin.StatusCompleted
	default:
		return plugin.StatusQueued
	}
}
