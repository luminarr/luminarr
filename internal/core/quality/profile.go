package quality

import (
	"strings"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// Profile defines a quality policy for a monitored movie.
// It controls which releases are acceptable and when upgrades are triggered.
type Profile struct {
	// ID is a stable identifier (e.g. a UUID or slug).
	ID string
	// Name is the human-readable label shown in the UI.
	Name string
	// Cutoff is the minimum quality that satisfies this profile.
	// Once a file at or above the cutoff is on disk, the movie is considered
	// "met" and no further grabs are triggered — unless upgrading is enabled.
	Cutoff plugin.Quality
	// Qualities lists every quality this profile will accept, ordered from
	// highest-preferred to lowest-preferred. Releases not in this list are
	// rejected regardless of other settings.
	Qualities []plugin.Quality
	// UpgradeAllowed, when true, permits grabbing a release that is better
	// than the current file even after the cutoff is met.
	UpgradeAllowed bool
	// UpgradeUntil, when non-nil, caps upgrades: once the current file meets
	// or exceeds this quality, no further upgrades are triggered.
	// Nil means "upgrade without limit" (subject to UpgradeAllowed).
	UpgradeUntil *plugin.Quality
	// MinCustomFormatScore is the minimum CF score a release must reach to be
	// considered acceptable. Releases below this threshold are rejected.
	MinCustomFormatScore int
	// UpgradeUntilCFScore caps CF-driven upgrades: once the current file's CF
	// score meets or exceeds this value (and quality cutoff is met), no further
	// upgrades are triggered.
	UpgradeUntilCFScore int
}

// WantRelease reports whether this profile should grab a release with
// releaseQuality, given the quality of the file already on disk
// (currentFileQuality). Pass nil for currentFileQuality when no file exists.
//
// Decision logic:
//  1. The release quality must be in the profile's allowed set.
//  2. If no file exists, grab it.
//  3. If the current file is below the cutoff, grab anything allowed that is
//     at least as good as what we have (or better).
//  4. If the current file meets/exceeds the cutoff and upgrading is disabled,
//     do not grab.
//  5. If upgrading is enabled, grab if the release is a strict upgrade.
func (p *Profile) WantRelease(releaseQuality plugin.Quality, currentFileQuality *plugin.Quality) bool {
	if !p.isAllowed(releaseQuality) {
		return false
	}

	// No file on disk — grab anything allowed.
	if currentFileQuality == nil {
		return true
	}

	current := *currentFileQuality

	// Below cutoff: keep trying to improve.
	if !current.AtLeast(p.Cutoff) {
		return releaseQuality.AtLeast(current)
	}

	// Cutoff is met — only grab if upgrading is permitted and worthwhile.
	if !p.UpgradeAllowed {
		return false
	}

	return p.IsUpgrade(releaseQuality, current)
}

// RejectReason returns a typed reason string explaining why WantRelease would
// return false. Returns "" if the release would be wanted (no rejection).
// The reason strings match autosearch.SkipReason constants.
func (p *Profile) RejectReason(releaseQuality plugin.Quality, currentFileQuality *plugin.Quality) string {
	if !p.isAllowed(releaseQuality) {
		return "quality_not_in_profile"
	}
	if currentFileQuality == nil {
		return "" // no file → want anything allowed
	}
	current := *currentFileQuality
	if !current.AtLeast(p.Cutoff) {
		if !releaseQuality.AtLeast(current) {
			return "no_upgrade_needed"
		}
		return ""
	}
	// Cutoff met.
	if !p.UpgradeAllowed {
		return "upgrade_disabled"
	}
	if !p.IsUpgrade(releaseQuality, current) {
		return "no_upgrade_needed"
	}
	return ""
}

// IsUpgrade reports whether releaseQuality is a strict improvement over
// currentQuality, subject to the UpgradeUntil ceiling defined in this profile.
func (p *Profile) IsUpgrade(releaseQuality plugin.Quality, currentQuality plugin.Quality) bool {
	if !releaseQuality.BetterThan(currentQuality) {
		return false
	}

	// If an upgrade ceiling is set and the current file already meets or
	// exceeds it, do not upgrade further.
	if p.UpgradeUntil != nil && currentQuality.AtLeast(*p.UpgradeUntil) {
		return false
	}

	// If the release itself exceeds the ceiling, cap the effective target —
	// but we still want it because it gets us to (or past) the ceiling and
	// we don't have a file there yet.
	// In practice: if the release is better than current and current is below
	// the ceiling, it's a valid upgrade regardless of how far past the ceiling
	// the release goes.
	return true
}

// ScoreWithBreakdown scores q against this profile's cutoff and returns both
// the aggregate 0–100 score and a per-dimension breakdown.
//
// Weights:  Resolution 40 pts, Source 30 pts, Codec 20 pts, HDR 10 pts.
//
// A dimension is "matched" when the release meets or exceeds the cutoff value
// for that dimension. If the cutoff field is empty/unknown, any value matches.
// The score for each dimension is awarded in full when matched, zero otherwise.
func (p *Profile) ScoreWithBreakdown(q plugin.Quality) (int, plugin.ScoreBreakdown) {
	res := scoreDimension("resolution",
		string(q.Resolution), string(p.Cutoff.Resolution), 40,
		resolutionRank(q.Resolution) >= resolutionRank(p.Cutoff.Resolution))

	src := scoreDimension("source",
		string(q.Source), string(p.Cutoff.Source), 30,
		sourceRank(q.Source) >= sourceRank(p.Cutoff.Source))

	// Codec: empty or "unknown" cutoff means "any" — always matches.
	codecWant := string(p.Cutoff.Codec)
	codecMatched := p.Cutoff.Codec == "" ||
		strings.EqualFold(codecWant, "unknown") ||
		codecRank(q.Codec) >= codecRank(p.Cutoff.Codec)
	cod := scoreDimension("codec", string(q.Codec), codecWant, 20, codecMatched)

	// HDR: empty or "none" cutoff means "any" — always matches.
	hdrWant := string(p.Cutoff.HDR)
	hdrMatched := p.Cutoff.HDR == "" ||
		p.Cutoff.HDR == plugin.HDRNone ||
		strings.EqualFold(hdrWant, "unknown") ||
		string(q.HDR) == hdrWant
	hdr := scoreDimension("hdr", string(q.HDR), hdrWant, 10, hdrMatched)

	total := res.Score + src.Score + cod.Score + hdr.Score
	bd := plugin.ScoreBreakdown{
		Total:      total,
		Dimensions: []plugin.ScoreDimension{res, src, cod, hdr},
	}
	return total, bd
}

func scoreDimension(name, got, want string, max int, matched bool) plugin.ScoreDimension {
	score := 0
	if matched {
		score = max
	}
	return plugin.ScoreDimension{
		Name:    name,
		Score:   score,
		Max:     max,
		Matched: matched,
		Got:     got,
		Want:    want,
	}
}

// resolutionRank maps a Resolution to a comparable integer (higher = better).
func resolutionRank(r plugin.Resolution) int {
	switch r {
	case plugin.Resolution2160p:
		return 4
	case plugin.Resolution1080p:
		return 3
	case plugin.Resolution720p:
		return 2
	case plugin.Resolution576p, plugin.Resolution480p, plugin.ResolutionSD:
		return 1
	default:
		return 0
	}
}

// sourceRank maps a Source to a comparable integer (higher = better).
func sourceRank(s plugin.Source) int {
	switch s {
	case plugin.SourceRawHD:
		return 9
	case plugin.SourceBRDisk:
		return 8
	case plugin.SourceRemux:
		return 7
	case plugin.SourceBluRay:
		return 6
	case plugin.SourceWEBDL:
		return 5
	case plugin.SourceWEBRip:
		return 4
	case plugin.SourceHDTV:
		return 3
	case plugin.SourceDVD, plugin.SourceDVDR:
		return 2
	case plugin.SourceDVDSCR, plugin.SourceRegional, plugin.SourceTELECINE:
		return 1
	case plugin.SourceTelesync, plugin.SourceCAM:
		return 0
	case plugin.SourceWorkprint:
		return 0
	default:
		return 0
	}
}

// codecRank maps a Codec to a comparable integer (higher = better).
func codecRank(c plugin.Codec) int {
	switch c {
	case plugin.CodecAV1:
		return 3
	case plugin.CodecX265:
		return 2
	case plugin.CodecX264:
		return 1
	default:
		return 0
	}
}

// AllowedQualities returns the list of quality values this profile accepts.
// The slice is a copy; mutations do not affect the profile.
func (p *Profile) AllowedQualities() []plugin.Quality {
	out := make([]plugin.Quality, len(p.Qualities))
	copy(out, p.Qualities)
	return out
}

// isAllowed checks whether q appears in p.Qualities using Score equality.
// We compare by Score rather than struct equality so that the Name field
// (a derived label) doesn't cause false negatives.
//
// An empty Qualities list means "accept any quality", allowing a simple
// catch-all "Any" profile without enumerating every quality combination.
func (p *Profile) isAllowed(q plugin.Quality) bool {
	if len(p.Qualities) == 0 {
		return true
	}
	score := q.Score()
	for _, allowed := range p.Qualities {
		if allowed.Score() == score {
			return true
		}
	}
	return false
}
