package parser

import (
	"regexp"
	"strings"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// All regexps are compiled once at package init — never inside parse functions.

var (
	// ── Resolution ───────────────────────────────────────────────────────
	re2160p = regexp.MustCompile(`(?i)(?:\b4k\b|\buhd\b|2160[pi]|3840x2160)`)
	re1080p = regexp.MustCompile(`(?i)(?:1080[pi]|1920x1080)`)
	re720p  = regexp.MustCompile(`(?i)(?:720p|1280x720)`)
	re576p  = regexp.MustCompile(`(?i)576p`)
	re480p  = regexp.MustCompile(`(?i)(?:480[pi]|640x480|540p)`)

	// ── Source ───────────────────────────────────────────────────────────
	// RawHD: MPEG-2 HDTV captures (must check MPEG-2 + HDTV combo)
	reRawHD     = regexp.MustCompile(`(?i)\braw[\s\-]?hd\b`)
	reMPEG2HDTV = regexp.MustCompile(`(?i)mpeg[\s\-]?2`)

	// BRDisk: full Blu-ray disc images, ISOs, HD DVD discs, COMPLETE BLURAY
	reBRDisk = regexp.MustCompile(`(?i)(?:\bbdmv\b|\bbd[\s\-]?25\b|\bbd[\s\-]?50\b|\bbd[\s\-]?66\b|\bbr[\s\-]?disk\b|\bbdiso\b|blu[\s\-]?ray[^\n]*\.iso\b|\bcomplete[\s.]blu[\s\-]?ray\b|\bhd[\s.]dvd\b|\buntouched\b|(?:^|\W)iso(?:\W|$))`)

	// Remux (must precede BluRay so REMUX wins)
	reRemux = regexp.MustCompile(`(?i)(?:\bremux\b|\bbdremux\b|\buhd[\s\-]?remux\b)`)

	// BluRay and its many variants: Blu-ray, BDRip, BDMux, BDLight,
	// UHDBD, UHDBDRip, UHD2BD, HDDVDRip, HDDVD, MBluRay, m2ts,
	// BD720p, BD1080p, BD2160p, [BD], (BD ...), BD inside brackets
	reBluRay = regexp.MustCompile(`(?i)(?:blu[\s\-]?ray|bluray|\bbdrip\b|\bbdmux\b|\bbdlight\b|\buhdbd\b|\buhdbdrip\b|\buhd2bd\b|\bhddvd(?:rip)?\b|\bmbluray\b|\bm2ts\b|\bbd\d{3,4}p\b|\bbd\b[\s]+\d{3,4}p|[\[\(]bd[\s\]\)]|\bbd\b[\s]*[\]\)])`)

	// WEB-DL and its variants (including bare "WEB", iTunesHD, WebHD)
	reWEBDL = regexp.MustCompile(`(?i)(?:web[\s\-.]?dl|\bwebdl\b|\bituneshd\b|\bwebhd\b)`)

	// WEBRip and WEBMux
	reWEBRip = regexp.MustCompile(`(?i)(?:web[\s\-.]?rip|\bwebrip\b|\bwebmux\b)`)

	// Bare "WEB" as fallback for WEB-DL (after WEB-DL and WEBRip fail)
	reWEBBare = regexp.MustCompile(`(?i)(?:\bweb\b|\[web\])`)

	// HDTV and its variants: PDTV, DSR, TVRip, HD TV, SD TV
	reHDTV = regexp.MustCompile(`(?i)(?:\bhdtv\b|\bpdtv\b|\bdsr\b|\btvrip\b|\bhd[\s\-.]tv\b|\bsd[\s\-.]tv\b)`)

	reDVDSCR = regexp.MustCompile(`(?i)(?:\bdvdscr\b|\bscreener\b|\bscr\b)`)

	// DVDR variants: DVD-R, DVDR, DVD5, DVD9 (with optional numeric prefix like 2xDVD9, 2DVD5)
	reDVDR = regexp.MustCompile(`(?i)(?:\bdvd[\s\-]?r\b|\bm?dvdr\b|\d*x?dvd9\b|\d*dvd5\b)`)

	reDVDRip   = regexp.MustCompile(`(?i)dvd[\s\-.]?rip`)
	reDVDBare  = regexp.MustCompile(`(?i)\bdvd\b`)
	reRegional = regexp.MustCompile(`(?i)(?:\br5\b|\bregional\b)`)
	reTelecine = regexp.MustCompile(`(?i)(?:\btelecine\b|\bhdtc\b|\btc\b)`)

	// Telesync and its variants: TSRip, TeleSynch
	reTelesync  = regexp.MustCompile(`(?i)(?:\btelesync\b|\btelesynch?\b|\bhdts\b|\bpdvd\b|\bts(?:rip)?\b)`)
	reCAM       = regexp.MustCompile(`(?i)(?:hd|hq|new)?cam(?:rip)?\b`)
	reWorkprint = regexp.MustCompile(`(?i)(?:\bworkprint\b|\bwp\b)`)

	// ── HDR ──────────────────────────────────────────────────────────────
	reDolbyVision = regexp.MustCompile(`(?i)(?:\bdv\b|dovi|dolby[\s\-.]?vision)`)
	reHDR10Plus   = regexp.MustCompile(`(?i)(?:hdr10\+|hdr10plus)`)
	reHDR10       = regexp.MustCompile(`(?i)hdr(?:10)?`)
	reHLG         = regexp.MustCompile(`(?i)\bhlg\b`)

	// ── Codec ────────────────────────────────────────────────────────────
	reX265 = regexp.MustCompile(`(?i)x265|h[\s\-.]?265|hevc`)
	reX264 = regexp.MustCompile(`(?i)x264|h[\s\-.]?264|avc`)
	reAV1  = regexp.MustCompile(`(?i)\bav1\b`)
	reXVID = regexp.MustCompile(`(?i)xvid|divx`)
)

func parseSource(norm string) plugin.Source {
	// RAW-HD explicit tag.
	if reRawHD.MatchString(norm) {
		return plugin.SourceRawHD
	}

	// MPEG-2 + HDTV = RawHD (Radarr: TV source + RAWHD modifier).
	hasHDTV := reHDTV.MatchString(norm)
	if reMPEG2HDTV.MatchString(norm) && hasHDTV {
		return plugin.SourceRawHD
	}

	// BRDisk must precede Remux and BluRay: full discs, ISOs, HD DVD.
	if reBRDisk.MatchString(norm) {
		return plugin.SourceBRDisk
	}

	// Remux must precede BluRay so "BluRay REMUX" → Remux.
	if reRemux.MatchString(norm) {
		return plugin.SourceRemux
	}

	// BluRay (including BDRip, BDMux, BDLight, UHDBD, etc.)
	if reBluRay.MatchString(norm) {
		return plugin.SourceBluRay
	}

	// WEB-DL explicit.
	if reWEBDL.MatchString(norm) {
		return plugin.SourceWEBDL
	}

	// WEBRip / WEBMux.
	if reWEBRip.MatchString(norm) {
		return plugin.SourceWEBRip
	}

	// Bare "WEB" falls back to WEB-DL (Radarr treats bare WEB as WEBDL).
	if reWEBBare.MatchString(norm) {
		return plugin.SourceWEBDL
	}

	// HDTV / PDTV / DSR / TVRip / HD TV / SD TV.
	if hasHDTV {
		return plugin.SourceHDTV
	}

	switch {
	case reDVDSCR.MatchString(norm):
		return plugin.SourceDVDSCR
	case reDVDR.MatchString(norm):
		return plugin.SourceDVDR
	case reDVDRip.MatchString(norm):
		return plugin.SourceDVD
	case reDVDBare.MatchString(norm):
		return plugin.SourceDVD
	case reRegional.MatchString(norm):
		return plugin.SourceRegional
	case reTelecine.MatchString(norm):
		return plugin.SourceTELECINE
	case reTelesync.MatchString(norm):
		return plugin.SourceTelesync
	case reCAM.MatchString(norm):
		return plugin.SourceCAM
	case reWorkprint.MatchString(norm):
		return plugin.SourceWorkprint
	default:
		return plugin.SourceUnknown
	}
}

// reExplicitRes matches explicit numeric resolution tags (e.g. "2160p",
// "1080p", "720p"). These take precedence over implicit hints like "4K"
// or "UHD" which may refer to the source disc rather than the output
// resolution (e.g. "1080p.UHD.BluRay" is 1080p, not 2160p).
var reExplicitRes = regexp.MustCompile(`(?i)\b(2160|1080|720|576|480|540)[pi]\b`)

func parseResolution(norm string, src plugin.Source) plugin.Resolution {
	// First check for explicit numeric resolution (highest priority).
	if m := reExplicitRes.FindStringSubmatch(norm); m != nil {
		switch m[1] {
		case "2160":
			return plugin.Resolution2160p
		case "1080":
			return plugin.Resolution1080p
		case "720":
			return plugin.Resolution720p
		case "576":
			return plugin.Resolution576p
		case "480", "540":
			return plugin.Resolution480p
		}
	}

	// Dimension-based resolution (3840x2160, 1920x1080, etc.)
	switch {
	case re2160p.MatchString(norm):
		return plugin.Resolution2160p
	case re1080p.MatchString(norm):
		return plugin.Resolution1080p
	case re720p.MatchString(norm):
		return plugin.Resolution720p
	case re576p.MatchString(norm):
		return plugin.Resolution576p
	case re480p.MatchString(norm):
		return plugin.Resolution480p
	case src == plugin.SourceDVD || src == plugin.SourceDVDR || src == plugin.SourceDVDSCR || src == plugin.SourceRegional:
		return plugin.ResolutionSD
	default:
		return plugin.ResolutionUnknown
	}
}

func parseHDR(norm string) plugin.HDRFormat {
	switch {
	case reDolbyVision.MatchString(norm):
		return plugin.HDRDolbyVision
	case reHDR10Plus.MatchString(norm):
		return plugin.HDRHDR10Plus
	case reHDR10.MatchString(norm):
		return plugin.HDRHDR10
	case reHLG.MatchString(norm):
		return plugin.HDRHLG
	default:
		return plugin.HDRNone
	}
}

func parseCodec(norm string) plugin.Codec {
	switch {
	case reX265.MatchString(norm):
		return plugin.CodecX265
	case reX264.MatchString(norm):
		return plugin.CodecX264
	case reAV1.MatchString(norm):
		return plugin.CodecAV1
	case reXVID.MatchString(norm):
		return plugin.CodecXVID
	default:
		return plugin.CodecUnknown
	}
}

// buildQualityName produces the human-readable quality label.
func buildQualityName(res plugin.Resolution, src plugin.Source, codec plugin.Codec, hdr plugin.HDRFormat) string {
	srcLabel := sourceLabel(src)
	resLabel := resolutionLabel(res)
	codecLabel := codecLabel(codec)
	hdrLabel := hdrLabel(hdr)

	var sb strings.Builder
	if srcLabel != "" && resLabel != "" {
		sb.WriteString(srcLabel)
		sb.WriteByte('-')
		sb.WriteString(resLabel)
	} else if srcLabel != "" {
		sb.WriteString(srcLabel)
	} else if resLabel != "" {
		sb.WriteString(resLabel)
	}
	if codecLabel != "" {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(codecLabel)
	}
	if hdrLabel != "" {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(hdrLabel)
	}
	return sb.String()
}

func sourceLabel(src plugin.Source) string {
	switch src {
	case plugin.SourceRawHD:
		return "Raw-HD"
	case plugin.SourceBRDisk:
		return "BR-DISK"
	case plugin.SourceRemux:
		return "Bluray Remux"
	case plugin.SourceBluRay:
		return "Bluray"
	case plugin.SourceWEBDL:
		return "WEBDL"
	case plugin.SourceWEBRip:
		return "WEBRip"
	case plugin.SourceHDTV:
		return "HDTV"
	case plugin.SourceDVDSCR:
		return "DVDSCR"
	case plugin.SourceDVDR:
		return "DVD-R"
	case plugin.SourceDVD:
		return "DVD"
	case plugin.SourceRegional:
		return "Regional"
	case plugin.SourceTELECINE:
		return "Telecine"
	case plugin.SourceTelesync:
		return "Telesync"
	case plugin.SourceCAM:
		return "CAM"
	case plugin.SourceWorkprint:
		return "Workprint"
	default:
		return ""
	}
}

func resolutionLabel(res plugin.Resolution) string {
	switch res {
	case plugin.Resolution2160p:
		return "2160p"
	case plugin.Resolution1080p:
		return "1080p"
	case plugin.Resolution720p:
		return "720p"
	case plugin.Resolution576p:
		return "576p"
	case plugin.Resolution480p:
		return "480p"
	case plugin.ResolutionSD:
		return "SD"
	default:
		return ""
	}
}

func codecLabel(codec plugin.Codec) string {
	switch codec {
	case plugin.CodecX265:
		return "x265"
	case plugin.CodecX264:
		return "x264"
	case plugin.CodecAV1:
		return "AV1"
	case plugin.CodecXVID:
		return "XviD"
	default:
		return ""
	}
}

func hdrLabel(hdr plugin.HDRFormat) string {
	switch hdr {
	case plugin.HDRHDR10:
		return "HDR10"
	case plugin.HDRDolbyVision:
		return "Dolby Vision"
	case plugin.HDRHLG:
		return "HLG"
	case plugin.HDRHDR10Plus:
		return "HDR10+"
	default:
		return ""
	}
}
