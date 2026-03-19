package quality_test

import (
	"testing"

	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/pkg/plugin"
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

// ── RejectReason tests ────────────────────────────────────────────────────────

func TestRejectReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		profile        *quality.Profile
		releaseQuality plugin.Quality
		currentFile    *plugin.Quality
		wantReason     string
	}{
		{
			name:           "quality not in allowed set",
			profile:        standardProfile(false, nil),
			releaseQuality: qSD(), // SD not in HD profile
			currentFile:    nil,
			wantReason:     "quality_not_in_profile",
		},
		{
			name:           "no file on disk (nil current) — no rejection",
			profile:        standardProfile(false, nil),
			releaseQuality: q1080pWEB(),
			currentFile:    nil,
			wantReason:     "",
		},
		{
			// To test "below cutoff, release worse": need current below cutoff
			// but release worse than current. Use a profile with cutoff at
			// 1080p BluRay, current at 1080p WEB (below cutoff), release at
			// 720p (worse than current).
			name: "current below cutoff, release is worse than current",
			profile: &quality.Profile{
				ID:     "wide",
				Name:   "Wide",
				Cutoff: q1080pBluRay(),
				Qualities: []plugin.Quality{
					q1080pRemux(),
					q1080pBluRay(),
					q1080pWEB(),
					q720p(),
				},
				UpgradeAllowed: false,
			},
			releaseQuality: q720p(),
			currentFile:    ptr(q1080pWEB()), // below cutoff (1080p WEB < 1080p BluRay), but release 720p < current 1080p WEB
			wantReason:     "no_upgrade_needed",
		},
		{
			name: "current below cutoff, release is better — no rejection",
			profile: &quality.Profile{
				ID:     "wide",
				Name:   "Wide",
				Cutoff: q1080pBluRay(),
				Qualities: []plugin.Quality{
					q1080pRemux(),
					q1080pBluRay(),
					q1080pWEB(),
					q720p(),
				},
				UpgradeAllowed: false,
			},
			releaseQuality: q1080pBluRay(),
			currentFile:    ptr(q720p()), // below cutoff, release is better
			wantReason:     "",
		},
		{
			name:           "current meets cutoff, upgrades disabled — upgrade_disabled",
			profile:        standardProfile(false, nil),
			releaseQuality: q1080pBluRay(),
			currentFile:    ptr(q1080pWEB()), // meets cutoff
			wantReason:     "upgrade_disabled",
		},
		{
			name:           "current meets cutoff, upgrades enabled, release not better — no_upgrade_needed",
			profile:        standardProfile(true, nil),
			releaseQuality: q720p(),
			currentFile:    ptr(q1080pWEB()), // meets cutoff, release is worse
			wantReason:     "no_upgrade_needed",
		},
		{
			name:           "current meets cutoff, upgrades enabled, release is better — no rejection",
			profile:        standardProfile(true, nil),
			releaseQuality: q1080pBluRay(),
			currentFile:    ptr(q1080pWEB()), // meets cutoff, release is upgrade
			wantReason:     "",
		},
		{
			name: "empty qualities list (allow-any profile) — no rejection",
			profile: &quality.Profile{
				ID:             "any",
				Name:           "Any",
				Cutoff:         q720p(),
				Qualities:      nil, // empty = allow any
				UpgradeAllowed: false,
			},
			releaseQuality: qSD(),
			currentFile:    nil,
			wantReason:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.profile.RejectReason(tc.releaseQuality, tc.currentFile)
			if got != tc.wantReason {
				t.Errorf("RejectReason() = %q, want %q", got, tc.wantReason)
			}
		})
	}
}

// TestRejectReason_consistentWithWantRelease verifies that RejectReason returns
// "" if and only if WantRelease returns true, for a broad sweep of inputs.
func TestRejectReason_consistentWithWantRelease(t *testing.T) {
	t.Parallel()

	qualities := []plugin.Quality{qSD(), q720p(), q1080pWEB(), q1080pBluRay(), q1080pRemux(), q2160pWEB()}
	profiles := []*quality.Profile{
		standardProfile(false, nil),
		standardProfile(true, nil),
		standardProfile(true, ptr(q1080pRemux())),
	}

	for _, prof := range profiles {
		for _, release := range qualities {
			// nil current
			want := prof.WantRelease(release, nil)
			reason := prof.RejectReason(release, nil)
			if want && reason != "" {
				t.Errorf("profile=%s release=%v current=nil: WantRelease=true but RejectReason=%q", prof.Name, release, reason)
			}
			if !want && reason == "" {
				t.Errorf("profile=%s release=%v current=nil: WantRelease=false but RejectReason is empty", prof.Name, release)
			}
			// with various current files
			for _, current := range qualities {
				cur := current
				want := prof.WantRelease(release, &cur)
				reason := prof.RejectReason(release, &cur)
				if want && reason != "" {
					t.Errorf("profile=%s release=%v current=%v: WantRelease=true but RejectReason=%q", prof.Name, release, cur, reason)
				}
				if !want && reason == "" {
					t.Errorf("profile=%s release=%v current=%v: WantRelease=false but RejectReason is empty", prof.Name, release, cur)
				}
			}
		}
	}
}
