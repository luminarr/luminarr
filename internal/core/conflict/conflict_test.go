package conflict

import (
	"testing"

	"github.com/luminarr/luminarr/pkg/plugin"
)

func TestCompare_NoConflictsOnUpgrade(t *testing.T) {
	t.Parallel()
	current := plugin.Quality{
		Resolution: plugin.Resolution720p, Source: plugin.SourceWEBDL,
		Codec: plugin.CodecX264, HDR: plugin.HDRNone,
		AudioCodec: plugin.AudioCodecAC3, AudioChannels: plugin.AudioChannels51,
	}
	candidate := plugin.Quality{
		Resolution: plugin.Resolution1080p, Source: plugin.SourceBluRay,
		Codec: plugin.CodecX265, HDR: plugin.HDRHDR10,
		AudioCodec: plugin.AudioCodecTrueHDAtmos, AudioChannels: plugin.AudioChannels71,
	}
	conflicts := Compare(current, candidate, "", "")
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for full upgrade, got %d: %v", len(conflicts), conflicts)
	}
}

func TestCompare_NoConflictsOnEqual(t *testing.T) {
	t.Parallel()
	q := plugin.Quality{
		Resolution: plugin.Resolution1080p, Source: plugin.SourceBluRay,
		Codec: plugin.CodecX265, HDR: plugin.HDRHDR10,
		AudioCodec: plugin.AudioCodecDTSHDMA, AudioChannels: plugin.AudioChannels51,
	}
	conflicts := Compare(q, q, "Extended", "Extended")
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for equal quality, got %d", len(conflicts))
	}
}

func TestCompare_SkipsUnknownDimensions(t *testing.T) {
	t.Parallel()
	current := plugin.Quality{Resolution: plugin.Resolution1080p}
	candidate := plugin.Quality{Resolution: plugin.Resolution720p}
	// All other dimensions are zero/empty — should only get resolution conflict.
	conflicts := Compare(current, candidate, "", "")
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %v", len(conflicts), conflicts)
	}
	if conflicts[0].Dimension != "resolution" {
		t.Errorf("dimension: got %q, want resolution", conflicts[0].Dimension)
	}
}

func TestCompare_Resolution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cur     plugin.Resolution
		cand    plugin.Resolution
		want    bool
		wantSev string
	}{
		{"2160p→1080p caution", plugin.Resolution2160p, plugin.Resolution1080p, true, SeverityCaution},
		{"2160p→720p warning", plugin.Resolution2160p, plugin.Resolution720p, true, SeverityWarning},
		{"1080p→720p caution", plugin.Resolution1080p, plugin.Resolution720p, true, SeverityCaution},
		{"720p→1080p no conflict", plugin.Resolution720p, plugin.Resolution1080p, false, ""},
		{"same no conflict", plugin.Resolution1080p, plugin.Resolution1080p, false, ""},
		{"unknown cur skip", plugin.ResolutionUnknown, plugin.Resolution1080p, false, ""},
		{"unknown cand skip", plugin.Resolution1080p, plugin.ResolutionUnknown, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := compareResolution(tc.cur, tc.cand)
			if tc.want && c == nil {
				t.Fatal("expected conflict, got nil")
			}
			if !tc.want && c != nil {
				t.Fatalf("expected no conflict, got %v", c)
			}
			if tc.want && c.Severity != tc.wantSev {
				t.Errorf("severity: got %q, want %q", c.Severity, tc.wantSev)
			}
		})
	}
}

func TestCompare_HDR(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cur  plugin.HDRFormat
		cand plugin.HDRFormat
		want bool
	}{
		{"DV→HDR10 warning", plugin.HDRDolbyVision, plugin.HDRHDR10, true},
		{"DV→SDR warning", plugin.HDRDolbyVision, plugin.HDRNone, true},
		{"HDR10→SDR warning", plugin.HDRHDR10, plugin.HDRNone, true},
		{"SDR→SDR no conflict", plugin.HDRNone, plugin.HDRNone, false},
		{"SDR→HDR10 no conflict", plugin.HDRNone, plugin.HDRHDR10, false},
		{"HDR10→DV no conflict", plugin.HDRHDR10, plugin.HDRDolbyVision, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := compareHDR(tc.cur, tc.cand)
			if tc.want && c == nil {
				t.Fatal("expected conflict, got nil")
			}
			if !tc.want && c != nil {
				t.Fatalf("expected no conflict, got %v", c)
			}
			if tc.want && c.Severity != SeverityWarning {
				t.Errorf("HDR conflicts should be warning severity, got %q", c.Severity)
			}
		})
	}
}

func TestCompare_HDR_SummaryText(t *testing.T) {
	t.Parallel()
	c := compareHDR(plugin.HDRDolbyVision, plugin.HDRNone)
	if c == nil {
		t.Fatal("expected conflict")
	}
	if c.Summary != "HDR lost: Dolby Vision → SDR" {
		t.Errorf("summary: got %q", c.Summary)
	}

	c = compareHDR(plugin.HDRDolbyVision, plugin.HDRHDR10)
	if c == nil {
		t.Fatal("expected conflict")
	}
	if c.Summary != "HDR downgrade: Dolby Vision → HDR10" {
		t.Errorf("summary: got %q", c.Summary)
	}
}

func TestCompare_AudioCodec(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cur     plugin.AudioCodec
		cand    plugin.AudioCodec
		want    bool
		wantSev string
	}{
		{"TrueHD Atmos→AC3 warning", plugin.AudioCodecTrueHDAtmos, plugin.AudioCodecAC3, true, SeverityWarning},
		{"TrueHD→EAC3 warning", plugin.AudioCodecTrueHD, plugin.AudioCodecEAC3, true, SeverityWarning},
		{"DTS-HD MA→DTS caution", plugin.AudioCodecDTSHDMA, plugin.AudioCodecDTS, true, SeverityWarning},
		{"EAC3→AC3 caution", plugin.AudioCodecEAC3, plugin.AudioCodecAC3, true, SeverityCaution},
		{"AC3→AAC caution", plugin.AudioCodecAC3, plugin.AudioCodecAAC, true, SeverityCaution},
		{"AC3→TrueHD no conflict", plugin.AudioCodecAC3, plugin.AudioCodecTrueHD, false, ""},
		{"same no conflict", plugin.AudioCodecEAC3, plugin.AudioCodecEAC3, false, ""},
		{"unknown cur skip", plugin.AudioCodecUnknown, plugin.AudioCodecAC3, false, ""},
		{"unknown cand skip", plugin.AudioCodecAC3, plugin.AudioCodecUnknown, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := compareAudioCodec(tc.cur, tc.cand)
			if tc.want && c == nil {
				t.Fatal("expected conflict, got nil")
			}
			if !tc.want && c != nil {
				t.Fatalf("expected no conflict, got %v", c)
			}
			if tc.want && c.Severity != tc.wantSev {
				t.Errorf("severity: got %q, want %q", c.Severity, tc.wantSev)
			}
		})
	}
}

func TestCompare_AudioChannels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cur     plugin.AudioChannels
		cand    plugin.AudioChannels
		want    bool
		wantSev string
	}{
		{"7.1→5.1 caution", plugin.AudioChannels71, plugin.AudioChannels51, true, SeverityCaution},
		{"7.1→2.0 warning", plugin.AudioChannels71, plugin.AudioChannels20, true, SeverityWarning},
		{"5.1→2.0 warning", plugin.AudioChannels51, plugin.AudioChannels20, true, SeverityWarning},
		{"5.1→7.1 no conflict", plugin.AudioChannels51, plugin.AudioChannels71, false, ""},
		{"same no conflict", plugin.AudioChannels51, plugin.AudioChannels51, false, ""},
		{"unknown skip", plugin.AudioChannelsUnknown, plugin.AudioChannels51, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := compareAudioChannels(tc.cur, tc.cand)
			if tc.want && c == nil {
				t.Fatal("expected conflict, got nil")
			}
			if !tc.want && c != nil {
				t.Fatalf("expected no conflict, got %v", c)
			}
			if tc.want && c.Severity != tc.wantSev {
				t.Errorf("severity: got %q, want %q", c.Severity, tc.wantSev)
			}
		})
	}
}

func TestCompare_Edition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cur  string
		cand string
		want bool
		msg  string
	}{
		{"lost", "Director's Cut", "", true, "Edition lost: Director's Cut → (none)"},
		{"changed", "Extended", "Theatrical", true, "Edition change: Extended → Theatrical"},
		{"same", "Extended", "Extended", false, ""},
		{"no current", "", "Extended", false, ""},
		{"both empty", "", "", false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := compareEdition(tc.cur, tc.cand)
			if tc.want && c == nil {
				t.Fatal("expected conflict, got nil")
			}
			if !tc.want && c != nil {
				t.Fatalf("expected no conflict, got %v", c)
			}
			if tc.want && c.Summary != tc.msg {
				t.Errorf("summary: got %q, want %q", c.Summary, tc.msg)
			}
		})
	}
}

func TestCompare_MixedUpgradeDowngrade(t *testing.T) {
	t.Parallel()
	// Video upgrade but audio downgrade — should detect the audio conflict.
	current := plugin.Quality{
		Resolution: plugin.Resolution720p, Source: plugin.SourceWEBDL,
		Codec: plugin.CodecX264, HDR: plugin.HDRNone,
		AudioCodec: plugin.AudioCodecTrueHDAtmos, AudioChannels: plugin.AudioChannels71,
	}
	candidate := plugin.Quality{
		Resolution: plugin.Resolution1080p, Source: plugin.SourceBluRay,
		Codec: plugin.CodecX265, HDR: plugin.HDRNone,
		AudioCodec: plugin.AudioCodecAC3, AudioChannels: plugin.AudioChannels51,
	}
	conflicts := Compare(current, candidate, "Extended", "")
	// Expect: audio codec downgrade, audio channels downgrade, edition lost.
	if len(conflicts) != 3 {
		t.Fatalf("expected 3 conflicts, got %d: %v", len(conflicts), summaries(conflicts))
	}
	dims := map[string]bool{}
	for _, c := range conflicts {
		dims[c.Dimension] = true
	}
	for _, d := range []string{"audio_codec", "audio_channels", "edition"} {
		if !dims[d] {
			t.Errorf("missing expected conflict for dimension %q", d)
		}
	}
}

func TestCompare_Codec(t *testing.T) {
	t.Parallel()
	c := compareCodec(plugin.CodecX265, plugin.CodecX264)
	if c == nil {
		t.Fatal("expected conflict for x265→x264")
	}
	if c.Severity != SeverityCaution {
		t.Errorf("severity: got %q, want caution", c.Severity)
	}

	c = compareCodec(plugin.CodecX264, plugin.CodecX265)
	if c != nil {
		t.Errorf("expected no conflict for x264→x265, got %v", c)
	}
}

func summaries(cs []Conflict) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Summary
	}
	return out
}
