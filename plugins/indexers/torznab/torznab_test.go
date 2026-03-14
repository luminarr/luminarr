package torznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// capsXML is a minimal but realistic Torznab capabilities response.
const capsXML = `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server version="1.0" title="Prowlarr"/>
  <searching>
    <search available="yes" supportedParams="q"/>
    <tv-search available="yes" supportedParams="q,season,ep"/>
    <movie-search available="yes" supportedParams="q,tmdbid,imdbid"/>
  </searching>
  <categories>
    <category id="2000" name="Movies"/>
    <category id="2010" name="Movies/Foreign"/>
    <category id="2030" name="Movies/HD"/>
  </categories>
</caps>`

// feedXML is a realistic Torznab RSS feed with two releases.
// pubDate is set 2 days in the past to produce a measurable AgeDays value.
var feedXML = buildFeedXML()

func buildFeedXML() string {
	twoDaysAgo := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC1123Z)
	oneDayAgo := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC1123Z)

	return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <title>Prowlarr</title>
    <item>
      <title>Inception.2010.1080p.BluRay.x264-GROUP</title>
      <guid isPermaLink="false">https://indexer.example.com/release/11111</guid>
      <link>https://indexer.example.com/details/11111</link>
      <pubDate>` + twoDaysAgo + `</pubDate>
      <enclosure url="https://indexer.example.com/download/11111.torrent" length="7516192768" type="application/x-bittorrent"/>
      <torznab:attr name="category" value="2030"/>
      <torznab:attr name="seeders" value="120"/>
      <torznab:attr name="peers" value="15"/>
      <torznab:attr name="size" value="7516192768"/>
    </item>
    <item>
      <title>Inception.2010.2160p.WEBDL.x265.HDR-SCENE</title>
      <guid isPermaLink="false">https://indexer.example.com/release/22222</guid>
      <link>https://indexer.example.com/details/22222</link>
      <pubDate>` + oneDayAgo + `</pubDate>
      <enclosure url="https://indexer.example.com/download/22222.torrent" length="21474836480" type="application/x-bittorrent"/>
      <torznab:attr name="category" value="2000"/>
      <torznab:attr name="seeders" value="55"/>
      <torznab:attr name="peers" value="3"/>
      <torznab:attr name="size" value="21474836480"/>
    </item>
  </channel>
</rss>`
}

// newTestIndexer starts an httptest.Server, registers a single handler that
// calls handlerFn, and returns an Indexer pointed at that server along with
// a cleanup function.
//
// The indexer's HTTP client is replaced with a plain client that bypasses the
// SSRF-blocking safe dialer so tests can connect to 127.0.0.1.
func newTestIndexer(t *testing.T, handlerFn http.HandlerFunc) (*Indexer, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handlerFn)
	idx := New(Config{
		URL:    srv.URL,
		APIKey: "testapikey",
	})
	idx.client = &http.Client{Timeout: 30 * time.Second} // bypass safedialer for tests
	return idx, srv
}

// queryParams parses the query string of a URL string. Panics on malformed input.
func queryParams(rawURL string) url.Values {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u.Query()
}

// TestSearch_WithTMDBID verifies that Search uses the movie endpoint and passes
// tmdbid when q.TMDBID is non-zero.
func TestSearch_WithTMDBID(t *testing.T) {
	var capturedURL string

	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(feedXML))
	})
	defer srv.Close()

	releases, err := idx.Search(context.Background(), plugin.SearchQuery{
		TMDBID: 27205,
		Year:   2010,
		Query:  "Inception",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	// Verify endpoint and parameters.
	params := queryParams(srv.URL + capturedURL)
	if got := params.Get("t"); got != "movie" {
		t.Errorf("expected t=movie, got %q", got)
	}
	if got := params.Get("tmdbid"); got != "27205" {
		t.Errorf("expected tmdbid=27205, got %q", got)
	}
	if params.Has("cat") {
		t.Errorf("cat should not be set for movie search, got %q", params.Get("cat"))
	}
	if got := params.Get("apikey"); got != "testapikey" {
		t.Errorf("expected apikey=testapikey, got %q", got)
	}

	// Verify parsed releases.
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	r0 := releases[0]
	if r0.Title != "Inception.2010.1080p.BluRay.x264-GROUP" {
		t.Errorf("release[0].Title = %q", r0.Title)
	}
	if r0.GUID != "https://indexer.example.com/release/11111" {
		t.Errorf("release[0].GUID = %q", r0.GUID)
	}
	if r0.DownloadURL != "https://indexer.example.com/download/11111.torrent" {
		t.Errorf("release[0].DownloadURL = %q", r0.DownloadURL)
	}
	if r0.Protocol != plugin.ProtocolTorrent {
		t.Errorf("release[0].Protocol = %q", r0.Protocol)
	}
	if r0.Seeds != 120 {
		t.Errorf("release[0].Seeds = %d, want 120", r0.Seeds)
	}
	if r0.Peers != 15 {
		t.Errorf("release[0].Peers = %d, want 15", r0.Peers)
	}
	if r0.Size != 7516192768 {
		t.Errorf("release[0].Size = %d, want 7516192768", r0.Size)
	}
	if r0.AgeDays < 1.5 || r0.AgeDays > 2.5 {
		t.Errorf("release[0].AgeDays = %f, want ~2.0", r0.AgeDays)
	}

	r1 := releases[1]
	if r1.Seeds != 55 {
		t.Errorf("release[1].Seeds = %d, want 55", r1.Seeds)
	}
	if r1.Peers != 3 {
		t.Errorf("release[1].Peers = %d, want 3", r1.Peers)
	}
	if r1.AgeDays < 0.5 || r1.AgeDays > 1.5 {
		t.Errorf("release[1].AgeDays = %f, want ~1.0", r1.AgeDays)
	}
}

// TestSearch_TMDBFallback verifies that when TMDBID is set but the movie
// search endpoint returns zero results, Search falls back to the text search
// endpoint (?t=search&q=title+year).
func TestSearch_TMDBFallback(t *testing.T) {
	var capturedURLs []string

	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURLs = append(capturedURLs, r.URL.String())
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		// Return an empty feed for the first (movie) request; return results
		// for the second (search) request so we can verify the fallback.
		if len(capturedURLs) == 1 {
			_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`))
		} else {
			_, _ = w.Write([]byte(feedXML))
		}
	})
	defer srv.Close()

	releases, err := idx.Search(context.Background(), plugin.SearchQuery{
		TMDBID: 27205,
		Query:  "Inception",
		Year:   2010,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(capturedURLs) != 2 {
		t.Fatalf("expected 2 requests (movie then search), got %d", len(capturedURLs))
	}

	// First request must use the movie endpoint.
	p1 := queryParams(srv.URL + capturedURLs[0])
	if got := p1.Get("t"); got != "movie" {
		t.Errorf("first request: expected t=movie, got %q", got)
	}

	// Second request must use the search endpoint with title+year.
	p2 := queryParams(srv.URL + capturedURLs[1])
	if got := p2.Get("t"); got != "search" {
		t.Errorf("second request: expected t=search, got %q", got)
	}
	q := p2.Get("q")
	if !strings.Contains(q, "Inception") || !strings.Contains(q, "2010") {
		t.Errorf("fallback q param %q does not contain title and year", q)
	}

	if len(releases) != 2 {
		t.Fatalf("expected 2 releases from fallback, got %d", len(releases))
	}
}

// TestSearch_WithoutTMDBID verifies that Search falls back to the text search
// endpoint when TMDBID is 0, and builds the query from Query + Year.
func TestSearch_WithoutTMDBID(t *testing.T) {
	var capturedURL string

	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(feedXML))
	})
	defer srv.Close()

	releases, err := idx.Search(context.Background(), plugin.SearchQuery{
		Query: "Inception",
		Year:  2010,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	params := queryParams(srv.URL + capturedURL)
	if got := params.Get("t"); got != "search" {
		t.Errorf("expected t=search, got %q", got)
	}
	q := params.Get("q")
	if !strings.Contains(q, "Inception") || !strings.Contains(q, "2010") {
		t.Errorf("q param %q does not contain title and year", q)
	}
	if params.Has("cat") {
		t.Errorf("cat should not be set for text search, got %q", params.Get("cat"))
	}
	// TMDBID must not be present.
	if params.Has("tmdbid") {
		t.Errorf("tmdbid should not be present in fallback search, got %q", params.Get("tmdbid"))
	}

	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
}

// TestGetRecent verifies that GetRecent calls the movie endpoint without a
// query parameter and returns parsed releases.
func TestGetRecent(t *testing.T) {
	var capturedURL string

	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(feedXML))
	})
	defer srv.Close()

	releases, err := idx.GetRecent(context.Background())
	if err != nil {
		t.Fatalf("GetRecent returned error: %v", err)
	}

	params := queryParams(srv.URL + capturedURL)
	if got := params.Get("t"); got != "movie" {
		t.Errorf("expected t=movie, got %q", got)
	}
	// No query param should be set for recent.
	if params.Has("q") {
		t.Errorf("q should not be present for GetRecent")
	}
	if params.Has("tmdbid") {
		t.Errorf("tmdbid should not be present for GetRecent")
	}
	if params.Has("cat") {
		t.Errorf("cat should not be set for GetRecent, got %q", params.Get("cat"))
	}

	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
}

// TestTest_Success verifies that Test returns nil when the server responds 200.
func TestTest_Success(t *testing.T) {
	var capturedURL string

	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(capsXML))
	})
	defer srv.Close()

	if err := idx.Test(context.Background()); err != nil {
		t.Fatalf("Test returned unexpected error: %v", err)
	}

	params := queryParams(srv.URL + capturedURL)
	if got := params.Get("t"); got != "caps" {
		t.Errorf("Test must call t=caps, got %q", got)
	}
}

// TestTest_Failure verifies that Test returns a non-nil error when the server
// responds with a non-200 status code.
func TestTest_Failure(t *testing.T) {
	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	defer srv.Close()

	err := idx.Test(context.Background())
	if err == nil {
		t.Fatal("expected Test to return an error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error message should mention status code, got: %v", err)
	}
}

// TestGUIDFallback verifies that when guid is empty, the release GUID falls
// back to the <link> element.
func TestGUIDFallback(t *testing.T) {
	// Feed where the guid element has no text content.
	noGUIDFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <item>
      <title>Some.Movie.2020.720p.WEBRip-GROUP</title>
      <guid isPermaLink="false"></guid>
      <link>https://indexer.example.com/details/99999</link>
      <pubDate>Mon, 01 Jan 2024 00:00:00 +0000</pubDate>
      <enclosure url="https://indexer.example.com/download/99999.torrent" length="1073741824" type="application/x-bittorrent"/>
      <torznab:attr name="seeders" value="10"/>
      <torznab:attr name="peers" value="2"/>
    </item>
  </channel>
</rss>`

	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(noGUIDFeed))
	})
	defer srv.Close()

	releases, err := idx.GetRecent(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	if releases[0].GUID != "https://indexer.example.com/details/99999" {
		t.Errorf("GUID fallback failed: got %q", releases[0].GUID)
	}
}

// TestCapabilities verifies that Capabilities parses the caps XML and
// extracts search availability flags and category IDs correctly.
func TestCapabilities(t *testing.T) {
	idx, srv := newTestIndexer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(capsXML))
	})
	defer srv.Close()

	caps, err := idx.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities returned error: %v", err)
	}
	if !caps.SearchAvailable {
		t.Error("SearchAvailable should be true")
	}
	if !caps.TVSearchAvailable {
		t.Error("TVSearchAvailable should be true")
	}
	if !caps.MovieSearch {
		t.Error("MovieSearch should be true")
	}
	if len(caps.Categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(caps.Categories))
	}
}

// TestToRelease_IndexerFlags verifies that indexer flags are correctly parsed
// from torznab:attr elements (downloadvolumefactor, uploadvolumefactor, tag).
func TestToRelease_IndexerFlags(t *testing.T) {
	idx := New(Config{URL: "http://localhost"})

	tests := []struct {
		name  string
		attrs []torznabAttr
		want  []plugin.IndexerFlag
	}{
		{
			name:  "no flags",
			attrs: []torznabAttr{{Name: "seeders", Value: "10"}},
			want:  nil,
		},
		{
			name:  "freeleech via downloadvolumefactor=0",
			attrs: []torznabAttr{{Name: "downloadvolumefactor", Value: "0"}},
			want:  []plugin.IndexerFlag{plugin.FlagFreeleech},
		},
		{
			name:  "halfleech via downloadvolumefactor=0.5",
			attrs: []torznabAttr{{Name: "downloadvolumefactor", Value: "0.5"}},
			want:  []plugin.IndexerFlag{plugin.FlagHalfleech},
		},
		{
			name:  "freeleech_25 via downloadvolumefactor=0.75",
			attrs: []torznabAttr{{Name: "downloadvolumefactor", Value: "0.75"}},
			want:  []plugin.IndexerFlag{plugin.FlagFreeleech25},
		},
		{
			name:  "freeleech_75 via downloadvolumefactor=0.25",
			attrs: []torznabAttr{{Name: "downloadvolumefactor", Value: "0.25"}},
			want:  []plugin.IndexerFlag{plugin.FlagFreeleech75},
		},
		{
			name:  "double_upload via uploadvolumefactor=2",
			attrs: []torznabAttr{{Name: "uploadvolumefactor", Value: "2"}},
			want:  []plugin.IndexerFlag{plugin.FlagDoubleUpload},
		},
		{
			name:  "tag freeleech case insensitive",
			attrs: []torznabAttr{{Name: "tag", Value: "Freeleech"}},
			want:  []plugin.IndexerFlag{plugin.FlagFreeleech},
		},
		{
			name:  "tag internal",
			attrs: []torznabAttr{{Name: "tag", Value: "Internal"}},
			want:  []plugin.IndexerFlag{plugin.FlagInternal},
		},
		{
			name:  "tag scene uppercase",
			attrs: []torznabAttr{{Name: "tag", Value: "SCENE"}},
			want:  []plugin.IndexerFlag{plugin.FlagScene},
		},
		{
			name:  "tag nuked",
			attrs: []torznabAttr{{Name: "tag", Value: "nuked"}},
			want:  []plugin.IndexerFlag{plugin.FlagNuked},
		},
		{
			name: "multiple flags combined",
			attrs: []torznabAttr{
				{Name: "downloadvolumefactor", Value: "0"},
				{Name: "uploadvolumefactor", Value: "2"},
				{Name: "tag", Value: "Internal"},
			},
			want: []plugin.IndexerFlag{plugin.FlagFreeleech, plugin.FlagDoubleUpload, plugin.FlagInternal},
		},
		{
			name: "duplicate freeleech from tag and factor deduplicated",
			attrs: []torznabAttr{
				{Name: "downloadvolumefactor", Value: "0"},
				{Name: "tag", Value: "freeleech"},
			},
			want: []plugin.IndexerFlag{plugin.FlagFreeleech},
		},
		{
			name:  "uploadvolumefactor=1 produces no flag",
			attrs: []torznabAttr{{Name: "uploadvolumefactor", Value: "1"}},
			want:  nil,
		},
		{
			name:  "downloadvolumefactor=1 produces no flag",
			attrs: []torznabAttr{{Name: "downloadvolumefactor", Value: "1"}},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := rssItem{
				Title: "Test.Release.2024.1080p",
				Attrs: tt.attrs,
			}
			r := idx.toRelease(item)

			if len(r.IndexerFlags) != len(tt.want) {
				t.Fatalf("got %d flags %v, want %d flags %v", len(r.IndexerFlags), r.IndexerFlags, len(tt.want), tt.want)
			}
			for i, f := range r.IndexerFlags {
				if f != tt.want[i] {
					t.Errorf("flag[%d] = %q, want %q", i, f, tt.want[i])
				}
			}
		})
	}
}

// TestParseAgeDays is a unit test for the internal pubDate parsing helper.
func TestParseAgeDays(t *testing.T) {
	cases := []struct {
		name    string
		pubDate string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "two days ago RFC1123Z",
			pubDate: time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC1123Z),
			wantMin: 1.9,
			wantMax: 2.1,
		},
		{
			name:    "empty string",
			pubDate: "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "invalid format",
			pubDate: "not-a-date",
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAgeDays(tc.pubDate)
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("parseAgeDays(%q) = %f, want [%f, %f]", tc.pubDate, got, tc.wantMin, tc.wantMax)
			}
		})
	}
}
