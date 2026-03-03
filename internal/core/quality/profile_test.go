package quality_test

import (
	"testing"

	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func qSD() plugin.Quality {
	return plugin.Quality{Resolution: plugin.ResolutionSD, Source: plugin.SourceDVD, Codec: plugin.CodecXVID}
}

func q720p() plugin.Quality {
	return plugin.Quality{Resolution: plugin.Resolution720p, Source: plugin.SourceWEBDL, Codec: plugin.CodecX264}
}

func q1080pWEB() plugin.Quality {
	return plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceWEBDL, Codec: plugin.CodecX264}
}

func q1080pBluRay() plugin.Quality {
	return plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceBluRay, Codec: plugin.CodecX265}
}

func q1080pRemux() plugin.Quality {
	return plugin.Quality{Resolution: plugin.Resolution1080p, Source: plugin.SourceRemux, Codec: plugin.CodecX265}
}

func q2160pWEB() plugin.Quality {
	return plugin.Quality{Resolution: plugin.Resolution2160p, Source: plugin.SourceWEBDL, Codec: plugin.CodecX265}
}

func ptr(q plugin.Quality) *plugin.Quality { return &q }

// standardProfile builds a typical HD profile: accepts 720p WEB through
// 1080p Remux, cutoff at 1080p WEB-DL, upgrades allowed up to 1080p Remux.
func standardProfile(upgradeAllowed bool, upgradeUntil *plugin.Quality) *quality.Profile {
	return &quality.Profile{
		ID:     "hd",
		Name:   "HD",
		Cutoff: q1080pWEB(),
		Qualities: []plugin.Quality{
			q1080pRemux(),
			q1080pBluRay(),
			q1080pWEB(),
			q720p(),
		},
		UpgradeAllowed: upgradeAllowed,
		UpgradeUntil:   upgradeUntil,
	}
}

// ── WantRelease tests ─────────────────────────────────────────────────────────

func TestWantRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		profile        *quality.Profile
		releaseQuality plugin.Quality
		currentFile    *plugin.Quality
		want           bool
	}{
		// ── No file on disk ──────────────────────────────────────────────────
		{
			name:           "no file, allowed quality → grab",
			profile:        standardProfile(false, nil),
			releaseQuality: q1080pWEB(),
			currentFile:    nil,
			want:           true,
		},
		{
			name:           "no file, disallowed quality (SD) → reject",
			profile:        standardProfile(false, nil),
			releaseQuality: qSD(),
			currentFile:    nil,
			want:           false,
		},
		{
			name:           "no file, 2160p not in allowed list → reject",
			profile:        standardProfile(false, nil),
			releaseQuality: q2160pWEB(),
			currentFile:    nil,
			want:           false,
		},

		// ── Below cutoff ─────────────────────────────────────────────────────
		{
			name:           "below cutoff, release is better → grab",
			profile:        standardProfile(false, nil),
			releaseQuality: q1080pWEB(), // cutoff
			currentFile:    ptr(q720p()),
			want:           true,
		},
		{
			name:           "below cutoff, release equals current → grab (lateral is ok below cutoff)",
			profile:        standardProfile(false, nil),
			releaseQuality: q720p(),
			currentFile:    ptr(q720p()),
			want:           true,
		},
		{
			name:           "below cutoff, release is worse than current → reject",
			profile:        standardProfile(false, nil),
			releaseQuality: q720p(),
			currentFile:    ptr(q1080pWEB()),
			want:           false, // current already meets cutoff, upgrading off
		},

		// ── Cutoff met, upgrades disabled ────────────────────────────────────
		{
			name:           "cutoff met, upgrade disabled, better release → reject",
			profile:        standardProfile(false, nil),
			releaseQuality: q1080pBluRay(),
			currentFile:    ptr(q1080pWEB()),
			want:           false,
		},
		{
			name:           "cutoff met, upgrade disabled, same quality → reject",
			profile:        standardProfile(false, nil),
			releaseQuality: q1080pWEB(),
			currentFile:    ptr(q1080pWEB()),
			want:           false,
		},

		// ── Cutoff met, upgrades enabled, no ceiling ─────────────────────────
		{
			name:           "cutoff met, upgrade enabled, better release → grab",
			profile:        standardProfile(true, nil),
			releaseQuality: q1080pBluRay(),
			currentFile:    ptr(q1080pWEB()),
			want:           true,
		},
		{
			name:           "cutoff met, upgrade enabled, same quality → no grab",
			profile:        standardProfile(true, nil),
			releaseQuality: q1080pWEB(),
			currentFile:    ptr(q1080pWEB()),
			want:           false,
		},
		{
			name:           "cutoff met, upgrade enabled, worse release → no grab",
			profile:        standardProfile(true, nil),
			releaseQuality: q720p(),
			currentFile:    ptr(q1080pWEB()),
			want:           false,
		},
		{
			name:           "cutoff met, upgrade enabled, release not in list → reject",
			profile:        standardProfile(true, nil),
			releaseQuality: qSD(),
			currentFile:    ptr(q1080pWEB()),
			want:           false,
		},

		// ── Upgrade ceiling ──────────────────────────────────────────────────
		{
			name:    "upgrade ceiling not yet reached, better release → grab",
			profile: standardProfile(true, ptr(q1080pRemux())),
			// current = 1080p WEB (score < 1080p Remux ceiling)
			releaseQuality: q1080pBluRay(),
			currentFile:    ptr(q1080pWEB()),
			want:           true,
		},
		{
			name:    "upgrade ceiling reached, better release → no grab",
			profile: standardProfile(true, ptr(q1080pRemux())),
			// current already at ceiling
			releaseQuality: q1080pRemux(),
			currentFile:    ptr(q1080pRemux()),
			want:           false,
		},
		{
			name:           "upgrade ceiling reached, current exceeds ceiling → no grab",
			profile:        standardProfile(true, ptr(q1080pBluRay())),
			releaseQuality: q1080pRemux(),
			currentFile:    ptr(q1080pRemux()),
			want:           false,
		},
		{
			name:           "upgrade ceiling not reached, release at ceiling → grab",
			profile:        standardProfile(true, ptr(q1080pRemux())),
			releaseQuality: q1080pRemux(),
			currentFile:    ptr(q1080pWEB()),
			want:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.profile.WantRelease(tc.releaseQuality, tc.currentFile)
			if got != tc.want {
				t.Errorf("WantRelease() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ── IsUpgrade tests ───────────────────────────────────────────────────────────

func TestIsUpgrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		profile        *quality.Profile
		releaseQuality plugin.Quality
		currentQuality plugin.Quality
		want           bool
	}{
		{
			name:           "strictly better, no ceiling → upgrade",
			profile:        standardProfile(true, nil),
			releaseQuality: q1080pBluRay(),
			currentQuality: q1080pWEB(),
			want:           true,
		},
		{
			name:           "equal quality → not an upgrade",
			profile:        standardProfile(true, nil),
			releaseQuality: q1080pWEB(),
			currentQuality: q1080pWEB(),
			want:           false,
		},
		{
			name:           "worse quality → not an upgrade",
			profile:        standardProfile(true, nil),
			releaseQuality: q720p(),
			currentQuality: q1080pWEB(),
			want:           false,
		},
		{
			name:           "better, current below ceiling → upgrade",
			profile:        standardProfile(true, ptr(q1080pRemux())),
			releaseQuality: q1080pBluRay(),
			currentQuality: q1080pWEB(),
			want:           true,
		},
		{
			name:           "better, current at ceiling → no upgrade",
			profile:        standardProfile(true, ptr(q1080pRemux())),
			releaseQuality: q1080pRemux(),
			currentQuality: q1080pRemux(),
			want:           false,
		},
		{
			name:           "better, current above ceiling → no upgrade",
			profile:        standardProfile(true, ptr(q1080pBluRay())),
			releaseQuality: q1080pRemux(),
			currentQuality: q1080pRemux(),
			want:           false,
		},
		{
			name:           "current below ceiling by 2 levels → upgrade",
			profile:        standardProfile(true, ptr(q1080pRemux())),
			releaseQuality: q1080pWEB(),
			currentQuality: q720p(),
			want:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.profile.IsUpgrade(tc.releaseQuality, tc.currentQuality)
			if got != tc.want {
				t.Errorf("IsUpgrade() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ── AllowedQualities tests ────────────────────────────────────────────────────

func TestAllowedQualities(t *testing.T) {
	t.Parallel()

	p := standardProfile(false, nil)
	got := p.AllowedQualities()

	if len(got) != len(p.Qualities) {
		t.Fatalf("AllowedQualities() returned %d items, want %d", len(got), len(p.Qualities))
	}

	// Verify it's a copy, not a reference.
	got[0] = qSD()
	if p.Qualities[0].Score() == qSD().Score() {
		t.Error("AllowedQualities() returned a reference to internal slice — expected a copy")
	}
}
