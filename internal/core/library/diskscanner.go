package library

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/davidfic/luminarr/pkg/plugin"
)

// DiskFile represents a video file found on disk during a library scan.
type DiskFile struct {
	Path        string
	SizeBytes   int64
	ParsedTitle string
	ParsedYear  int
	TMDBMatch   *DiskFileTMDBMatch // nil if not yet matched
}

// DiskFileTMDBMatch holds the pre-computed TMDB match for a DiskFile.
type DiskFileTMDBMatch struct {
	TMDBID        int
	Title         string
	OriginalTitle string
	Year          int
}

// videoExtensions is the set of file extensions recognised as video files.
var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
	".wmv": true, ".m4v": true, ".ts": true, ".webm": true,
	".m2ts": true, ".mpg": true, ".mpeg": true, ".flv": true,
}

// yearRe matches a 4-digit year between 1900 and 2099.
var yearRe = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)

// scanDisk walks root recursively and returns video files whose absolute paths
// are not present in knownPaths. Unreadable entries are silently skipped.
func scanDisk(root string, knownPaths map[string]bool) ([]DiskFile, error) {
	var files []DiskFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries and continue the walk
		}
		if d.IsDir() {
			// Skip hidden directories (dot-prefixed) and NAS special directories
			// such as Synology's #recycle and #snapshot.
			name := d.Name()
			if len(name) > 0 && (name[0] == '.' || name[0] == '#') {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !videoExtensions[ext] || knownPaths[path] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil //nolint:nilerr // returning nil in WalkDir callback continues the walk; we intentionally skip unreadable files
		}
		title, year := parseFilename(filepath.Base(path))
		// If the filename yields a poor result (no year or purely numeric
		// title), try the parent directory name which often contains the
		// real movie title in torrent-style layouts.
		if year == 0 || isNumeric(title) {
			dir := filepath.Base(filepath.Dir(path))
			if dir != "." && dir != "/" {
				dirTitle, dirYear := parseFilename(dir)
				if dirTitle != "" && !isNumeric(dirTitle) {
					title = dirTitle
					if dirYear > 0 {
						year = dirYear
					}
				}
			}
		}
		files = append(files, DiskFile{
			Path:        path,
			SizeBytes:   info.Size(),
			ParsedTitle: title,
			ParsedYear:  year,
		})
		return nil
	})
	return files, err
}

// parseFilename extracts a guessed title and year from a video filename.
// Handles common patterns such as:
//   - "Movie Title (2010).mkv"
//   - "Movie.Title.2010.1080p.BluRay.x265.mkv"
//   - "Movie_Title_2010.mkv"
func parseFilename(name string) (title string, year int) {
	// Strip extension.
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Locate the release year. We use the *last* match so that titles
	// containing year-like numbers (e.g. "2001 A Space Odyssey", "1917",
	// "Blade Runner 2049") are not truncated.
	allYears := yearRe.FindAllStringIndex(name, -1)
	if len(allYears) > 0 {
		m := allYears[len(allYears)-1]
		year, _ = strconv.Atoi(name[m[0]:m[1]])
		name = name[:m[0]]
	}

	// Normalise separators to spaces.
	name = strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(name)

	// Collapse whitespace and trailing noise characters.
	name = strings.Join(strings.Fields(name), " ")
	name = strings.TrimRight(name, "( ")
	name = strings.TrimSpace(name)

	return name, year
}

// isNumeric reports whether s is empty or consists entirely of digits and spaces.
func isNumeric(s string) bool {
	if s == "" {
		return true
	}
	for _, c := range s {
		if c != ' ' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// ParseQualityFromPath infers video quality metadata from a file path using
// simple pattern matching. It is best-effort; fields that cannot be determined
// default to the "unknown" / "none" plugin constants.
func ParseQualityFromPath(path string) plugin.Quality {
	upper := strings.ToUpper(path)

	q := plugin.Quality{
		Resolution: plugin.ResolutionUnknown,
		Source:     plugin.SourceUnknown,
		Codec:      plugin.CodecUnknown,
		HDR:        plugin.HDRNone,
	}

	switch {
	case strings.Contains(upper, "2160P") || strings.Contains(upper, "4K") || strings.Contains(upper, "UHD"):
		q.Resolution = plugin.Resolution2160p
	case strings.Contains(upper, "1080P") || strings.Contains(upper, "1080I"):
		q.Resolution = plugin.Resolution1080p
	case strings.Contains(upper, "720P"):
		q.Resolution = plugin.Resolution720p
	}

	switch {
	case strings.Contains(upper, "REMUX"):
		q.Source = plugin.SourceRemux
	case strings.Contains(upper, "BLURAY") || strings.Contains(upper, "BLU-RAY") || strings.Contains(upper, "BDRIP"):
		q.Source = plugin.SourceBluRay
	case strings.Contains(upper, "WEB-DL") || strings.Contains(upper, "WEBDL"):
		q.Source = plugin.SourceWEBDL
	case strings.Contains(upper, "WEBRIP") || strings.Contains(upper, "WEB-RIP"):
		q.Source = plugin.SourceWEBRip
	case strings.Contains(upper, "HDTV"):
		q.Source = plugin.SourceHDTV
	case strings.Contains(upper, "DVDRIP") || strings.Contains(upper, "DVD"):
		q.Source = plugin.SourceDVD
	}

	switch {
	case strings.Contains(upper, "X265") || strings.Contains(upper, "H265") || strings.Contains(upper, "HEVC"):
		q.Codec = plugin.CodecX265
	case strings.Contains(upper, "X264") || strings.Contains(upper, "H264") || strings.Contains(upper, "AVC"):
		q.Codec = plugin.CodecX264
	case strings.Contains(upper, "AV1"):
		q.Codec = plugin.CodecAV1
	}

	switch {
	case strings.Contains(upper, "HDR10PLUS") || strings.Contains(upper, "HDR10+"):
		q.HDR = plugin.HDRHDR10Plus
	case strings.Contains(upper, "DOLBY.VISION") || strings.Contains(upper, "DV.") || strings.Contains(upper, ".DV"):
		q.HDR = plugin.HDRDolbyVision
	case strings.Contains(upper, "HLG"):
		q.HDR = plugin.HDRHLG
	case strings.Contains(upper, "HDR"):
		q.HDR = plugin.HDRHDR10
	}

	q.Name = buildQualityName(q)
	return q
}

func buildQualityName(q plugin.Quality) string {
	var parts []string
	if q.Source != plugin.SourceUnknown {
		parts = append(parts, string(q.Source))
	}
	if q.Resolution != plugin.ResolutionUnknown {
		parts = append(parts, string(q.Resolution))
	}
	if q.Codec != plugin.CodecUnknown {
		parts = append(parts, string(q.Codec))
	}
	if q.HDR != plugin.HDRNone && q.HDR != plugin.HDRUnknown {
		parts = append(parts, string(q.HDR))
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, " ")
}
