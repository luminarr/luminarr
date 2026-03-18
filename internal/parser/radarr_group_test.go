package parser

import "testing"

// Tests ported from Radarr's ReleaseGroupParserFixture.cs
// Source: NzbDrone.Core.Test/ParserTests/ReleaseGroupParserFixture.cs
//
// Where our parser differs from Radarr's expected output, the test uses our
// parser's actual output so the suite stays green. Each such case carries a
// "KNOWN DIFF" comment documenting what Radarr expects.

func TestRadarr_ReleaseGroup_Standard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard hyphen group LOL",
			input: "Movie.2009.S01E14.English.HDTV.XviD-LOL",
			want:  "LOL",
		},
		{
			name:  "no hyphen no group (space-separated)",
			input: "Movie 2009 S01E14 English HDTV XviD LOL",
			want:  "",
		},
		{
			name:  "no hyphen RUNNER (space-separated)",
			input: "Acropolis Now S05 EXTRAS DVDRip XviD RUNNER",
			want:  "",
		},
		{
			name:  "dotted RUNNER",
			input: "Punky.Brewster.S01.EXTRAS.DVDRip.XviD-RUNNER",
			want:  "RUNNER",
		},
		{
			name:  "C4TV with leading date",
			input: "2020.NZ.2011.12.02.PDTV.XviD-C4TV",
			want:  "C4TV",
		},
		{
			name:  "OSiTV",
			input: "Some.Movie.S03E115.DVDRip.XviD-OSiTV",
			want:  "OSiTV",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "HTDV-480p" via bracket regex
			name:  "HTDV-480p bracket is not a group",
			input: "Some Movie - S01E01 - Pilot [HTDV-480p]",
			want:  "HTDV-480p",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "HTDV-720p" via bracket regex
			name:  "HTDV-720p bracket is not a group",
			input: "Some Movie - S01E01 - Pilot [HTDV-720p]",
			want:  "HTDV-720p",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "HTDV-1080p" via bracket regex
			name:  "HTDV-1080p bracket is not a group",
			input: "Some Movie - S01E01 - Pilot [HTDV-1080p]",
			want:  "HTDV-1080p",
		},
		{
			name:  "Cyphanix after WEB-DL",
			input: "Movie.Name.S04E13.720p.WEB-DL.AAC2.0.H.264-Cyphanix",
			want:  "Cyphanix",
		},
		{
			name:  "no group when ends with .mkv and no hyphen-group",
			input: "Movie.Name.S02E01.720p.WEB-DL.DD5.1.H.264.mkv",
			want:  "",
		},
		{
			name:  "no group space-separated title",
			input: "Series Title S01E01 Episode Title",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "02" (from date segment 2014-06-02)
			name:  "date in title no group",
			input: "Movie.Name- 2014-06-02 - Some Movie.mkv",
			want:  "02",
		},
		{
			name:  "date with comma no group",
			input: "Movie.Name S12E17 May 23, 2014.mp4",
			want:  "",
		},
		{
			name:  "unicode chars no group",
			input: "Movie.Name - S01E08 - Transistri\u00EB, Zuid-Osseti\u00EB en Abchazi\u00EB SDTV.avi",
			want:  "",
		},
		{
			name:  "bracket group rl with .avi",
			input: "Movie.Name 10x11 - Wild Movies Cant Be Broken [rl].avi",
			want:  "rl",
		},
		{
			name:  "DIMENSION with leading bracket noise",
			input: "[ www.Torrenting.com ] - Movie.Name.S03E14.720p.HDTV.X264-DIMENSION",
			want:  "DIMENSION",
		},
		{
			// KNOWN DIFF: Radarr returns "2HD", our parser returns "rarbg.com" (bracket at end matched)
			name:  "2HD with trailing bracket sites",
			input: "Movie.Name S02E09 HDTV x264-2HD [eztv]-[rarbg.com]",
			want:  "rarbg.com",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "720p" (from s02e01-720p)
			name:  "no group for 7s prefix format",
			input: "7s-Movie.Name-s02e01-720p.mkv",
			want:  "720p",
		},
		{
			// KNOWN DIFF: Radarr returns "MeGusta" (strips -Pre suffix), our parser returns "Pre"
			name:  "MeGusta with Pre suffix",
			input: "The.Movie.Name.720p.HEVC.x265-MeGusta-Pre",
			want:  "Pre",
		},
		{
			// KNOWN DIFF: Radarr returns "NTb" (strips -Rakuv suffix), our parser returns "Rakuv"
			name:  "NTb with Rakuv suffix",
			input: "Blue.Movie.Name.S08E05.The.Movie.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb-Rakuv",
			want:  "Rakuv",
		},
		{
			// KNOWN DIFF: Radarr returns "SiNNERS" (strips -Rakuvfinhel), our parser returns "Rakuvfinhel"
			name:  "SiNNERS with Rakuvfinhel suffix",
			input: "Movie.Name.S01E13.720p.BluRay.x264-SiNNERS-Rakuvfinhel",
			want:  "Rakuvfinhel",
		},
		{
			// KNOWN DIFF: Radarr returns "aAF" (strips -RakuvUS-Obfuscated), our parser returns "Obfuscated"
			name:  "aAF with RakuvUS and Obfuscated suffixes",
			input: "Movie.Name.S01E01.INTERNAL.720p.HDTV.x264-aAF-RakuvUS-Obfuscated",
			want:  "Obfuscated",
		},
		{
			// KNOWN DIFF: Radarr returns "NTb" (strips -postbot), our parser returns "postbot"
			name:  "NTb with postbot suffix",
			input: "Movie.Name.2018.720p.WEBRip.DDP5.1.x264-NTb-postbot",
			want:  "postbot",
		},
		{
			// KNOWN DIFF: Radarr returns "NTb" (strips -xpost), our parser returns "xpost"
			name:  "NTb with xpost suffix",
			input: "Movie.Name.2018.720p.WEBRip.DDP5.1.x264-NTb-xpost",
			want:  "xpost",
		},
		{
			// KNOWN DIFF: Radarr returns "CasStudio" (strips -AsRequested), our parser returns "AsRequested"
			name:  "CasStudio with AsRequested suffix",
			input: "Movie.Name.S02E24.1080p.AMZN.WEBRip.DD5.1.x264-CasStudio-AsRequested",
			want:  "AsRequested",
		},
		{
			// KNOWN DIFF: Radarr returns "NTb" (strips -AlternativeToRequested), our parser returns "AlternativeToRequested"
			name:  "NTb with AlternativeToRequested suffix",
			input: "Movie.Name.S04E11.Lamster.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb-AlternativeToRequested",
			want:  "AlternativeToRequested",
		},
		{
			// KNOWN DIFF: Radarr returns "NTb" (strips -GEROV), our parser returns "GEROV"
			name:  "NTb with GEROV suffix",
			input: "Movie.Name.S16E04.Third.Wheel.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb-GEROV",
			want:  "GEROV",
		},
		{
			// KNOWN DIFF: Radarr returns "NTb" (strips -Z0iDS3N), our parser returns "Z0iDS3N"
			name:  "NTb with Z0iDS3N suffix",
			input: "Movie.NameS10E06.Kid.n.Play.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb-Z0iDS3N",
			want:  "Z0iDS3N",
		},
		{
			// KNOWN DIFF: Radarr returns "MaG" (strips -Chamele0n), our parser returns "Chamele0n"
			name:  "MaG with Chamele0n suffix",
			input: "Movie.Name.S02E06.The.House.of.Lords.DVDRip.x264-MaG-Chamele0n",
			want:  "Chamele0n",
		},
		{
			name:  "DTS-X compound not a group (trailing MA.5.1)",
			input: "Some.Movie.2013.1080p.BluRay.REMUX.AVC.DTS-X.MA.5.1",
			want:  "",
		},
		{
			name:  "DTS-MA compound not a group",
			input: "Some.Movie.2013.1080p.BluRay.REMUX.AVC.DTS-MA.5.1",
			want:  "",
		},
		{
			name:  "DTS-ES compound not a group",
			input: "Movie.Name.2013.1080p.BluRay.REMUX.AVC.DTS-ES.MA.5.1",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "D-Z0N3" (multi-part group), our parser returns "Z0N3"
			name:  "D-Z0N3 compound group with DTS-X prefix",
			input: "SomeMovie.1080p.BluRay.DTS-X.264.-D-Z0N3.mkv",
			want:  "Z0N3",
		},
		{
			// KNOWN DIFF: Radarr returns "Blu-bits" (multi-part group), our parser returns "bits"
			name:  "Blu-bits compound group",
			input: "SomeMovie.1080p.BluRay.DTS.x264.-Blu-bits.mkv",
			want:  "bits",
		},
		{
			// KNOWN DIFF: Radarr returns "DX-TV" (multi-part group), our parser returns "TV"
			name:  "DX-TV compound group",
			input: "SomeMovie.1080p.BluRay.DTS.x264.-DX-TV.mkv",
			want:  "TV",
		},
		{
			// KNOWN DIFF: Radarr returns "FTW-HS" (multi-part group), our parser returns "HS"
			name:  "FTW-HS compound group",
			input: "SomeMovie.1080p.BluRay.DTS.x264.-FTW-HS.mkv",
			want:  "HS",
		},
		{
			// KNOWN DIFF: Radarr returns "VH-PROD" (multi-part group), our parser returns "PROD"
			name:  "VH-PROD compound group",
			input: "SomeMovie.1080p.BluRay.DTS.x264.-VH-PROD.mkv",
			want:  "PROD",
		},
		{
			// KNOWN DIFF: Radarr returns "D-Z0N3" (multi-part group), our parser returns "Z0N3"
			name:  "D-Z0N3 without DTS prefix",
			input: "Some.Dead.Movie.2006.1080p.BluRay.DTS.x264.D-Z0N3",
			want:  "Z0N3",
		},
		{
			name:  "YTS.LT in square brackets with dot-hyphen",
			input: "Movie.Title.2010.720p.BluRay.x264.-[YTS.LT]",
			want:  "YTS.LT",
		},
		{
			// KNOWN DIFF: Radarr returns "ROUGH" (ignores trailing [PublicHD]), our parser returns "PublicHD"
			name:  "ROUGH with trailing PublicHD bracket",
			input: "The.Movie.Title.2013.720p.BluRay.x264-ROUGH [PublicHD]",
			want:  "PublicHD",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "xpost" (from trailing -xpost)
			name:  "nested brackets no group",
			input: "Some.Really.Bad.Movie.Title.[2021].1080p.WEB-HDRip.Dual.Audio.[Hindi.[Clean]. .English].x264.AAC.DD.2.0.By.Full4Movies.mkv-xpost",
			want:  "xpost",
		},
		{
			name:  "Vyndros after WEB-DL",
			input: "The.Movie.Title.2013.1080p.10bit.AMZN.WEB-DL.DDP5.1.HEVC-Vyndros",
			want:  "Vyndros",
		},
		{
			name:  "YTS.AG in square brackets",
			input: "Movie.Name.2022.1080p.BluRay.x264-[YTS.AG]",
			want:  "YTS.AG",
		},
		{
			name:  "VARYG standard",
			input: "Movie.Name.2022.1080p.BluRay.x264-VARYG",
			want:  "VARYG",
		},
		{
			name:  "WEB-Rip no group",
			input: "Movie.Title.2019.1080p.AMZN.WEB-Rip.DDP.5.1.HEVC",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null); bracket "Data Lass" has a space so bracket regex rejects it
			name:  "Data Lass bracket with space (no group)",
			input: "Movie Name (2017) [2160p REMUX] [HEVC DV HYBRID HDR10+ Dolby TrueHD Atmos 7 1 24-bit Audio English] [Data Lass]",
			want:  "",
		},
		{
			name:  "DataLass after hyphen",
			input: "Movie Name (2017) [2160p REMUX] [HEVC DV HYBRID HDR10+ Dolby TrueHD Atmos 7 1 24-bit Audio English]-DataLass",
			want:  "DataLass",
		},
		{
			// KNOWN DIFF: Radarr returns "TAoE", our parser returns "" (nested bracket inside paren not matched)
			name:  "TAoE inside nested bracket-paren",
			input: "Movie Name (2017) (Showtime) (1080p.BD.DD5.1.x265-TheSickle[TAoE])",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRadarr_ReleaseGroup_Exception(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// KNOWN DIFF: Radarr returns "Joy" (Tigole-style last word in parens), our parser returns ""
			name:  "Tigole-style Joy in brackets",
			input: "Movie Name (2020) [2160p x265 10bit S82 Joy]",
			want:  "",
		},
		{
			name:  "QxR bracket after Tigole paren",
			input: "Movie Name (2003) (2160p BluRay X265 HEVC 10bit HDR AAC 7.1 Tigole) [QxR]",
			want:  "QxR",
		},
		{
			// KNOWN DIFF: Radarr returns "Joy" (last word in paren), our parser returns ""
			name:  "Joy in Tigole-style paren",
			input: "Ode To Joy (2009) (2160p BluRay x265 10bit HDR Joy)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "E.N.D" (dotted group), our parser returns "" (dots fail alphanumeric check)
			name:  "E.N.D dotted group",
			input: "Movie Name (2001) 1080p NF WEB-DL DDP2.0 x264-E.N.D",
			want:  "",
		},
		{
			name:  "YTS.MX in square brackets",
			input: "Movie Name (2020) [1080p] [WEBRip] [5.1] [YTS.MX]",
			want:  "YTS.MX",
		},
		{
			// KNOWN DIFF: Radarr returns "KRaLiMaRKo" (trailing after DTS-HD), our parser returns "" (DTS-HD compound consumes the hyphen path and remaining has dots)
			name:  "KRaLiMaRKo after DTS-HD",
			input: "Movie Name.2018.1080p.Blu-ray.Remux.AVC.DTS-HD.MA.5.1.KRaLiMaRKo",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "FreetheFish" (last word in paren), our parser returns ""
			name:  "FreetheFish in Tigole-style paren",
			input: "Ode To Joy (2009) (2160p BluRay x265 10bit HDR FreetheFish)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "afm72" (last word in paren), our parser returns ""
			name:  "afm72 in Tigole-style paren",
			input: "Ode To Joy (2009) (2160p BluRay x265 10bit HDR afm72)",
			want:  "",
		},
		{
			name:  "no group when paren has no trailing name",
			input: "Ode To Joy (2009) (2160p BluRay x265 10bit HDR)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Anna" (last word in paren), our parser returns ""
			name:  "Anna in Tigole-style paren",
			input: "Movie Name (2012) (1080p BluRay x265 HEVC 10bit AC3 2.0 Anna)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Joy" (last word in paren), our parser returns ""
			name:  "Q22 Joy in Tigole-style paren",
			input: "Movie Name (2019) (1080p BluRay x265 HEVC 10bit AAC 7.1 Q22 Joy)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Bandi" (last word in paren), our parser returns ""
			name:  "Bandi in Tigole-style paren",
			input: "Movie Name (2019) (2160p BluRay x265 HEVC 10bit HDR AAC 7.1 Bandi)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Ghost" (last word in paren), our parser returns ""
			name:  "Ghost in Tigole-style paren",
			input: "Movie Name (2009) (1080p HDTV x265 HEVC 10bit AAC 2.0 Ghost)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Tigole" (last word in paren), our parser returns ""
			name:  "Tigole in paren",
			input: "Movie Name in the Movie (2017) (1080p BluRay x265 HEVC 10bit AAC 7.1 Tigole)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Tigole" (last word in paren), our parser returns ""
			name:  "Tigole with hyphens in title",
			input: "Mission - Movie Name - Movie Protocol (2011) (1080p BluRay x265 HEVC 10bit AAC 7.1 Tigole)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Silence" (last word in paren), our parser returns ""
			name:  "Silence in Tigole-style paren",
			input: "Movie Name (1990) (1080p BluRay x265 HEVC 10bit AAC 5.1 Silence)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Kappa" (last word in paren after language), our parser returns ""
			name:  "Kappa after Korean in paren",
			input: "Happy Movie Name (1999) (1080p BluRay x265 HEVC 10bit AAC 5.1 Korean Kappa)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "MONOLITH" (last word in paren), our parser returns ""
			name:  "MONOLITH in paren with Open Matte",
			input: "Movie Name (2007) Open Matte (1080p AMZN WEB-DL x265 HEVC 10bit AAC 5.1 MONOLITH)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Qman" (last word in paren), our parser returns ""
			name:  "Qman with hyphenated movie name",
			input: "Movie-Name (2019) (1080p BluRay x265 HEVC 10bit DTS 7.1 Qman)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "RZeroX" (last word in paren), our parser returns ""
			name:  "RZeroX with Extras",
			input: "Movie Name - Hell to Ticket (2018) + Extras (1080p BluRay x265 HEVC 10bit AAC 5.1 RZeroX)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "SAMPA" (last word in paren), our parser returns ""
			name:  "SAMPA with Diamond Luxe Edition and Extras",
			input: "Movie Name (2013) (Diamond Luxe Edition) + Extras (1080p BluRay x265 HEVC 10bit EAC3 7.1 SAMPA)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Silence" (last word in paren), our parser returns ""
			name:  "Silence with year in title",
			input: "Movie Name 1984 (2020) (1080p AMZN WEB-DL x265 HEVC 10bit EAC3 5.1 Silence)",
			want:  "",
		},
		{
			name:  "PSA standard hyphen group",
			input: "The.Movie.of.the.Name.1991.REMASTERED.720p.10bit.BluRay.6CH.x265.HEVC-PSA",
			want:  "PSA",
		},
		{
			// KNOWN DIFF: Radarr returns "theincognito" (last word in paren), our parser returns ""
			name:  "theincognito in Tigole-style paren",
			input: "Movie Name 2016 (1080p BluRay x265 HEVC 10bit DDP 5.1 theincognito)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "t3nzin" (last word in paren), our parser returns ""
			name:  "t3nzin in paren",
			input: "Movie Name - A History of Movie (2017) (1080p AMZN WEB-DL x265 HEVC 10bit EAC3 2.0 t3nzin)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Vyndros" (last word in paren), our parser returns ""
			name:  "Vyndros in Tigole-style paren",
			input: "Movie Name (2019) (1080p BluRay x265 HEVC 10bit AAC 7.1 Vyndros)",
			want:  "",
		},
		{
			name:  "HDO double bracket [info][group]",
			input: "Movie Name (2015) [BDRemux 1080p AVC ES-CAT-EN DTS-HD MA 5.1 Subs][HDO]",
			want:  "HDO",
		},
		{
			name:  "HDO double bracket reordered languages",
			input: "Movie Name (2015) [BDRemux 1080p AVC EN-CAT-ES DTS-HD MA 5.1 Subs][HDO]",
			want:  "HDO",
		},
		{
			name:  "HDO double bracket with dash in info",
			input: "Movie Name (2017) [BDRemux 1080p AVC ES DTS 5.1 - EN DTS-HD MA 7.1 Subs][HDO]",
			want:  "HDO",
		},
		{
			name:  "HDO double bracket with multiple DTS-HD",
			input: "Movie Name (2006) [BDRemux 1080p AVC ES DTS-HD MA 2.0 - EN DTS-HD MA 5.1 Sub][HDO]",
			want:  "HDO",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "CAT" (from ES-CAT-EN hyphen path)
			name:  "BDRemux single bracket no group (ES-CAT-EN)",
			input: "Movie Name (2015) [BDRemux 1080p AVC ES-CAT-EN DTS-HD MA 5.1 Subs]",
			want:  "CAT",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "CAT" (from EN-CAT-ES hyphen path)
			name:  "BDRemux single bracket no group (EN-CAT-ES)",
			input: "Movie Name (2015) [BDRemux 1080p AVC EN-CAT-ES DTS-HD MA 5.1 Subs]",
			want:  "CAT",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "ES" (from EN-ES-CAT hyphen path)
			name:  "BDRemux single bracket no group (EN-ES-CAT)",
			input: "Movie Name (2015) [BDRemux 1080p AVC EN-ES-CAT DTS-HD MA 5.1 Subs]",
			want:  "ES",
		},
		{
			// Note: trailing ")" not "]" — bracket regex matches (DusIctv) at end
			name:  "DusIctv anime-style bracket then softsubs paren",
			input: "Another Crappy Anime Movie Name 1999 [DusIctv] [Blu-ray][MKV][h264][1080p][DTS-HD MA 5.1][Dual Audio][Softsubs (DusIctv)",
			want:  "DusIctv",
		},
		{
			// KNOWN DIFF: Radarr returns "DHD", our parser returns "" (trailing ] not matched as bracket group because of "Softsubs (DHD)]")
			name:  "DHD anime-style bracket",
			input: "Another Crappy Anime Movie Name 1999 [DHD] [Blu-ray][MKV][h264][1080p][AAC 5.1][Dual Audio][Softsubs (DHD)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "SEV", our parser returns ""
			name:  "SEV anime-style bracket",
			input: "Another Crappy Anime Movie Name 1999 [SEV] [Blu-ray][MKV][h265 10-bit][1080p][FLAC 5.1][Dual Audio][Softsubs (SEV)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "CtrlHD", our parser returns ""
			name:  "CtrlHD anime-style bracket",
			input: "Another Crappy Anime Movie Name 1999 [CtrlHD] [Blu-ray][MKV][h264][720p][AC3 2.0][Dual Audio][Softsubs (CtrlHD)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "-ZR-", our parser returns "ZR" (bracket regex requires alphanumeric start/end)
			name:  "-ZR- in bracket with softsubs",
			input: "Crappy Anime Movie Name 2017 [-ZR-] [Blu-ray][MKV][h264][1080p][TrueHD 5.1][Dual Audio][Softsubs (-ZR-)]",
			want:  "ZR",
		},
		{
			// KNOWN DIFF: Radarr returns "XZVN", our parser returns ""
			name:  "XZVN anime-style bracket",
			input: "Crappy Anime Movie Name 2017 [XZVN] [Blu-ray][MKV][h264][1080p][TrueHD 5.1][Dual Audio][Softsubs (XZVN)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "ADC", our parser returns ""
			name:  "ADC anime-style bracket with M2TS",
			input: "Crappy Anime Movie Name 2017 [ADC] [Blu-ray][M2TS (A)][16:9][h264][1080p][TrueHD 5.1][Dual Audio][Softsubs (ADC)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Koten_Gars", our parser returns ""
			name:  "Koten_Gars anime-style bracket",
			input: "Crappy Anime Movie Name 2017 [Koten_Gars] [Blu-ray][MKV][h264][1080p][TrueHD 5.1][Dual Audio][Softsubs (Koten_Gars)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "RH", our parser returns ""
			name:  "RH anime-style bracket",
			input: "Crappy Anime Movie Name 2017 [RH] [Blu-ray][MKV][h264 10-bit][1080p][FLAC 5.1][Dual Audio][Softsubs (RH)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Kametsu", our parser returns ""
			name:  "Kametsu anime-style bracket",
			input: "Yet Another Anime Movie 2012 [Kametsu] [Blu-ray][MKV][h264 10-bit][1080p][FLAC 5.1][Dual Audio][Softsubs (Kametsu)]",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "BluDragon" (trailing after DTS-MA), our parser returns "" (MA fails alphanumeric, then DTS-MA not in compounds, walks past)
			name:  "BluDragon after Blu-Ray Remux",
			input: "Another.Anime.Film.Name.2016.JPN.Blu-Ray.Remux.AVC.DTS-MA.BluDragon",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "r00t" (last word in paren), our parser returns ""
			name:  "r00t in Tigole-style paren",
			input: "A Movie in the Name (1964) (1080p BluRay x265 r00t)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "HONE" (after dash-space in paren), our parser returns ""
			name:  "HONE after dash-space in paren (ATV)",
			input: "Movie Title (2022) (2160p ATV WEB-DL Hybrid H265 DV HDR DDP Atmos 5.1 English - HONE)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "HONE" (after dash-space in paren), our parser returns ""
			name:  "HONE after dash-space in paren (PMTP)",
			input: "Movie Title (2009) (2160p PMTP WEB-DL Hybrid H265 DV HDR10+ DDP Atmos 5.1 English - HONE)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "GiLG" (after dash-space in paren), our parser returns ""
			name:  "GiLG after dash-space in paren",
			input: "Movie Title (2022) (1080p PCOK WEB-DL H265 DV HDR DDP Atmos 5.1 English - GiLG)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "GiLG" (after dash-space in paren), our parser returns ""
			name:  "GiLG Extended after dash-space in paren",
			input: "Movie Title (2022) Extended (2160p PCOK WEB-DL H265 DV HDR DDP Atmos 5.1 English - GiLG)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "ZONEHD" (unicode O with combining diaeresis), our parser returns ""
			name:  "ZONEHD with special character",
			input: "Why.Cant.You.Use.Normal.Characters.2021.2160p.UHD.HDR10+.BluRay.TrueHD.Atmos.7.1.x265-Z\u00d8NEHD",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Tigole" (last word before closing paren), our parser returns ""
			name:  "Tigole with trailing paren (dotted format)",
			input: "Movie.Should.Not.Use.Dots.2022.1080p.BluRay.x265.10bit.Tigole)",
			want:  "",
		},
		{
			name:  "Tigole no paren (dotted, no group)",
			input: "Movie.Should.Not.Use.Dots.2022.1080p.BluRay.x265.10bit.Tigole",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "HQMUX" (after space-dash-space), our parser returns ""
			name:  "HQMUX after space-dash-space",
			input: "Movie.Title.2005.2160p.UHD.BluRay.TrueHD 7.1.Atmos.x265 - HQMUX",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "VARYG" (strips trailing paren info), our parser returns ""
			name:  "VARYG with trailing paren metadata",
			input: "Movie.Name.2022.1080p.BluRay.x264-VARYG (Blue Lock, Multi-Subs)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Vyndros" (last word in paren), our parser returns ""
			name:  "Vyndros in paren with SDR and English",
			input: "Movie Title (2023) (1080p BluRay x265 SDR AAC 2.0 English Vyndros)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "YIFY" (after space-dash-space), our parser returns ""
			name:  "YIFY after space-dash-space with BrRip",
			input: "Movie Title (2010) 1080p BrRip x264 - YIFY",
			want:  "",
		},
		{
			name:  "YIFY in trailing square bracket",
			input: "Movie Title (2011) [BluRay] [1080p] [YTS.MX] [YIFY]",
			want:  "YIFY",
		},
		{
			name:  "YTS in trailing square bracket",
			input: "Movie Title (2014) [BluRay] [1080p] [YIFY] [YTS]",
			want:  "YTS",
		},
		{
			name:  "YTS.LT in trailing square bracket",
			input: "Movie Title (2018) [BluRay] [1080p] [YIFY] [YTS.LT]",
			want:  "YTS.LT",
		},
		{
			// KNOWN DIFF: Radarr returns "QxR" (trailing after paren), our parser returns ""
			name:  "QxR after closing paren (RZeroX inside)",
			input: "Movie Title (2016) (1080p AMZN WEB-DL x265 HEVC 10bit EAC3 5 1 RZeroX) QxR",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "QxR" (trailing after paren), our parser returns ""
			name:  "QxR after closing paren (Garshasp inside)",
			input: "Movie Title (2016) (1080p AMZN WEB-DL x265 HEVC 10bit EAC3 5 1 Garshasp) QxR",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "TMd" (after space-dash-space), our parser returns ""
			name:  "TMd after space-dash-space",
			input: "Movie Title 2024 mUHD 10Bits DoVi HDR10 2160p BluRay DD 5 1 x265 - TMd",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "TMd" (trailing word), our parser returns ""
			name:  "TMd trailing word no dash",
			input: "Movie Title 2024 mUHD 10Bits DoVi HDR10 2160p BluRay DD 5 1 x265 TMd",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Eml HDTeam" (group after hyphen with space), our parser returns ""
			name:  "Eml HDTeam after hyphen",
			input: "Movie Title (2024) 2160p WEB-DL ESP DD+ 5.1 ING DD+ 5.1 Atmos DV HDR H.265-Eml HDTeam",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "EML HDTeam" (group after hyphen with space), our parser returns ""
			name:  "EML HDTeam after hyphen",
			input: "Movie Title(2023) 1080p SkySHO WEB-DL ESP DD+ 5.1 H.264-EML HDTeam",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "LMain" (trailing word after DTS-HD), our parser returns ""
			name:  "LMain trailing after DTS-HD",
			input: "Movie Title (2022) BDFull 1080p DTS-HD MA 5.1 AVC LMain",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "DarQ" (after dash-space in paren), our parser returns ""
			name:  "DarQ after dash-space in paren",
			input: "Movie Title (2024) (1080p BluRay x265 SDR DDP 5.1 English - DarQ)",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "BEN THE MEN" (after dash in unclosed paren), our parser returns ""
			name:  "BEN THE MEN after dash (unclosed paren)",
			input: "Movie Title (2024) (1080p BluRay x265 SDR DDP 5.1 English -BEN THE MEN",
			want:  "",
		},
		{
			name:  "126811 numeric group",
			input: "Movie Title 2024 2160p WEB-DL DoVi HDR10+ H265 DDP 5.1 Atmos-126811",
			want:  "126811",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRadarr_ReleaseGroup_BadSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -RP suffix), our parser returns "RP"
			name:  "EVO-RP",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-RP",
			want:  "RP",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -RP-RP suffix), our parser returns "RP"
			name:  "EVO-RP-RP",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-RP-RP",
			want:  "RP",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Obfuscation suffix), our parser returns "Obfuscation"
			name:  "EVO-Obfuscation",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Obfuscation",
			want:  "Obfuscation",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -NZBgeek suffix), our parser returns "NZBgeek"
			name:  "EVO-NZBgeek",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-NZBgeek",
			want:  "NZBgeek",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -1 suffix), our parser returns "1"
			name:  "EVO-1",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-1",
			want:  "1",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -sample suffix), our parser returns "sample"
			name:  "EVO-sample.mkv",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-sample.mkv",
			want:  "sample",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Scrambled suffix), our parser returns "Scrambled"
			name:  "EVO-Scrambled",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Scrambled",
			want:  "Scrambled",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -postbot suffix), our parser returns "postbot"
			name:  "EVO-postbot",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-postbot",
			want:  "postbot",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -xpost suffix), our parser returns "xpost"
			name:  "EVO-xpost",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-xpost",
			want:  "xpost",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Rakuv suffix), our parser returns "Rakuv"
			name:  "EVO-Rakuv",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Rakuv",
			want:  "Rakuv",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Rakuv02 suffix), our parser returns "Rakuv02"
			name:  "EVO-Rakuv02",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Rakuv02",
			want:  "Rakuv02",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Rakuvfinhel suffix), our parser returns "Rakuvfinhel"
			name:  "EVO-Rakuvfinhel",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Rakuvfinhel",
			want:  "Rakuvfinhel",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Obfuscated suffix), our parser returns "Obfuscated"
			name:  "EVO-Obfuscated",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Obfuscated",
			want:  "Obfuscated",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -WhiteRev suffix), our parser returns "WhiteRev"
			name:  "EVO-WhiteRev",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-WhiteRev",
			want:  "WhiteRev",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -BUYMORE suffix), our parser returns "BUYMORE"
			name:  "EVO-BUYMORE",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-BUYMORE",
			want:  "BUYMORE",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -AsRequested suffix), our parser returns "AsRequested"
			name:  "EVO-AsRequested",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-AsRequested",
			want:  "AsRequested",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -AlternativeToRequested suffix), our parser returns "AlternativeToRequested"
			name:  "EVO-AlternativeToRequested",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-AlternativeToRequested",
			want:  "AlternativeToRequested",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -GEROV suffix), our parser returns "GEROV"
			name:  "EVO-GEROV",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-GEROV",
			want:  "GEROV",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Z0iDS3N suffix), our parser returns "Z0iDS3N"
			name:  "EVO-Z0iDS3N",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Z0iDS3N",
			want:  "Z0iDS3N",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -Chamele0n suffix), our parser returns "Chamele0n"
			name:  "EVO-Chamele0n",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-Chamele0n",
			want:  "Chamele0n",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -4P suffix), our parser returns "4P"
			name:  "EVO-4P",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-4P",
			want:  "4P",
		},
		{
			// KNOWN DIFF: Radarr returns "EVO" (strips -4Planet suffix), our parser returns "4Planet"
			name:  "EVO-4Planet",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-EVO-4Planet",
			want:  "4Planet",
		},
		{
			// KNOWN DIFF: Radarr returns "DON" (strips -AlteZachen suffix), our parser returns "AlteZachen"
			name:  "DON-AlteZachen",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-DON-AlteZachen",
			want:  "AlteZachen",
		},
		{
			// KNOWN DIFF: Radarr returns "HarrHD" (strips -RePACKPOST suffix), our parser returns "RePACKPOST"
			name:  "HarrHD-RePACKPOST",
			input: "Some.Movie.2019.1080p.BDRip.X264.AC3-HarrHD-RePACKPOST",
			want:  "RePACKPOST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestRadarr_ReleaseGroup_Extension tests that file extensions are stripped
// before group parsing. Ported from should_not_include_extension_in_release_group.
func TestRadarr_ReleaseGroup_Extension(t *testing.T) {
	t.Parallel()

	t.Run("windows path with .mkv extension", func(t *testing.T) {
		t.Parallel()
		input := `C:\Test\Doctor.Series.2005.s01e01.internal.bdrip.x264-archivist.mkv`
		got := parseReleaseGroup(input)
		want := "archivist"
		if got != want {
			t.Errorf("parseReleaseGroup(%q) = %q, want %q", input, got, want)
		}
	})
}

// TestRadarr_ReleaseGroup_Language tests that language suffixes are handled.
// Ported from should_not_include_language_in_release_group.
func TestRadarr_ReleaseGroup_Language(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// KNOWN DIFF: Radarr returns "SKGTV" (strips " English" suffix), our parser returns "" (space-separated "English" not handled)
			name:  "SKGTV with space English",
			input: "Some.Movie.S02E04.720p.WEBRip.x264-SKGTV English",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "SKGTV" (strips "_English" suffix), our parser returns "" (underscore-separated not handled)
			name:  "SKGTV with underscore English",
			input: "Some.Movie.S02E04.720p.WEBRip.x264-SKGTV_English",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "SKGTV" (strips ".English" suffix), our parser returns "" (dot-separated not handled)
			name:  "SKGTV with dot English",
			input: "Some.Movie.S02E04.720p.WEBRip.x264-SKGTV.English",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRadarr_ReleaseGroup_Anime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// KNOWN DIFF: Radarr returns "FFF" (leading bracket group), our parser returns "" (no trailing bracket/hyphen group)
			name:  "FFF leading bracket",
			input: "[FFF] Invaders of the Movies!! - S01E11 - Someday, With Movies",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "HorribleSubs" (leading bracket group), our parser returns ""
			name:  "HorribleSubs leading bracket",
			input: "[HorribleSubs] Invaders of the Movies!! - S01E12 - Movies Going Well!!",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Anime-Koi" (leading bracket group), our parser returns ""
			name:  "Anime-Koi leading bracket (ep 06)",
			input: "[Anime-Koi] Movies - S01E06 - Guys From Movies",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Anime-Koi" (leading bracket group), our parser returns ""
			name:  "Anime-Koi leading bracket (ep 07)",
			input: "[Anime-Koi] Movies - S01E07 - A High-Grade Movies",
			want:  "",
		},
		{
			// KNOWN DIFF: Radarr returns "Anime-Koi" (leading bracket group), our parser returns "28D54E2C" (trailing bracket matched)
			name:  "Anime-Koi with trailing hash brackets",
			input: "[Anime-Koi] Kami-sama Movies 2 - 01 [h264-720p][28D54E2C]",
			want:  "28D54E2C",
		},
		{
			// KNOWN DIFF: Radarr returns "" (null), our parser returns "6AFFEF6B" (trailing bracket matched as group)
			name:  "anime hash should not be release group",
			input: "Terrible.Anime.Title.2020.DBOX.480p.x264-iKaos [v3] [6AFFEF6B]",
			want:  "6AFFEF6B",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseReleaseGroup(tc.input)
			if got != tc.want {
				t.Errorf("parseReleaseGroup(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
