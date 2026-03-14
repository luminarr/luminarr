// Package renamer applies naming format templates to produce filesystem-safe
// filenames for imported movie files.
package renamer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// DefaultFileFormat is used when a library has no naming_format set.
const DefaultFileFormat = "{Movie Title} ({Release Year}) {Quality Full}"

// DefaultFolderFormat is used when a library has no folder_format set.
const DefaultFolderFormat = "{Movie Title} ({Release Year})"

// ColonReplacement controls how colons in movie titles are handled when
// producing filesystem-safe filenames.
type ColonReplacement string

const (
	// ColonDelete removes colons: "Batman: Begins" → "Batman Begins"
	ColonDelete ColonReplacement = "delete"
	// ColonDash replaces colons with a dash: "Batman: Begins" → "Batman- Begins"
	ColonDash ColonReplacement = "dash"
	// ColonSpaceDash replaces ": " with " - ": "Batman: Begins" → "Batman - Begins"
	ColonSpaceDash ColonReplacement = "space-dash"
	// ColonSmart uses space-dash when followed by a space, dash otherwise.
	ColonSmart ColonReplacement = "smart"
)

// Movie holds the movie metadata the renamer needs.
type Movie struct {
	Title         string
	OriginalTitle string
	Year          int
	Edition       string // detected edition of the file; empty = untagged
}

// Apply returns the formatted filename (without extension) for the given movie,
// quality, and format string. Substitution variables:
//
//	{Movie Title}          → movie.Title
//	{Movie CleanTitle}     → filesystem-safe version of movie.Title (delete colon strategy)
//	{Original Title}       → movie.OriginalTitle
//	{Release Year}         → movie.Year
//	{Quality Full}         → quality.Name  (e.g. "Bluray-1080p")
//	{MediaInfo VideoCodec} → quality.Codec (e.g. "x265")
//	{Edition}              → movie.Edition (e.g. "Director's Cut"); empty when untagged
func Apply(format string, m Movie, q plugin.Quality) string {
	return ApplyWithOptions(format, m, q, ColonDelete)
}

// ApplyWithOptions is like Apply but allows specifying the colon replacement strategy.
func ApplyWithOptions(format string, m Movie, q plugin.Quality, colon ColonReplacement) string {
	r := strings.NewReplacer(
		"{Movie Title}", m.Title,
		"{Movie CleanTitle}", CleanTitleColon(m.Title, colon),
		"{Original Title}", m.OriginalTitle,
		"{Release Year}", yearStr(m.Year),
		"{Quality Full}", q.Name,
		"{MediaInfo VideoCodec}", string(q.Codec),
		"{Edition}", m.Edition,
	)
	result := r.Replace(format)
	return sanitize(result)
}

// FolderName returns the library sub-directory name for a movie using the
// given folder format template. Pass DefaultFolderFormat for the standard behaviour.
func FolderName(format string, m Movie) string {
	return Apply(format, m, plugin.Quality{})
}

// DestPath returns the absolute destination path for an imported file.
//
//	libraryRoot / FolderName(folderFormat, m) / ApplyWithOptions(fileFormat, m, q, colon) + ext
func DestPath(libraryRoot, fileFormat, folderFormat string, m Movie, q plugin.Quality, colon ColonReplacement, sourceExt string) string {
	folder := FolderName(folderFormat, m)
	file := ApplyWithOptions(fileFormat, m, q, colon) + sourceExt
	return filepath.Join(libraryRoot, folder, file)
}

// CleanTitle strips characters that are problematic on common filesystems
// while preserving readability. Colons are replaced with " - " (space-dash).
// Used for {Movie CleanTitle} with the default space-dash strategy.
func CleanTitle(title string) string {
	return CleanTitleColon(title, ColonSpaceDash)
}

// CleanTitleColon is like CleanTitle but applies the specified colon replacement.
func CleanTitleColon(title string, colon ColonReplacement) string {
	switch colon {
	case ColonDash:
		title = strings.ReplaceAll(title, ":", "-")
	case ColonSpaceDash, ColonSmart:
		// Replace ": " (colon-space) with " - "; bare ":" with "-"
		title = strings.ReplaceAll(title, ": ", " - ")
		title = strings.ReplaceAll(title, ":", "-")
	default: // ColonDelete
		title = strings.ReplaceAll(title, ":", " ")
	}
	// Remove characters invalid on most filesystems.
	title = invalidCharsRe.ReplaceAllString(title, "")
	// Collapse multiple spaces.
	title = multiSpaceRe.ReplaceAllString(title, " ")
	return strings.TrimSpace(title)
}

// sanitize makes a string safe to use as a filename: removes path separators
// and collapses whitespace. Does not strip colons or other title chars so that
// the full {Movie Title} variable retains its value; use CleanTitle for that.
func sanitize(s string) string {
	// Remove path separators and null bytes.
	s = strings.NewReplacer("/", "", "\x00", "").Replace(s)
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func yearStr(y int) string {
	if y == 0 {
		return ""
	}
	return fmt.Sprintf("%d", y)
}

var (
	invalidCharsRe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	multiSpaceRe   = regexp.MustCompile(`\s{2,}`)
)
