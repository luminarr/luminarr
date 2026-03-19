// Package conflict detects quality regressions between a current file and a
// candidate release. It is a pure, stateless package with no service dependencies.
package conflict

import (
	"fmt"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// Severity indicates how significant a regression is.
const (
	SeverityWarning = "warning" // significant regression (HDR lost, lossless audio lost)
	SeverityCaution = "caution" // minor regression (codec change, small channel drop)
)

// Conflict describes a single dimension regression.
type Conflict struct {
	Dimension string `json:"dimension"` // "audio_codec", "audio_channels", "hdr", "edition", "resolution", "codec"
	Severity  string `json:"severity"`  // "warning" or "caution"
	Current   string `json:"current"`   // human-readable current value
	Candidate string `json:"candidate"` // human-readable candidate value
	Summary   string `json:"summary"`   // "Audio downgrade: TrueHD Atmos → AC3 5.1"
}

// Compare checks each quality dimension for regressions. If the candidate
// ranks lower than current in any dimension, a Conflict is returned for it.
// Dimensions where either side is unknown/empty are skipped.
func Compare(current, candidate plugin.Quality, currentEdition, candidateEdition string) []Conflict {
	var conflicts []Conflict

	// 1. Resolution
	if c := compareResolution(current.Resolution, candidate.Resolution); c != nil {
		conflicts = append(conflicts, *c)
	}

	// 2. Video codec
	if c := compareCodec(current.Codec, candidate.Codec); c != nil {
		conflicts = append(conflicts, *c)
	}

	// 3. HDR
	if c := compareHDR(current.HDR, candidate.HDR); c != nil {
		conflicts = append(conflicts, *c)
	}

	// 4. Audio codec
	if c := compareAudioCodec(current.AudioCodec, candidate.AudioCodec); c != nil {
		conflicts = append(conflicts, *c)
	}

	// 5. Audio channels
	if c := compareAudioChannels(current.AudioChannels, candidate.AudioChannels); c != nil {
		conflicts = append(conflicts, *c)
	}

	// 6. Edition
	if c := compareEdition(currentEdition, candidateEdition); c != nil {
		conflicts = append(conflicts, *c)
	}

	return conflicts
}

// ── Resolution ──────────────────────────────────────────────────────────────

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

func compareResolution(cur, cand plugin.Resolution) *Conflict {
	cr, candr := resolutionRank(cur), resolutionRank(cand)
	if cr == 0 || candr == 0 || candr >= cr {
		return nil
	}
	sev := SeverityCaution
	if cr-candr >= 2 {
		sev = SeverityWarning
	}
	return &Conflict{
		Dimension: "resolution",
		Severity:  sev,
		Current:   displayRes(cur),
		Candidate: displayRes(cand),
		Summary:   fmt.Sprintf("Resolution downgrade: %s → %s", displayRes(cur), displayRes(cand)),
	}
}

func displayRes(r plugin.Resolution) string {
	if r == plugin.ResolutionSD {
		return "SD"
	}
	return string(r)
}

// ── Video Codec ─────────────────────────────────────────────────────────────

func codecRank(c plugin.Codec) int {
	switch c {
	case plugin.CodecAV1:
		return 4
	case plugin.CodecX265:
		return 3
	case plugin.CodecX264:
		return 2
	case plugin.CodecXVID:
		return 1
	default:
		return 0
	}
}

func compareCodec(cur, cand plugin.Codec) *Conflict {
	cr, candr := codecRank(cur), codecRank(cand)
	if cr == 0 || candr == 0 || candr >= cr {
		return nil
	}
	return &Conflict{
		Dimension: "codec",
		Severity:  SeverityCaution,
		Current:   displayCodec(cur),
		Candidate: displayCodec(cand),
		Summary:   fmt.Sprintf("Codec downgrade: %s → %s", displayCodec(cur), displayCodec(cand)),
	}
}

func displayCodec(c plugin.Codec) string {
	switch c {
	case plugin.CodecX265:
		return "x265"
	case plugin.CodecX264:
		return "x264"
	case plugin.CodecAV1:
		return "AV1"
	case plugin.CodecXVID:
		return "XviD"
	default:
		return string(c)
	}
}

// ── HDR ─────────────────────────────────────────────────────────────────────

func hdrRank(h plugin.HDRFormat) int {
	switch h {
	case plugin.HDRDolbyVision:
		return 5
	case plugin.HDRHDR10Plus:
		return 4
	case plugin.HDRHDR10:
		return 3
	case plugin.HDRHLG:
		return 2
	case plugin.HDRNone:
		return 1
	default:
		return 0
	}
}

func compareHDR(cur, cand plugin.HDRFormat) *Conflict {
	cr, candr := hdrRank(cur), hdrRank(cand)
	// Only flag if current actually has HDR (rank > 1) and candidate is worse.
	if cr <= 1 || candr == 0 || candr >= cr {
		return nil
	}
	label := "HDR downgrade"
	if candr <= 1 {
		label = "HDR lost"
	}
	return &Conflict{
		Dimension: "hdr",
		Severity:  SeverityWarning,
		Current:   displayHDR(cur),
		Candidate: displayHDR(cand),
		Summary:   fmt.Sprintf("%s: %s → %s", label, displayHDR(cur), displayHDR(cand)),
	}
}

func displayHDR(h plugin.HDRFormat) string {
	switch h {
	case plugin.HDRDolbyVision:
		return "Dolby Vision"
	case plugin.HDRHDR10Plus:
		return "HDR10+"
	case plugin.HDRHDR10:
		return "HDR10"
	case plugin.HDRHLG:
		return "HLG"
	case plugin.HDRNone:
		return "SDR"
	default:
		return string(h)
	}
}

// ── Audio Codec ─────────────────────────────────────────────────────────────

func audioCodecRank(a plugin.AudioCodec) int {
	switch a {
	case plugin.AudioCodecTrueHDAtmos:
		return 10
	case plugin.AudioCodecDTSX:
		return 9
	case plugin.AudioCodecTrueHD:
		return 8
	case plugin.AudioCodecDTSHDMA:
		return 7
	case plugin.AudioCodecFLAC:
		return 6
	case plugin.AudioCodecPCM:
		return 6
	case plugin.AudioCodecEAC3Atmos:
		return 5
	case plugin.AudioCodecDTSHD:
		return 4
	case plugin.AudioCodecEAC3:
		return 3
	case plugin.AudioCodecDTS:
		return 2
	case plugin.AudioCodecAC3:
		return 2
	case plugin.AudioCodecAAC:
		return 1
	case plugin.AudioCodecMP3:
		return 1
	case plugin.AudioCodecOpus:
		return 1
	default:
		return 0
	}
}

func compareAudioCodec(cur, cand plugin.AudioCodec) *Conflict {
	cr, candr := audioCodecRank(cur), audioCodecRank(cand)
	if cr == 0 || candr == 0 || candr >= cr {
		return nil
	}
	sev := SeverityCaution
	// Significant: dropping from lossless (rank >= 6) to lossy (rank < 6).
	if cr >= 6 && candr < 6 {
		sev = SeverityWarning
	}
	return &Conflict{
		Dimension: "audio_codec",
		Severity:  sev,
		Current:   displayAudioCodec(cur),
		Candidate: displayAudioCodec(cand),
		Summary:   fmt.Sprintf("Audio downgrade: %s → %s", displayAudioCodec(cur), displayAudioCodec(cand)),
	}
}

func displayAudioCodec(a plugin.AudioCodec) string {
	switch a {
	case plugin.AudioCodecTrueHDAtmos:
		return "TrueHD Atmos"
	case plugin.AudioCodecDTSX:
		return "DTS:X"
	case plugin.AudioCodecTrueHD:
		return "TrueHD"
	case plugin.AudioCodecDTSHDMA:
		return "DTS-HD MA"
	case plugin.AudioCodecFLAC:
		return "FLAC"
	case plugin.AudioCodecPCM:
		return "PCM"
	case plugin.AudioCodecEAC3Atmos:
		return "DD+ Atmos"
	case plugin.AudioCodecDTSHD:
		return "DTS-HD"
	case plugin.AudioCodecEAC3:
		return "DD+"
	case plugin.AudioCodecDTS:
		return "DTS"
	case plugin.AudioCodecAC3:
		return "DD"
	case plugin.AudioCodecAAC:
		return "AAC"
	case plugin.AudioCodecMP3:
		return "MP3"
	case plugin.AudioCodecOpus:
		return "Opus"
	default:
		return string(a)
	}
}

// ── Audio Channels ──────────────────────────────────────────────────────────

func audioChannelsRank(ch plugin.AudioChannels) int {
	switch ch {
	case plugin.AudioChannels71:
		return 4
	case plugin.AudioChannels51:
		return 3
	case plugin.AudioChannels20:
		return 2
	case plugin.AudioChannels10:
		return 1
	default:
		return 0
	}
}

func compareAudioChannels(cur, cand plugin.AudioChannels) *Conflict {
	cr, candr := audioChannelsRank(cur), audioChannelsRank(cand)
	if cr == 0 || candr == 0 || candr >= cr {
		return nil
	}
	sev := SeverityCaution
	// Dropping from surround (5.1+) to stereo/mono is significant.
	if cr >= 3 && candr < 3 {
		sev = SeverityWarning
	}
	return &Conflict{
		Dimension: "audio_channels",
		Severity:  sev,
		Current:   string(cur),
		Candidate: string(cand),
		Summary:   fmt.Sprintf("Channel downgrade: %s → %s", string(cur), string(cand)),
	}
}

// ── Edition ─────────────────────────────────────────────────────────────────

func compareEdition(cur, cand string) *Conflict {
	if cur == "" {
		return nil // no current edition to lose
	}
	if cand == cur {
		return nil // same edition
	}
	if cand == "" {
		return &Conflict{
			Dimension: "edition",
			Severity:  SeverityCaution,
			Current:   cur,
			Candidate: "(none)",
			Summary:   fmt.Sprintf("Edition lost: %s → (none)", cur),
		}
	}
	return &Conflict{
		Dimension: "edition",
		Severity:  SeverityCaution,
		Current:   cur,
		Candidate: cand,
		Summary:   fmt.Sprintf("Edition change: %s → %s", cur, cand),
	}
}
