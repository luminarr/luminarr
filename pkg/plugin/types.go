// Package plugin defines the public interfaces and value types for
// Luminarr's plugin system. Indexer, DownloadClient, and Notifier
// implementations (built-in or external) depend only on this package.
package plugin

// Protocol identifies the release download mechanism.
type Protocol string

const (
	ProtocolTorrent Protocol = "torrent"
	ProtocolNZB     Protocol = "nzb"
	ProtocolUnknown Protocol = "unknown"
)

// Resolution is the video resolution of a release.
type Resolution string

const (
	ResolutionUnknown Resolution = "unknown"
	ResolutionSD      Resolution = "sd" // 480p and below
	Resolution720p    Resolution = "720p"
	Resolution1080p   Resolution = "1080p"
	Resolution2160p   Resolution = "2160p" // 4K
)

// Source is the origin/format of a release.
type Source string

const (
	SourceUnknown  Source = "unknown"
	SourceCAM      Source = "cam"
	SourceTELECINE Source = "telecine"
	SourceDVD      Source = "dvd"
	SourceHDTV     Source = "hdtv"
	SourceWEBRip   Source = "webrip"
	SourceWEBDL    Source = "webdl"
	SourceBluRay   Source = "bluray"
	SourceRemux    Source = "remux"
)

// Codec is the video codec of a release.
type Codec string

const (
	CodecUnknown Codec = "unknown"
	CodecX264    Codec = "x264"
	CodecX265    Codec = "x265"
	CodecAV1     Codec = "av1"
	CodecXVID    Codec = "xvid"
)

// HDRFormat is the high dynamic range format of a release.
type HDRFormat string

const (
	HDRNone        HDRFormat = "none"
	HDRUnknown     HDRFormat = "unknown"
	HDRHDR10       HDRFormat = "hdr10"
	HDRDolbyVision HDRFormat = "dolby_vision"
	HDRHLG         HDRFormat = "hlg"
	HDRHDR10Plus   HDRFormat = "hdr10plus"
)

// Quality describes the technical characteristics of a release.
// It is a value type — embedded in releases, files, and profiles.
type Quality struct {
	Resolution Resolution `json:"resolution"`
	Source     Source     `json:"source"`
	Codec      Codec      `json:"codec"`
	HDR        HDRFormat  `json:"hdr"`
	// Name is the human-readable label derived from the other fields,
	// e.g. "Bluray-1080p x265" or "WEBDL-2160p HDR10".
	Name string `json:"name"`
}

// Score returns a numeric rank for this quality used when comparing
// two qualities. Higher is better.
func (q Quality) Score() int {
	return resolutionScore(q.Resolution)*100 + sourceScore(q.Source)*10 + codecScore(q.Codec)
}

// BetterThan reports whether q is strictly better than other.
func (q Quality) BetterThan(other Quality) bool {
	return q.Score() > other.Score()
}

// AtLeast reports whether q meets or exceeds other.
func (q Quality) AtLeast(other Quality) bool {
	return q.Score() >= other.Score()
}

func resolutionScore(r Resolution) int {
	switch r {
	case Resolution2160p:
		return 4
	case Resolution1080p:
		return 3
	case Resolution720p:
		return 2
	case ResolutionSD:
		return 1
	default:
		return 0
	}
}

func sourceScore(s Source) int {
	switch s {
	case SourceRemux:
		return 7
	case SourceBluRay:
		return 6
	case SourceWEBDL:
		return 5
	case SourceWEBRip:
		return 4
	case SourceHDTV:
		return 3
	case SourceDVD:
		return 2
	case SourceTELECINE:
		return 1
	case SourceCAM:
		return 0
	default:
		return 0
	}
}

func codecScore(c Codec) int {
	switch c {
	case CodecAV1:
		return 3
	case CodecX265:
		return 2
	case CodecX264:
		return 1
	default:
		return 0
	}
}

// Release is the transient result of an indexer search.
// It is not stored in the database. A summary is written to
// GrabHistory when a release is grabbed.
type Release struct {
	GUID        string
	Title       string
	Indexer     string
	Protocol    Protocol
	DownloadURL string
	InfoURL     string
	Size        int64
	Seeds       int
	Peers       int
	AgeDays     float64
	Quality     Quality
}
