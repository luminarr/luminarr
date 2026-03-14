package edition_test

import (
	"testing"

	"github.com/luminarr/luminarr/internal/core/edition"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string // empty string means expect nil (no edition detected)
	}{
		// ── Director's Cut ───────────────────────────────────────────────────
		{
			name:     "Directors Cut dot-separated",
			input:    "Blade.Runner.1982.Directors.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Director's Cut",
		},
		{
			name:     "Director's Cut with apostrophe",
			input:    "Blade.Runner.1982.Director's.Cut.1080p.BluRay.x265-GROUP",
			wantName: "Director's Cut",
		},
		{
			name:     "Directors Edition",
			input:    "Star.Trek.TMP.1979.Directors.Edition.1080p.BluRay.x265-GROUP",
			wantName: "Director's Cut",
		},

		// ── Extended ─────────────────────────────────────────────────────────
		{
			name:     "Extended Cut",
			input:    "Kingdom.of.Heaven.2005.Extended.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Extended",
		},
		{
			name:     "Extended Edition",
			input:    "The.Lord.of.the.Rings.2001.Extended.Edition.2160p.BluRay.REMUX.HEVC-GROUP",
			wantName: "Extended",
		},
		{
			name:     "bare Extended",
			input:    "The.Lord.of.the.Rings.2001.Extended.2160p.UHD.BluRay.REMUX.HDR.HEVC-GROUP",
			wantName: "Extended",
		},
		{
			name:     "Extended Version",
			input:    "Aliens.1986.Extended.Version.1080p.BluRay.x264-GROUP",
			wantName: "Extended",
		},

		// ── Theatrical ──────────────────────────────────────────────────────
		{
			name:     "Theatrical Cut",
			input:    "Donnie.Darko.2001.Theatrical.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Theatrical",
		},
		{
			name:     "bare Theatrical",
			input:    "Zack.Snyders.Justice.League.2021.Theatrical.2160p.WEB-DL.x265-GROUP",
			wantName: "Theatrical",
		},

		// ── Unrated ─────────────────────────────────────────────────────────
		{
			name:     "Unrated",
			input:    "Bad.Santa.2003.Unrated.1080p.BluRay.x264-GROUP",
			wantName: "Unrated",
		},
		{
			name:     "Unrated Cut",
			input:    "Live.Free.or.Die.Hard.2007.Unrated.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Unrated",
		},
		{
			name:     "Uncensored",
			input:    "Movie.2020.Uncensored.1080p.WEB-DL.x264-GROUP",
			wantName: "Unrated",
		},

		// ── IMAX ────────────────────────────────────────────────────────────
		{
			name:     "IMAX",
			input:    "Justice.League.2021.IMAX.2160p.WEB-DL.DDP5.1.HDR.HEVC-GROUP",
			wantName: "IMAX",
		},
		{
			name:     "IMAX Edition",
			input:    "Batman.Begins.2005.IMAX.Edition.2160p.WEB-DL.DDP5.1.HEVC-GROUP",
			wantName: "IMAX",
		},

		// ── Final Cut ───────────────────────────────────────────────────────
		{
			name:     "The Final Cut",
			input:    "Blade.Runner.1982.The.Final.Cut.2160p.UHD.BluRay.x265-GROUP",
			wantName: "Final Cut",
		},
		{
			name:     "Final Cut no article",
			input:    "Movie.2020.Final.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Final Cut",
		},

		// ── Redux ───────────────────────────────────────────────────────────
		{
			name:     "Redux",
			input:    "Apocalypse.Now.1979.Redux.1080p.BluRay.x264-GROUP",
			wantName: "Redux",
		},

		// ── Remastered ──────────────────────────────────────────────────────
		{
			name:     "Remastered",
			input:    "Jaws.1975.Remastered.1080p.BluRay.x265-GROUP",
			wantName: "Remastered",
		},
		{
			name:     "4K Remastered",
			input:    "Jaws.1975.4K.Remastered.2160p.BluRay.x265-GROUP",
			wantName: "Remastered",
		},
		{
			name:     "Digitally Remastered",
			input:    "Movie.1990.Digitally.Remastered.1080p.BluRay.x264-GROUP",
			wantName: "Remastered",
		},

		// ── Special Edition ─────────────────────────────────────────────────
		{
			name:     "Special Edition",
			input:    "Aliens.1986.Special.Edition.1080p.BluRay.x265-GROUP",
			wantName: "Special Edition",
		},

		// ── Criterion ───────────────────────────────────────────────────────
		{
			name:     "Criterion Collection",
			input:    "Seven.Samurai.1954.Criterion.Collection.1080p.BluRay.x264-GROUP",
			wantName: "Criterion",
		},
		{
			name:     "bare Criterion",
			input:    "Stalker.1979.Criterion.1080p.BluRay.x265-GROUP",
			wantName: "Criterion",
		},

		// ── Ultimate ────────────────────────────────────────────────────────
		{
			name:     "Ultimate Cut",
			input:    "Watchmen.2009.Ultimate.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Ultimate",
		},
		{
			name:     "Ultimate Edition",
			input:    "Batman.v.Superman.2016.Ultimate.Edition.2160p.BluRay.x265-GROUP",
			wantName: "Ultimate",
		},

		// ── Anniversary ─────────────────────────────────────────────────────
		{
			name:     "Anniversary Edition",
			input:    "E.T.1982.Anniversary.Edition.1080p.BluRay.x264-GROUP",
			wantName: "Anniversary",
		},
		{
			name:     "25th Anniversary",
			input:    "Blade.Runner.1982.25th.Anniversary.1080p.BluRay.x264-GROUP",
			wantName: "Anniversary",
		},
		{
			name:     "40th Anniversary Edition",
			input:    "Alien.1979.40th.Anniversary.Edition.2160p.BluRay.x265-GROUP",
			wantName: "Anniversary",
		},

		// ── Rogue Cut ───────────────────────────────────────────────────────
		{
			name:     "Rogue Cut",
			input:    "X-Men.Days.of.Future.Past.2014.Rogue.Cut.1080p.BluRay.x264-GROUP",
			wantName: "Rogue Cut",
		},

		// ── Black and Chrome ────────────────────────────────────────────────
		{
			name:     "Black and Chrome",
			input:    "Mad.Max.Fury.Road.2015.Black.and.Chrome.1080p.BluRay.x264-GROUP",
			wantName: "Black and Chrome",
		},

		// ── Open Matte ──────────────────────────────────────────────────────
		{
			name:     "Open Matte",
			input:    "The.Shining.1980.Open.Matte.1080p.BluRay.x264-GROUP",
			wantName: "Open Matte",
		},

		// ── No edition detected ─────────────────────────────────────────────
		{
			name:     "no edition standard release",
			input:    "Movie.2020.1080p.BluRay.x264-GROUP",
			wantName: "",
		},
		{
			name:     "no edition 4K release",
			input:    "Dune.2021.2160p.BluRay.x265.10bit.HDR-GROUP",
			wantName: "",
		},
		{
			name:     "no edition WEB-DL",
			input:    "The.Batman.2022.1080p.WEB-DL.DDP5.1.x265-GROUP",
			wantName: "",
		},

		// ── False positive guards ───────────────────────────────────────────
		{
			name:     "DC abbreviation NOT matched",
			input:    "Movie.2020.DC.1080p.BluRay.x264-GROUP",
			wantName: "",
		},
		{
			name:     "DC in title NOT matched",
			input:    "DC.League.of.Super-Pets.2022.1080p.BluRay.x264-GROUP",
			wantName: "",
		},
		{
			name:     "SE abbreviation NOT matched",
			input:    "Movie.2020.SE.1080p.BluRay.x264-GROUP",
			wantName: "",
		},
		{
			name:     "CC abbreviation NOT matched",
			input:    "Movie.2020.CC.1080p.BluRay.x264-GROUP",
			wantName: "",
		},

		// ── Edition with various separators ──────────────────────────────────
		{
			name:     "underscore separators",
			input:    "Blade_Runner_1982_Directors_Cut_1080p_BluRay_x264-GROUP",
			wantName: "Director's Cut",
		},
		{
			name:     "space separators",
			input:    "Blade Runner 1982 Directors Cut 1080p BluRay x264-GROUP",
			wantName: "Director's Cut",
		},
		{
			name:     "mixed separators",
			input:    "Blade.Runner 1982.Directors_Cut.1080p.BluRay.x264-GROUP",
			wantName: "Director's Cut",
		},

		// ── Edition position variants ───────────────────────────────────────
		{
			name:     "edition after quality tokens",
			input:    "Movie.2020.2160p.Extended.BluRay.x265-GROUP",
			wantName: "Extended",
		},
		{
			name:     "edition before year",
			input:    "Apocalypse.Now.Redux.1979.1080p.BluRay.x264-GROUP",
			wantName: "Redux",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := edition.Parse(tc.input)

			if tc.wantName == "" {
				if got != nil {
					t.Errorf("Parse(%q) = %q, want nil", tc.input, got.Name)
				}
				return
			}

			if got == nil {
				t.Fatalf("Parse(%q) = nil, want %q", tc.input, tc.wantName)
			}
			if got.Name != tc.wantName {
				t.Errorf("Parse(%q).Name = %q, want %q", tc.input, got.Name, tc.wantName)
			}
			if got.Raw == "" {
				t.Errorf("Parse(%q).Raw is empty, want non-empty", tc.input)
			}
		})
	}
}

func TestCanonical(t *testing.T) {
	t.Parallel()

	names := edition.Canonical()
	if len(names) == 0 {
		t.Fatal("Canonical() returned empty list")
	}
	// Verify no duplicates.
	seen := make(map[string]bool, len(names))
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate canonical name: %q", n)
		}
		seen[n] = true
	}
}
