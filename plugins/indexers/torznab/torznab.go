// Package torznab implements the Torznab indexer plugin for Luminarr.
// Torznab is an RSS/Atom-style XML protocol used by Prowlarr and Jackett
// to expose torrent indexers over a common API.
package torznab

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/safedialer"
	"github.com/luminarr/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterIndexer("torznab", func(settings json.RawMessage) (plugin.Indexer, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("torznab: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("torznab: url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterIndexerSanitizer("torznab", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["api_key"]; ok {
			m["api_key"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the user-supplied settings for a Torznab indexer instance.
type Config struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key,omitempty"`
}

// Indexer is a Torznab indexer plugin instance.
type Indexer struct {
	cfg    Config
	client *http.Client
}

// New creates a new Torznab Indexer from the given config.
func New(cfg Config) *Indexer {
	return &Indexer{
		cfg: cfg,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: safedialer.LANTransport(),
		},
	}
}

// Name returns the human-readable plugin name.
func (idx *Indexer) Name() string { return "Torznab" }

// Protocol returns the protocol this indexer serves.
func (idx *Indexer) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

// Capabilities fetches and parses the indexer's capabilities document.
func (idx *Indexer) Capabilities(ctx context.Context) (plugin.Capabilities, error) {
	u := idx.buildURL("caps", url.Values{})
	resp, err := idx.get(ctx, u)
	if err != nil {
		return plugin.Capabilities{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return plugin.Capabilities{}, fmt.Errorf("torznab: capabilities returned HTTP %d", resp.StatusCode)
	}

	var caps capsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&caps); err != nil {
		return plugin.Capabilities{}, fmt.Errorf("torznab: decoding capabilities: %w", err)
	}

	categories := make([]int, 0, len(caps.Categories.Items))
	for _, c := range caps.Categories.Items {
		categories = append(categories, c.ID)
	}

	return plugin.Capabilities{
		SearchAvailable:   caps.Searching.Search.Available == "yes",
		TVSearchAvailable: caps.Searching.TVSearch.Available == "yes",
		MovieSearch:       caps.Searching.MovieSearch.Available == "yes",
		Categories:        categories,
	}, nil
}

// Search queries the indexer for releases matching q.
//
// Strategy:
//  1. If q.TMDBID is set, try the movie search endpoint (?t=movie&tmdbid=…).
//     Not all indexers or Prowlarr-backed indexers support this endpoint, so
//     if it returns zero results we fall through to step 2.
//  2. Fall back to free-text search (?t=search&q=title+year) which is
//     universally supported.
func (idx *Indexer) Search(ctx context.Context, q plugin.SearchQuery) ([]plugin.Release, error) {
	// 1. TMDB movie search — most specific, not universally supported.
	if q.TMDBID != 0 {
		params := url.Values{}
		params.Set("tmdbid", strconv.Itoa(q.TMDBID))
		u := idx.buildURL("movie", params)
		releases, err := idx.fetchReleases(ctx, u)
		if err == nil && len(releases) > 0 {
			return releases, nil
		}
		// err is non-nil or zero results — fall through to text search.
	}

	// 2. Free-text search — universally supported. No category filter: many
	// indexers return empty when cat=2000 is specified, and the title+year
	// query is specific enough without it.
	query := q.Query
	if q.Year != 0 {
		query = fmt.Sprintf("%s %d", query, q.Year)
	}
	params := url.Values{}
	params.Set("q", query)
	u := idx.buildURL("search", params)
	return idx.fetchReleases(ctx, u)
}

// GetRecent returns the most recent releases from the indexer's RSS feed.
func (idx *Indexer) GetRecent(ctx context.Context) ([]plugin.Release, error) {
	u := idx.buildURL("movie", url.Values{})
	return idx.fetchReleases(ctx, u)
}

// Test checks that the indexer is reachable and returns a valid capabilities
// response. Returns a non-nil error if connectivity or configuration fails.
func (idx *Indexer) Test(ctx context.Context) error {
	u := idx.buildURL("caps", url.Values{})
	resp, err := idx.get(ctx, u)
	if err != nil {
		return fmt.Errorf("torznab: test request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("torznab: test returned HTTP %d", resp.StatusCode)
	}

	var caps capsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&caps); err != nil {
		// EOF means the body was empty — the indexer answered HTTP 200 but
		// returned no caps document. This is acceptable: some Prowlarr-backed
		// indexers omit caps while still serving search results. Only reject
		// responses that are clearly not XML at all (e.g. a JSON REST API).
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("torznab: invalid caps response (wrong URL?): %w", err)
		}
	}
	return nil
}

// buildURL constructs the full API URL for the given Torznab function and
// additional query parameters. The apikey is appended when configured.
// Trailing slashes on the configured URL are stripped so callers can enter
// either "http://host/5" or "http://host/5/" and both produce valid URLs.
func (idx *Indexer) buildURL(t string, params url.Values) string {
	params.Set("t", t)
	if idx.cfg.APIKey != "" {
		params.Set("apikey", idx.cfg.APIKey)
	}
	base := strings.TrimRight(idx.cfg.URL, "/")
	return base + "/api?" + params.Encode()
}

// get executes an HTTP GET against the given URL using the context for
// cancellation and deadline propagation.
func (idx *Indexer) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("torznab: building request: %w", err)
	}
	return idx.client.Do(req)
}

// fetchReleases performs a GET to rawURL, decodes the Torznab RSS response,
// and returns the parsed releases.
func (idx *Indexer) fetchReleases(ctx context.Context, rawURL string) ([]plugin.Release, error) {
	resp, err := idx.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("torznab: search returned HTTP %d", resp.StatusCode)
	}

	var feed rssResponse
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("torznab: empty response body — check the API key and indexer URL")
		}
		return nil, fmt.Errorf("torznab: decoding response: %w", err)
	}

	releases := make([]plugin.Release, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		releases = append(releases, idx.toRelease(item))
	}
	return releases, nil
}

// toRelease converts a parsed RSS item into a plugin.Release.
func (idx *Indexer) toRelease(item rssItem) plugin.Release {
	guid := item.GUID.Value
	if guid == "" {
		guid = item.Link
	}

	r := plugin.Release{
		GUID:        guid,
		Title:       item.Title,
		Indexer:     item.JackettIndexer, // Prowlarr fills this; empty for standalone indexers.
		Protocol:    plugin.ProtocolTorrent,
		DownloadURL: item.Enclosure.URL,
		InfoURL:     item.Link,
		Size:        item.Enclosure.Length,
	}

	// Extract torznab-specific attributes.
	for _, attr := range item.Attrs {
		switch attr.Name {
		case "seeders":
			r.Seeds, _ = strconv.Atoi(attr.Value)
		case "peers":
			r.Peers, _ = strconv.Atoi(attr.Value)
		case "size":
			// Prefer the torznab size attribute over enclosure length when present.
			if sz, err := strconv.ParseInt(attr.Value, 10, 64); err == nil && sz > 0 {
				r.Size = sz
			}
		}
	}

	r.AgeDays = parseAgeDays(item.PubDate)
	return r
}

// parseAgeDays parses an RFC1123Z pubDate string and returns the number of
// calendar days elapsed since that time. Returns 0 on parse failure.
func parseAgeDays(pubDate string) float64 {
	if pubDate == "" {
		return 0
	}
	t, err := time.Parse(time.RFC1123Z, pubDate)
	if err != nil {
		// Try RFC1123 as a fallback (some indexers omit the timezone offset).
		t, err = time.Parse(time.RFC1123, pubDate)
		if err != nil {
			return 0
		}
	}
	return time.Since(t).Hours() / 24
}

// ---------------------------------------------------------------------------
// XML types
// ---------------------------------------------------------------------------

// rssResponse is the top-level RSS document returned by Torznab.
type rssResponse struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

// rssItem represents a single release entry in the Torznab RSS feed.
type rssItem struct {
	Title     string    `xml:"title"`
	GUID      rssGUID   `xml:"guid"`
	Link      string    `xml:"link"`
	PubDate   string    `xml:"pubDate"`
	Enclosure enclosure `xml:"enclosure"`
	// Prowlarr/Jackett include the upstream indexer name per item.
	JackettIndexer string `xml:"jackettindexer"`
	// Torznab attributes use the namespace http://torznab.com/schemas/2015/feed.
	Attrs []torznabAttr `xml:"http://torznab.com/schemas/2015/feed attr"`
}

// rssGUID holds the guid element, which may carry an isPermaLink attribute.
type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// enclosure is the RSS enclosure element that carries the download URL and size.
type enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// torznabAttr is a <torznab:attr name="..." value="..."/> element.
type torznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// capsResponse is the top-level capabilities document.
type capsResponse struct {
	XMLName    xml.Name       `xml:"caps"`
	Searching  capsSearching  `xml:"searching"`
	Categories capsCategories `xml:"categories"`
}

type capsSearching struct {
	Search      capsSearchType `xml:"search"`
	TVSearch    capsSearchType `xml:"tv-search"`
	MovieSearch capsSearchType `xml:"movie-search"`
}

type capsSearchType struct {
	Available string `xml:"available,attr"`
}

type capsCategories struct {
	Items []capsCategory `xml:"category"`
}

type capsCategory struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
}
