package quality_test

import (
	"testing"

	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func makeProfile(cutoff plugin.Quality) *quality.Profile {
	return &quality.Profile{
		ID:     "test",
		Name:   "Test",
		Cutoff: cutoff,
	}
}

func TestScoreBreakdown_perfectMatch(t *testing.T) {
	cutoff := plugin.Quality{
		Resolution: plugin.Resolution1080p,
		Source:     plugin.SourceBluRay,
		Codec:      plugin.CodecX265,
		HDR:        plugin.HDRNone,
	}
	release := cutoff // identical
	prof := makeProfile(cutoff)

	score, bd := prof.ScoreWithBreakdown(release)
	if score != 100 {
		t.Errorf("expected score 100, got %d", score)
	}
	if bd.Total != 100 {
		t.Errorf("expected bd.Total 100, got %d", bd.Total)
	}
	if len(bd.Dimensions) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(bd.Dimensions))
	}
	for _, d := range bd.Dimensions {
		if !d.Matched {
			t.Errorf("dimension %q should be matched", d.Name)
		}
		if d.Score != d.Max {
			t.Errorf("dimension %q: score %d != max %d", d.Name, d.Score, d.Max)
		}
	}
}

func TestScoreBreakdown_codecMismatch(t *testing.T) {
	cutoff := plugin.Quality{
		Resolution: plugin.Resolution1080p,
		Source:     plugin.SourceBluRay,
		Codec:      plugin.CodecX265, // want x265
		HDR:        plugin.HDRNone,
	}
	release := plugin.Quality{
		Resolution: plugin.Resolution1080p,
		Source:     plugin.SourceBluRay,
		Codec:      plugin.CodecX264, // got x264 — lower
		HDR:        plugin.HDRNone,
	}
	prof := makeProfile(cutoff)

	score, bd := prof.ScoreWithBreakdown(release)
	// Resolution(40) + Source(30) + HDR(10) = 80; codec mismatches → 0
	if score != 80 {
		t.Errorf("expected score 80, got %d", score)
	}

	var codecDim *plugin.ScoreDimension
	for i := range bd.Dimensions {
		if bd.Dimensions[i].Name == "codec" {
			codecDim = &bd.Dimensions[i]
		}
	}
	if codecDim == nil {
		t.Fatal("codec dimension missing")
	}
	if codecDim.Matched {
		t.Error("codec dimension should not be matched")
	}
	if codecDim.Score != 0 {
		t.Errorf("codec score should be 0, got %d", codecDim.Score)
	}
	if codecDim.Got != string(plugin.CodecX264) {
		t.Errorf("codec Got: got %q, want %q", codecDim.Got, plugin.CodecX264)
	}
	if codecDim.Want != string(plugin.CodecX265) {
		t.Errorf("codec Want: got %q, want %q", codecDim.Want, plugin.CodecX265)
	}
}

func TestScoreBreakdown_anyCodec(t *testing.T) {
	// Profile with empty codec cutoff means "any codec" — always matches.
	cutoff := plugin.Quality{
		Resolution: plugin.Resolution1080p,
		Source:     plugin.SourceBluRay,
		Codec:      "", // any
		HDR:        plugin.HDRNone,
	}
	release := plugin.Quality{
		Resolution: plugin.Resolution1080p,
		Source:     plugin.SourceBluRay,
		Codec:      plugin.CodecX264,
		HDR:        plugin.HDRNone,
	}
	prof := makeProfile(cutoff)

	score, bd := prof.ScoreWithBreakdown(release)
	// All 4 dimensions should match → 100
	if score != 100 {
		t.Errorf("expected score 100 (any codec), got %d", score)
	}
	for _, d := range bd.Dimensions {
		if !d.Matched {
			t.Errorf("dimension %q should be matched (any codec profile)", d.Name)
		}
	}
}

func TestScoreBreakdown_resolutionExceedsCutoff(t *testing.T) {
	// 4K release against a 1080p cutoff — exceeds → full resolution points.
	cutoff := plugin.Quality{
		Resolution: plugin.Resolution1080p,
		Source:     plugin.SourceBluRay,
		Codec:      plugin.CodecX265,
		HDR:        plugin.HDRNone,
	}
	release := plugin.Quality{
		Resolution: plugin.Resolution2160p, // exceeds cutoff
		Source:     plugin.SourceBluRay,
		Codec:      plugin.CodecX265,
		HDR:        plugin.HDRNone,
	}
	prof := makeProfile(cutoff)

	score, bd := prof.ScoreWithBreakdown(release)
	if score != 100 {
		t.Errorf("expected score 100 (4K exceeds 1080p cutoff), got %d", score)
	}
	var resDim *plugin.ScoreDimension
	for i := range bd.Dimensions {
		if bd.Dimensions[i].Name == "resolution" {
			resDim = &bd.Dimensions[i]
		}
	}
	if resDim == nil {
		t.Fatal("resolution dimension missing")
	}
	if !resDim.Matched {
		t.Error("resolution dimension should be matched (4K >= 1080p)")
	}
	if resDim.Score != 40 {
		t.Errorf("resolution score: got %d, want 40", resDim.Score)
	}
}
