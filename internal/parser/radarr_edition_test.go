package parser

import "testing"

// Tests ported from Radarr EditionParserFixture.cs.
// Our parser maps to canonical names; Radarr extracts raw edition text.
// Cases that Radarr detects but do not map to a canonical name expect "".

func TestRadarr_Edition_Positive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// --- Cases that map to a canonical edition ---

		{
			name:  "Directors Cut after year",
			input: "Movie Title 2012 Directors Cut",
			want:  "Director's Cut",
		},
		{
			name:  "Special Edition Remastered in parens after year",
			input: "Movie Title.2012.(Special.Edition.Remastered).[Bluray-1080p].mkv",
			want:  "Special Edition",
		},
		{
			name:  "Extended after year",
			input: "Movie Title 2012 Extended",
			want:  "Extended",
		},
		{
			name:  "Extended Directors Cut Fan Edit after year",
			input: "Movie Title 2012 Extended Directors Cut Fan Edit",
			want:  "Director's Cut",
		},
		{
			name:  "Director's Cut with apostrophe after year",
			input: "Movie Title 2012 Director's Cut",
			want:  "Director's Cut",
		},
		{
			name:  "Directors Cut without apostrophe after year",
			input: "Movie Title 2012 Directors Cut",
			want:  "Director's Cut",
		},
		{
			name:  "Extended Theatrical Version IMAX in parens after year",
			input: "Movie Title.2012.(Extended.Theatrical.Version.IMAX).BluRay.1080p.2012.asdf",
			want:  "IMAX",
		},
		{
			name:  "Director's Cut with apostrophe after paren year",
			input: "2021 A Movie (1968) Director's Cut .mkv",
			want:  "Director's Cut",
		},
		{
			name:  "Extended Directors Cut FanEdit in parens",
			input: "2021 A Movie 1968 (Extended Directors Cut FanEdit)",
			want:  "Director's Cut",
		},
		{
			name:  "Director's Cut after year no apostrophe in title",
			input: "Movie 2049 Director's Cut.mkv",
			want:  "Director's Cut",
		},
		{
			name:  "50th Anniversary Edition after year",
			input: "Movie Title 2012 50th Anniversary Edition.mkv",
			want:  "Anniversary",
		},
		{
			name:  "IMAX after year",
			input: "Movie 2012 IMAX.mkv",
			want:  "IMAX",
		},
		{
			name:  "Special Edition Fan Edit with dots before year",
			input: "Movie Title.Special.Edition.Fan Edit.2012..BRRip.x264.AAC-m2g",
			want:  "Special Edition",
		},
		{
			name:  "Special Edition Remastered in parens before year",
			input: "Movie Title.(Special.Edition.Remastered).2012.[Bluray-1080p].mkv",
			want:  "Special Edition",
		},
		{
			name:  "Extended before year",
			input: "Movie Title Extended 2012",
			want:  "Extended",
		},
		{
			name:  "Extended Directors Cut Fan Edit before year",
			input: "Movie Title Extended Directors Cut Fan Edit 2012",
			want:  "Director's Cut",
		},
		{
			name:  "Director's Cut with apostrophe before year",
			input: "Movie Title Director's Cut 2012",
			want:  "Director's Cut",
		},
		{
			name:  "Directors Cut without apostrophe before year",
			input: "Movie Title Directors Cut 2012",
			want:  "Director's Cut",
		},
		{
			name:  "Extended Theatrical Version IMAX in parens before year",
			input: "Movie Title.(Extended.Theatrical.Version.IMAX).2012.BluRay.1080p.asdf",
			want:  "IMAX",
		},
		{
			name:  "Director's Cut before paren year",
			input: "Movie Director's Cut (1968).mkv",
			want:  "Director's Cut",
		},
		{
			name:  "Extended Directors Cut FanEdit in parens before year",
			input: "2021 A Movie (Extended Directors Cut FanEdit) 1968 Bluray 1080p",
			want:  "Director's Cut",
		},
		{
			name:  "Director's Cut with four-digit title",
			input: "Movie Director's Cut 2049.mkv",
			want:  "Director's Cut",
		},
		{
			name:  "50th Anniversary Edition before year",
			input: "Movie Title 50th Anniversary Edition 2012.mkv",
			want:  "Anniversary",
		},
		{
			name:  "IMAX before year",
			input: "Movie IMAX 2012.mkv",
			want:  "IMAX",
		},
		{
			name:  "Final Cut before year",
			input: "Fake Movie Final Cut 2016",
			want:  "Final Cut",
		},
		{
			name:  "Final Cut after year",
			input: "Fake Movie 2016 Final Cut ",
			want:  "Final Cut",
		},
		{
			name:  "Extended Cut with language tag",
			input: "My Movie GERMAN Extended Cut 2016",
			want:  "Extended",
		},
		{
			name:  "Extended Cut with language tag dotted",
			input: "My.Movie.GERMAN.Extended.Cut.2016",
			want:  "Extended",
		},
		{
			name:  "Extended Cut with language tag no year",
			input: "My.Movie.GERMAN.Extended.Cut",
			want:  "Extended",
		},
		{
			name:  "Open Matte dotted",
			input: "Movie.1997.Open.Matte.1080p.BluRay.x264.DTS-FGT",
			want:  "Open Matte",
		},

		// --- Cases where Radarr detects an edition that we do NOT support ---

		{
			name:  "Despecialized after year in parens",
			input: "Movie Title 1999 (Despecialized).mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "Despecialized"
		},
		{
			name:  "Directors alone after year (Radarr: Directors)",
			input: "A Fake Movie 2035 2012 Directors.mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "Directors"
		},
		{
			name:  "2in1 after year",
			input: "Movie 2012 2in1.mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "2in1"
		},
		{
			name:  "Restored after year",
			input: "Movie 2012 Restored.mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "Restored"
		},
		{
			name:  "Despecialized in parens before year",
			input: "Movie Title (Despecialized) 1999.mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "Despecialized"
		},
		{
			name:  "Directors alone before year (Radarr: Directors)",
			input: "A Fake Movie 2035 Directors 2012.mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "Directors"
		},
		{
			name:  "2in1 before year",
			input: "Movie 2in1 2012.mkv",
			want:  "", // NOT SUPPORTED: Radarr detects "2in1"
		},
		{
			name:  "Assembly Cut dotted",
			input: "My.Movie.Assembly.Cut.1992.REPACK.1080p.BluRay.DD5.1.x264-Group",
			want:  "", // NOT SUPPORTED: Radarr detects "Assembly Cut"
		},
		{
			name:  "Ultimate Hunter Edition dotted",
			input: "Movie.1987.Ultimate.Hunter.Edition.DTS-HD.DTS.MULTISUBS.1080p.BluRay.x264.HQ-TUSAHD",
			want:  "", // NOT SUPPORTED: Radarr detects "Ultimate Hunter Edition"
		},
		{
			name:  "Diamond Edition dotted",
			input: "Movie.1950.Diamond.Edition.1080p.BluRay.x264-nikt0",
			want:  "", // NOT SUPPORTED: Radarr detects "Diamond Edition"
		},
		{
			name:  "Ultimate Rekall Edition dotted",
			input: "Movie.Title.1990.Ultimate.Rekall.Edition.NORDiC.REMUX.1080p.BluRay.AVC.DTS-HD.MA5.1-TWA",
			want:  "", // NOT SUPPORTED: Radarr detects "Ultimate Rekall Edition"
		},
		{
			name:  "Signature Edition dotted",
			input: "Movie.Title.1971.Signature.Edition.1080p.BluRay.FLAC.2.0.x264-TDD",
			want:  "", // NOT SUPPORTED: Radarr detects "Signature Edition"
		},
		{
			name:  "Imperial Edition dotted",
			input: "Movie.1979.The.Imperial.Edition.BluRay.720p.DTS.x264-CtrlHD",
			want:  "", // NOT SUPPORTED: Radarr detects "Imperial Edition"
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != tc.want {
				t.Errorf("Parse(%q).Edition = %q, want %q", tc.input, got.Edition, tc.want)
			}
		})
	}
}

func TestRadarr_Edition_Negative(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Holiday Special is not Special Edition",
			input: "Movie.Holiday.Special.1978.DVD.REMUX.DD.2.0-ViETNAM",
		},
		// KNOWN DIFF: Radarr's title-aware parser recognizes "Directors Cut" as
		// the movie title, not an edition. Our regex-based parser cannot
		// distinguish title words from edition tags without TMDB context.
		// Skipped: "Directors.Cut.German.2006.COMPLETE.PAL.DVDR-LoD"
		{
			name:  "Rogue in movie title not Rogue Cut",
			input: "Movie Impossible: Rogue Movie 2012 Bluray",
		},
		{
			name:  "TS source not edition",
			input: "Loving.Movie.2018.TS.FRENCH.MD.x264-DROGUERiE",
		},
		{
			name:  "Uncut is not Unrated",
			input: "Uncut.Movie.2019.720p.BluRay.x264-YOL0W",
		},
		{
			name:  "Christmas Edition is not Special Edition",
			input: "The.Christmas.Edition.1941.720p.HDTV.x264-CRiMSON",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(tc.input)
			if got.Edition != "" {
				t.Errorf("Parse(%q).Edition = %q, want empty (should not detect edition)", tc.input, got.Edition)
			}
		})
	}
}
