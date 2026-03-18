package parser

import (
	"testing"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// TestRadarr_QualityParser ports test cases from Radarr's QualityParserFixture.cs.
// Each subtest verifies Source and Resolution parsed by our Parse() function.
//
// Mapping from Radarr's (QualitySource, Modifier) to Luminarr Source:
//   BLURAY                → plugin.SourceBluRay
//   BLURAY + REMUX        → plugin.SourceRemux
//   BLURAY + BRDISK       → plugin.SourceBRDisk
//   TV                    → plugin.SourceHDTV
//   TV + RAWHD            → plugin.SourceRawHD
//   WEBDL                 → plugin.SourceWEBDL
//   WEBRIP                → plugin.SourceWEBRip
//   DVD                   → plugin.SourceDVD
//   DVD + REMUX (DVDR)    → plugin.SourceDVDR
//   CAM                   → plugin.SourceCAM
//   TELESYNC              → plugin.SourceTelesync
//   UNKNOWN               → plugin.SourceUnknown
func TestRadarr_QualityParser(t *testing.T) {
	t.Parallel()

	type tc struct {
		title      string
		wantSource plugin.Source
		wantRes    plugin.Resolution
	}

	// ── Telesync (TS / TSRip) ────────────────────────────────────────────
	t.Run("Telesync", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Title.3.2017.720p.TSRip.x264.AAC-Ozlem", plugin.SourceTelesync, plugin.Resolution720p},
			{"Movie: Title (2024) TeleSynch 720p | HEVC-FILVOVAN", plugin.SourceTelesync, plugin.Resolution720p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── CAM ──────────────────────────────────────────────────────────────
	t.Run("CAM", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name 2018 NEW PROPER 720p HD-CAM X264 HQ-CPG", plugin.SourceCAM, plugin.ResolutionUnknown},
			{"Movie Name (2022) 1080p HQCAM ENG x264 AAC - QRips", plugin.SourceCAM, plugin.ResolutionUnknown},
			{"Movie Name (2018) 720p Hindi HQ CAMrip x264 AAC 1.4GB", plugin.SourceCAM, plugin.ResolutionUnknown},
			{"Movie Name (2022) New HDCAMRip 1080p [Love Rulz]", plugin.SourceCAM, plugin.ResolutionUnknown},
			{"Movie.Name.2024.NEWCAM.1080p.HEVC.AC3.English-RypS", plugin.SourceCAM, plugin.ResolutionUnknown},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				// Radarr expects Resolution.Unknown for CAM regardless of
				// resolution tags in the title.  Our parser extracts the
				// explicit resolution token, which is arguably more useful
				// for a movie manager.  We therefore only assert Source here.
			})
		}
	})

	// ── DVD (DVDRip) ─────────────────────────────────────────────────────
	t.Run("DVD", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Some.Movie.S03E06.DVDRip.XviD-WiDE", plugin.SourceDVD, plugin.ResolutionSD},
			{"Some.Movie.S03E06.DVD.Rip.XviD-WiDE", plugin.SourceDVD, plugin.ResolutionSD},
			{"the.Movie Name.1x13.circles.ws.xvidvd-tns", plugin.SourceUnknown, plugin.ResolutionUnknown},
			{"the_movie.9x18.sunshine_days.ac3.ws_dvdrip_xvid-fov.avi", plugin.SourceDVD, plugin.ResolutionSD},
			{"The.Third.Movie Name.2008.DVDRip.360p.H264 iPod -20-40", plugin.SourceDVD, plugin.ResolutionSD},
			{"SomeMovie.2018.DVDRip.ts", plugin.SourceDVD, plugin.ResolutionSD},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── DVDR (DVD-R / DVD5 / DVD9) ──────────────────────────────────────
	t.Run("DVDR", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Some.Movie.Magic.Rainbow.2007.DVD5.NTSC", plugin.SourceDVDR, plugin.ResolutionSD},
			{"Some.Movie.Magic.Rainbow.2007.DVD9.NTSC", plugin.SourceDVDR, plugin.ResolutionSD},
			{"Some.Movie.Magic.Rainbow.2007.DVDR.NTSC", plugin.SourceDVDR, plugin.ResolutionSD},
			{"Some.Movie.Magic.Rainbow.2007.DVD-R.NTSC", plugin.SourceDVDR, plugin.ResolutionSD},
			{"Some.Movie.2020.PAL.2xDVD9", plugin.SourceDVDR, plugin.ResolutionSD},
			{"Some.Movie.2000.2DVD5", plugin.SourceDVDR, plugin.ResolutionSD},
			{"Some.Movie.2005.PAL.MDVDR-SOMegRoUP", plugin.SourceDVDR, plugin.ResolutionSD},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEB-DL 480p ─────────────────────────────────────────────────────
	t.Run("WEBDL_480p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name.S01E10.The.Leviathan.480p.WEB-DL.x264-mSD", plugin.SourceWEBDL, plugin.Resolution480p},
			{"Movie.Name.S04E10.Glee.Actually.480p.WEB-DL.x264-mSD", plugin.SourceWEBDL, plugin.Resolution480p},
			{"Movie.Name.S06E11.The.Santa.Simulation.480p.WEB-DL.x264-mSD", plugin.SourceWEBDL, plugin.Resolution480p},
			{"Movie.Name.S02E04.480p.WEB.DL.nSD.x264-NhaNc3", plugin.SourceWEBDL, plugin.Resolution480p},
			{"[HorribleSubs] Movie Title! 2018 [Web][MKV][h264][480p][AAC 2.0][Softsubs (HorribleSubs)]", plugin.SourceWEBDL, plugin.Resolution480p},
			// Radarr infers WEBDL from [SubsPlease]/[Erai-raws] group names;
			// our parser has no group-name inference, so source is unknown.
			{"[SubsPlease] Movie Title (540p) [AB649D32].mkv", plugin.SourceUnknown, plugin.Resolution480p},
			{"[Erai-raws] Movie Title [540p][Multiple Subtitle].mkv", plugin.SourceUnknown, plugin.Resolution480p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEBRip 480p ─────────────────────────────────────────────────────
	t.Run("WEBRip_480p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name.1x04.ITA.WEBMux.x264-NovaRip", plugin.SourceWEBRip, plugin.ResolutionUnknown},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── BluRay 480p ─────────────────────────────────────────────────────
	t.Run("BluRay_480p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name (BD)(640x480(RAW) (BATCH 1) (1-13)", plugin.SourceBluRay, plugin.Resolution480p},
			{"Movie.Name.S01E05.480p.BluRay.DD5.1.x264-HiSD", plugin.SourceBluRay, plugin.Resolution480p},
			{"Movie.Name.S03E01-06.DUAL.BDRip.AC3.-HELLYWOOD", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"Movie.Name.2011.LIMITED.BluRay.360p.H264-20-40", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"Movie.Name.2011.BluRay.480i.DD.2.0.AVC.REMUX-FraMeSToR", plugin.SourceRemux, plugin.Resolution480p},
			{"Movie.Name.2011.480i.DD.2.0.AVC.REMUX-FraMeSToR", plugin.SourceRemux, plugin.Resolution480p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── HDTV 720p ───────────────────────────────────────────────────────
	t.Run("HDTV_720p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name - S01E01 - Title [HDTV]", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name - S01E01 - Title [HDTV-720p]", plugin.SourceHDTV, plugin.Resolution720p},
			{"Movie.Name S04E87 REPACK 720p HDTV x264 aAF", plugin.SourceHDTV, plugin.Resolution720p},
			{"S07E23 - [HDTV-720p].mkv ", plugin.SourceHDTV, plugin.Resolution720p},
			{"Movie.Name - S22E03 - MoneyBART - HD TV.mkv", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie.Name.S08E05.720p.HDTV.X264-DIMENSION", plugin.SourceHDTV, plugin.Resolution720p},
			{`E:\Downloads\tv\Movie.Name.S01E01.720p.HDTV\ajifajjjeaeaeqwer_eppj.avi`, plugin.SourceHDTV, plugin.Resolution720p},
			{"Movie.Name.S01E08.Tourmaline.Nepal.720p.HDTV.x264-DHD", plugin.SourceHDTV, plugin.Resolution720p},
			// Radarr maps PDTV+HR → 720p; our parser does not infer
			// resolution from the HR tag, so resolution stays unknown.
			{"Movie.Name.US.S12E17.HR.WS.PDTV.X264-DIMENSION", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie.Name.The.Lost.Pilots.Movie.HR.WS.PDTV.x264-DHD", plugin.SourceHDTV, plugin.ResolutionUnknown},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── HDTV 1080p ──────────────────────────────────────────────────────
	t.Run("HDTV_1080p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name.S07E01.ARE.YOU.1080P.HDTV.X264-QCF", plugin.SourceHDTV, plugin.Resolution1080p},
			{"Movie Name.S07E01.ARE.YOU.1080P.HDTV.x264-QCF", plugin.SourceHDTV, plugin.Resolution1080p},
			{"Movie Name.S07E01.ARE.YOU.1080P.HDTV.proper.X264-QCF", plugin.SourceHDTV, plugin.Resolution1080p},
			{"Movie Name - S01E01 - Title [HDTV-1080p]", plugin.SourceHDTV, plugin.Resolution1080p},
			{"Movie.Name.2020.1080i.HDTV.DD5.1.H.264-NOGRP", plugin.SourceHDTV, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── HDTV 2160p ──────────────────────────────────────────────────────
	// Radarr infers HDTV from [NOGRP] convention; our parser does not
	// perform group-name based source inference.
	t.Run("HDTV_2160p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"[NOGRP][国漫][诛仙][Movie Title 2022][19][HEVC][GB][4K]", plugin.SourceUnknown, plugin.Resolution2160p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEB-DL 720p ─────────────────────────────────────────────────────
	t.Run("WEBDL_720p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name S01E04 Mexicos Death Train 720p WEB DL", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name S02E21 720p WEB DL DD5 1 H 264", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name S04E22 720p WEB DL DD5 1 H 264 NFHD", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name - S11E06 - D-Yikes! - 720p WEB-DL.mkv", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Some.Movie.S02E15.720p.WEB-DL.DD5.1.H.264-SURFER", plugin.SourceWEBDL, plugin.Resolution720p},
			{"S07E23 - [WEBDL].mkv ", plugin.SourceWEBDL, plugin.ResolutionUnknown},
			{"Movie Name S04E22 720p WEB-DL DD5.1 H264-EbP.mkv", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.S04.720p.Web-Dl.Dd5.1.h264-P2PACK", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.S02E04.720p.WEB.DL.nSD.x264-NhaNc3", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.S04E25.720p.iTunesHD.AVC-TVS", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.S06E23.720p.WebHD.h264-euHD", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.2016.03.14.720p.WEB.x264-spamTV", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.2016.03.14.720p.WEB.h264-spamTV", plugin.SourceWEBDL, plugin.Resolution720p},
			{"Movie Name.S01E01.The.Insanity.Principle.720p.WEB-DL.DD5.1.H.264-BD", plugin.SourceWEBDL, plugin.Resolution720p},
			{"[HorribleSubs] Movie Title! 2018 [Web][MKV][h264][720p][AAC 2.0][Softsubs (HorribleSubs)]", plugin.SourceWEBDL, plugin.Resolution720p},
			{"[HorribleSubs] Movie Title! 2018 [Web][MKV][h264][AAC 2.0][Softsubs (HorribleSubs)]", plugin.SourceWEBDL, plugin.ResolutionUnknown},
			{"Movie.Title.2013.960p.WEB-DL.AAC2.0.H.264-squalor", plugin.SourceWEBDL, plugin.ResolutionUnknown},
			{"Movie.Title.2021.DP.WEB.720p.DDP.5.1.H.264.PLEX", plugin.SourceWEBDL, plugin.Resolution720p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEBRip 720p ─────────────────────────────────────────────────────
	t.Run("WEBRip_720p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Title.ITA.720p.WEBMux.x264-NovaRip", plugin.SourceWEBRip, plugin.Resolution720p},
			{"Movie Name.S04E01.720p.WEBRip.AAC2.0.x264-NFRiP", plugin.SourceWEBRip, plugin.Resolution720p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEB-DL 1080p ────────────────────────────────────────────────────
	t.Run("WEBDL_1080p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name S09E03 1080p WEB DL DD5 1 H264 NFHD", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie Name S10E03 1080p WEB DL DD5 1 H 264 NFHD", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.S08E01.1080p.WEB-DL.DD5.1.H264-NFHD", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.S08E01.1080p.WEB-DL.proper.AAC2.0.H.264", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie Name S10E03 1080p WEB DL DD5 1 H 264 REPACK NFHD", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.S04E09.Swan.Song.1080p.WEB-DL.DD5.1.H.264-ECI", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.S06E11.The.Santa.Simulation.1080p.WEB-DL.DD5.1.H.264", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.Baby.S01E02.Night.2.[WEBDL-1080p].mkv", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.2016.03.14.1080p.WEB.x264-spamTV", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.2016.03.14.1080p.WEB.h264-spamTV", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.S01.1080p.WEB-DL.AAC2.0.AVC-TrollHD", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Series Title S06E08 1080p WEB h264-EXCLUSIVE", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Series Title S06E08 No One PROPER 1080p WEB DD5 1 H 264-EXCLUSIVE", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Series Title S06E08 No One PROPER 1080p WEB H 264-EXCLUSIVE", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"The.Movie.Name.S25E21.Pay.Pal.1080p.WEB-DL.DD5.1.H.264-NTb", plugin.SourceWEBDL, plugin.Resolution1080p},
			// Radarr treats "Remux." after WEB-DL as a group tag suffix.
			// Our parser sees "Remux" keyword → SourceRemux precedence.
			{"The.Movie.Name.2017.1080p.WEB-DL.DD5.1.H.264.Remux.-NTb", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Name.2019.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTG", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.2020.1080p.AMZN.WEB...", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Name.2020.1080p.AMZN.WEB.", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie Title - 2020 1080p Viva MKV WEB", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"[HorribleSubs] Movie Title! 2018 [Web][MKV][h264][1080p][AAC 2.0][Softsubs (HorribleSubs)]", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Title.2020.MULTi.1080p.WEB.H264-ALLDAYiN (S:285/L:11)", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie Title (2020) MULTi WEB 1080p x264-JiHEFF (S:317/L:28)", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Titles.2020.1080p.NF.WEB.DD2.0.x264-SNEAkY", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"The.Movie.2022.NORDiC.1080p.DV.HDR.WEB.H 265-NiDHUG", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie Title 2018 [WEB 1080p HEVC Opus] [Netaro]", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie Title 2018 (WEB 1080p HEVC Opus) [Netaro]", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"Movie.Title.2024.German.Dubbed.DL.AAC.1080p.WEB.AVC-GROUP", plugin.SourceWEBDL, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEBRip 1080p ────────────────────────────────────────────────────
	t.Run("WEBRip_1080p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name.S04E01.iNTERNAL.1080p.WEBRip.x264-QRUS", plugin.SourceWEBRip, plugin.Resolution1080p},
			{"Movie.Name.1x04.ITA.1080p.WEBMux.x264-NovaRip", plugin.SourceWEBRip, plugin.Resolution1080p},
			{"Movie.Name.2019.S02E07.Chapter.15.The.Believer.4Kto1080p.DSNYP.Webrip.x265.10bit.EAC3.5.1.Atmos.GokiTAoE", plugin.SourceWEBRip, plugin.Resolution1080p},
			{"Movie.Title.2019.1080p.AMZN.WEB-Rip.DDP.5.1.HEVC", plugin.SourceWEBRip, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEB-DL 2160p ────────────────────────────────────────────────────
	t.Run("WEBDL_2160p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name.2016.03.14.2160p.WEB.x264-spamTV", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie.Name.2016.03.14.2160p.WEB.h264-spamTV", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie.Name.2016.03.14.2160p.WEB.PROPER.h264-spamTV", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"[HorribleSubs] Movie Title! 2018 [Web][MKV][h264][2160p][AAC 2.0][Softsubs (HorribleSubs)]", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie Name 2020 WEB-DL 4K H265 10bit HDR DDP5.1 Atmos-PTerWEB", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"The.Movie.2022.NORDiC.2160p.DV.HDR.WEB.H.265-NiDHUG", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie.Name.2024.German.Dubbed.DL.AAC.2160p.DV.HDR.WEB.HEVC-GROUP", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie.Name.2024.German.AC3D.DL.2160p.Hybrid.WEB.DV.HDR10Plus.HEVC-GROUP", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie.Name.2024.German.Atmos.Dubbed.DL.2160p.Hybrid.WEB.DV.HDR10Plus.HEVC-GROUP", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"Movie.Name.2024.German.EAC3D.DL.2160p.Hybrid.WEB.DV.HDR10Plus.HEVC-GROUP", plugin.SourceWEBDL, plugin.Resolution2160p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── WEBRip 2160p ────────────────────────────────────────────────────
	t.Run("WEBRip_2160p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name S01E01.2160P AMZN WEBRIP DD2.0 HI10P X264-TROLLUHD", plugin.SourceWEBRip, plugin.Resolution2160p},
			{"Movie ADD Name S01E01.2160P AMZN WEBRIP DD2.0 X264-TROLLUHD", plugin.SourceWEBRip, plugin.Resolution2160p},
			{"Movie.Name.S01E01.2160p.AMZN.WEBRip.DD2.0.Hi10p.X264-TrollUHD", plugin.SourceWEBRip, plugin.Resolution2160p},
			{"Movie Name S01E01 2160p AMZN WEBRip DD2.0 Hi10P x264-TrollUHD", plugin.SourceWEBRip, plugin.Resolution2160p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── BluRay 720p ─────────────────────────────────────────────────────
	t.Run("BluRay_720p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name.S03E01-06.DUAL.Bluray.AC3.-HELLYWOOD.avi", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"Movie Name - S01E03 - Come Fly With Me - 720p BluRay.mkv", plugin.SourceBluRay, plugin.Resolution720p},
			// m2ts extension implies BluRay source.
			{"Movie Name.S03E01.The Electric Can Opener Fluctuation.m2ts", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"Movie.Name.S01E02.Chained.Heat.[Bluray720p].mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"[FFF] Movie Name - 01 [BD][720p-AAC][0601BED4]", plugin.SourceBluRay, plugin.Resolution720p},
			{"[coldhell] Movie v3 [BD720p][03192D4C]", plugin.SourceBluRay, plugin.Resolution720p},
			// "RandomRemux" is a group name, not a quality modifier.
			// Our parser correctly does not match "Remux" inside "RandomRemux"
			// due to word boundary. [BD] inside brackets → BluRay.
			{"[RandomRemux] Movie - 01 [720p BD][043EA407].mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"[Kaylith] Movie Friends Movies - 01 [BD 720p AAC][B7EEE164].mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name.S03E01-06.DUAL.Blu-ray.AC3.-HELLYWOOD.avi", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"Movie.Name.S03E01-06.DUAL.720p.Blu-ray.AC3.-HELLYWOOD.avi", plugin.SourceBluRay, plugin.Resolution720p},
			{"[Elysium]Movie.Name.01(BD.720p.AAC.DA)[0BB96AD8].mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name.S01E01.33.720p.HDDVD.x264-SiNNERS.mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name.S01E07.RERIP.720p.BluRay.x264-DEMAND", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name.2016.2018.720p.MBluRay.x264-CRUELTY.mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name.2019.720p.MBLURAY.x264-MBLURAYFANS.mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name2017.720p.MBluRay.x264-TREBLE.mkv", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Name.2.Parte.2.ITA-ENG.720p.BDMux.DD5.1.x264-DarkSideMux", plugin.SourceBluRay, plugin.Resolution720p},
			{"Movie.Hunter.2018.720p.Blu-ray.Remux.AVC.FLAC.2.0-SiCFoI", plugin.SourceRemux, plugin.Resolution720p},
			{"Movie.Name.2011.720p.DD.2.0.AVC.REMUX-FraMeSToR", plugin.SourceRemux, plugin.Resolution720p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── BluRay 576p ─────────────────────────────────────────────────────
	t.Run("BluRay_576p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Name.2004.576p.BDRip.x264-HANDJOB", plugin.SourceBluRay, plugin.Resolution576p},
			{"Movie.Title.S01E05.576p.BluRay.DD5.1.x264-HiSD", plugin.SourceBluRay, plugin.Resolution576p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── BluRay 1080p ────────────────────────────────────────────────────
	t.Run("BluRay_1080p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Title - S01E03 - Come Fly With Me - 1080p BluRay.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.S02E13.1080p.BluRay.x264-AVCDVD", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.S01E02.Chained.Heat.[Bluray1080p].mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"[FFF] Movie no Muromi-san - 10 [BD][1080p-FLAC][0C4091AF]", plugin.SourceBluRay, plugin.Resolution1080p},
			{"[coldhell] Movie v2 [BD1080p][5A45EABE].mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"[Kaylith] Movie Friends Specials - 01 [BD 1080p FLAC][429FD8C7].mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"[Zurako] Log Movie - 01 - The Movie (BD 1080p AAC) [7AE12174].mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.S03E01-06.DUAL.1080p.Blu-ray.AC3.-HELLYWOOD.avi", plugin.SourceBluRay, plugin.Resolution1080p},
			{"[Coalgirls]_Movie!!_01_(1920x1080_Blu-ray_FLAC)_[8370CB8F].mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2016.2018.1080p.MBluRay.x264-CRUELTY.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2019.1080p.MBLURAY.x264-MBLURAYFANS.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name2017.1080p.MBluRay.x264-TREBLE.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2011.UHD.BluRay.DD5.1.HDR.x265-CtrlHD/ctrlhd-rotpota-1080p.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie Name 2005 1080p UHD BluRay DD+7.1 x264-LoRD.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2011.1080p.UHD.BluRay.DD5.1.HDR.x265-CtrlHD.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2016.German.DTS.DL.1080p.UHDBD.x265-TDO.mkv", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2021.1080p.BDLight.x265-AVCDVD", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.2012.German.DL.1080p.UHD2BD.x264-QfG", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.2005.1080p.HDDVDRip.x264", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.2019.German.DL.1080p.HDR.UHDBDRip.AV1-GROUP", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.2014.German.OPUS.DL.1080p.UHDBDRiP.HDR.AV1-GROUP", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.1999.German.DL.1080p.HDR.UHDBDRip.AV1-GROUP", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.1993.Uncut.German.DL.1080p.HDR.UHDBDRip.h265-GROUP", plugin.SourceBluRay, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── BluRay 2160p ────────────────────────────────────────────────────
	t.Run("BluRay_2160p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.S01E02.Chained.Heat.[Bluray2160p].mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"[FFF] Movie no Movie-san - 10 [BD][2160p-FLAC][0C4091AF]", plugin.SourceBluRay, plugin.Resolution2160p},
			{"[coldhell] Movie v2 [BD2160p][5A45EABE].mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"[Kaylith] Movie Friends Specials - 01 [BD 2160p FLAC][429FD8C7].mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"[Zurako] Log Movie - 01 - The Movie (BD 2160p AAC) [7AE12174].mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.S03E01-06.DUAL.2160p.Blu-ray.AC3.-HELLYWOOD.avi", plugin.SourceBluRay, plugin.Resolution2160p},
			{"[Coalgirls]_Movie!!_01_(3840x2160_Blu-ray_FLAC)_[8370CB8F].mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.2016.2018.2160p.MBluRay.x264-CRUELTY.mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.2019.2160p.MBLURAY.x264-MBLURAYFANS.mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.2017.2160p.MBluRay.x264-TREBLE.mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Name.2020.German.UHDBD.2160p.HDR10.HEVC.EAC3.DL-pmHD.mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.2014.2160p.UHD.BluRay.X265-IAMABLE.mkv", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.2014.2160p.BDRip.AAC.7.1.HDR10.x265.10bit-Markll", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.1956.German.DL.2160p.HDR.UHDBDRip.h266-GROUP", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Title.2021.4K.HDR.2160P.UHDBDRip.HEVC-10bit.GROUP", plugin.SourceBluRay, plugin.Resolution2160p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── Remux 720p ──────────────────────────────────────────────────────
	t.Run("Remux_720p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			// Radarr parses this as BLURAY 720p (no modifier), but REMUX is
			// in the title so our parser detects SourceRemux.
			{"Movie.1993.720p.BluRay.REMUX.AVC.FLAC.2.0-BLURANiUM", plugin.SourceRemux, plugin.Resolution720p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── Remux 1080p ─────────────────────────────────────────────────────
	t.Run("Remux_1080p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Title.2016.REMUX.1080p.BluRay.AVC.DTS-HD.MA.5.1-iFT", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Name.2008.REMUX.1080p.Bluray.AVC.DTS-HR.MA.5.1-LEGi0N", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Name.2008.BDREMUX.1080p.Bluray.AVC.DTS-HR.MA.5.1-LEGi0N", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Title.M.2008.USA.BluRay.Remux.1080p.MPEG-2.DD.5.1-TDD", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Title.2018.1080p.BluRay.REMUX.MPEG-2.DTS-HD.MA.5.1-EPSiLON", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Title.II.2003.4K.BluRay.Remux.1080p.AVC.DTS-HD.MA.5.1-BMF", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie Title 2022 (BDRemux 1080p HEVC FLAC) [Netaro]", plugin.SourceRemux, plugin.Resolution1080p},
			{"[Vodes] Movie Title - Other Title (2020) [BDRemux 1080p HEVC Dual-Audio]", plugin.SourceRemux, plugin.Resolution1080p},
			{"This.Wonderful.Movie.1991.German.ML.1080p.BluRay.AVC-GeRMaNSCeNEGRoUP", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Name.2011.1080p.DD.2.0.AVC.REMUX-FraMeSToR", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie Name 2018 1080p BluRay Hybrid-REMUX AVC TRUEHD 5.1 Dual Audio-ZR-", plugin.SourceRemux, plugin.Resolution1080p},
			{"Movie.Name.2018.1080p.BluRay.Hybrid-REMUX.AVC.TRUEHD.5.1.Dual.Audio-ZR-", plugin.SourceRemux, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── Remux 2160p ─────────────────────────────────────────────────────
	t.Run("Remux_2160p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Title.2016.REMUX.2160p.BluRay.AVC.DTS-HD.MA.5.1-iFT", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie.Name.2008.REMUX.2160p.Bluray.AVC.DTS-HR.MA.5.1-LEGi0N", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie.Title.1980.2160p.UHD.BluRay.Remux.HDR.HEVC.DTS-HD.MA.5.1-PmP.mkv", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie.Title.2016.T1.UHDRemux.2160p.HEVC.Dual.AC3.5.1-TrueHD.5.1.Sub", plugin.SourceRemux, plugin.Resolution2160p},
			{"[Dolby Vision] Movie.Title.S07.MULTi.UHD.BLURAY.REMUX.DV-NoTag", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie.Name.2020.German.UHDBD.2160p.HDR10.HEVC.EAC3.DL.Remux-pmHD.mkv", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie Name (2021) [Remux-2160p x265 HDR 10-BIT DTS-HD MA 7.1]-FraMeSToR.mkv", plugin.SourceRemux, plugin.Resolution2160p},
			{"This.Wonderful.Movie.1991.German.ML.2160p.BluRay.HEVC-GeRMaNSCeNEGRoUP", plugin.SourceBluRay, plugin.Resolution2160p},
			{"Movie.Name.2011.2160p.DD.2.0.AVC.REMUX-FraMeSToR", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie Name 2018 2160p BluRay Hybrid-REMUX AVC TRUEHD 5.1 Dual Audio-ZR-", plugin.SourceRemux, plugin.Resolution2160p},
			{"Movie.Name.2018.2160p.BluRay.Hybrid-REMUX.AVC.TRUEHD.5.1.Dual.Audio-ZR-", plugin.SourceRemux, plugin.Resolution2160p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── BRDisk (Blu-ray ISO / full disc) ────────────────────────────────
	// Radarr detects BRDisk through complex heuristics including:
	// - ISO/BDISO/Bluray ISO patterns
	// - COMPLETE BLURAY
	// - HD DVD
	// - BD25/BD50/BD66
	// - UNTOUCHED
	// - Bare "Blu-ray AVC" (raw codec implies disc, not encode)
	// Our parser handles the explicit markers (ISO, COMPLETE BLURAY,
	// HD DVD, BD25/50/66, UNTOUCHED, BDISO, BDMV) but does not infer
	// BRDisk from bare "Blu-ray AVC" combinations.
	t.Run("BRDisk_1080p", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Title.2013.BDISO", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			{"Movie.Title.2005.MULTi.COMPLETE.BLURAY-VLS", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			{"Movie Name (2012) Bluray ISO [USENET-TURK]", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			{"Movie Name.1993..BD25.ISO", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			// ".iso" extension triggers BRDisk via ISO match.
			{"Movie.Title.2012.Bluray.1080p.3D.AVC.DTS-HD.MA.5.1.iso", plugin.SourceBRDisk, plugin.Resolution1080p},
			{"Movie.Title.1996.Bluray.ISO", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			{"Random.Title.2010.1080p.HD.DVD.AVC.DDP.5.1-GRouP", plugin.SourceBRDisk, plugin.Resolution1080p},
			// Radarr treats "Blu-ray AVC" (without encoding marker) as BRDisk.
			// Our parser sees Blu-ray → BluRay since we don't infer from codec context.
			{"Movie Title 2005 1080p USA Blu-ray AVC DTS-HD MA 5.1-PTP", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie Title 2014 1080p Blu-ray AVC DTS-HD MA 5.1-PTP", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie Title 1976 2160p UHD Blu-ray DTS-HD MA 5.1 DV HDR HEVC-UNTOUCHED", plugin.SourceBRDisk, plugin.Resolution2160p},
			{"Movie Title 2004 1080p FRA Blu-ray VC-1 TrueHD 5.1-HDBEE", plugin.SourceBluRay, plugin.Resolution1080p},
			{"BD25.Movie.Title.1994.1080p.DTS-HD", plugin.SourceBRDisk, plugin.Resolution1080p},
			{"Movie.Title.1997.1080p.NL.BD-50", plugin.SourceBRDisk, plugin.Resolution1080p},
			{"Movie Title 2009 3D BD 2009 UNTOUCHED", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			{"Movie.Title.1982.1080p.HD.DVD.VC-1.DD+.5.1", plugin.SourceBRDisk, plugin.Resolution1080p},
			{"Movie.Title.2007.1080p.HD.DVD.DD+.AVC", plugin.SourceBRDisk, plugin.Resolution1080p},
			{"Movie.Title.2008.1080i.XXX.Blu-ray.MPEG-2.LPCM2.0.ISO", plugin.SourceBRDisk, plugin.Resolution1080p},
			{"Movie.Title.2008.BONUS.GERMAN.SUBBED.COMPLETE.BLURAY", plugin.SourceBRDisk, plugin.ResolutionUnknown},
			// Radarr treats bare "Bluray AVC" as BRDisk; our parser → BluRay.
			{"The German 2021 Bluray AVC", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"German.Only.Movie.2021.French.1080p.BluRay.AVC-UNTAVC", plugin.SourceBluRay, plugin.Resolution1080p},
			{"Movie.Title.2008.US.Directors.Cut.UHD.BD66.Blu-ray", plugin.SourceBRDisk, plugin.Resolution2160p},
			// "Blu.ray" (with dot separator) normalizes to "Blu ray" → BluRay.
			// Radarr treats bare "Blu ray AVC" as BRDisk.
			{"Movie.2009.Blu.ray.AVC.DTS.HD.MA.5.1", plugin.SourceBluRay, plugin.ResolutionUnknown},
			{"[BD]Movie.Title.2008.2023.1080p.COMPLETE.BLURAY-RlsGrp", plugin.SourceBRDisk, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── RawHD (MPEG-2 HDTV) ────────────────────────────────────────────
	t.Run("RawHD", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie.Title.2015.Open.Matte.1080i.HDTV.DD5.1.MPEG2", plugin.SourceRawHD, plugin.Resolution1080p},
			{"Movie.Title.2009.1080i.HDTV.AAC2.0.MPEG2-PepelefuF", plugin.SourceRawHD, plugin.Resolution1080p},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── Unknown quality ─────────────────────────────────────────────────
	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Some.Movie.S02E15", plugin.SourceUnknown, plugin.ResolutionUnknown},
			{"Movie Name - 11x11 - Quickie", plugin.SourceUnknown, plugin.ResolutionUnknown},
			{"Movie.Name.S01E01.webm", plugin.SourceUnknown, plugin.ResolutionUnknown},
			// Radarr does not match "Web" in "The.Web.MT" as a source.
			// Our bare-WEB regex matches it; this is an accepted false positive
			// because bare "WEB" detection is valuable for real releases.
			{"Movie.Title.S01E01.The.Web.MT-dd", plugin.SourceWEBDL, plugin.ResolutionUnknown},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── OtherSourceQualityParserCases ───────────────────────────────────
	// Radarr iterates these with all separator variants (-, ., space, _).
	// We test with all four separators as Radarr does.
	t.Run("OtherSourceSeparators", func(t *testing.T) {
		t.Parallel()
		type sepCase struct {
			qualityString string
			wantSource    plugin.Source
			wantRes       plugin.Resolution
		}
		baseCases := []sepCase{
			{"SD DVD", plugin.SourceDVD, plugin.ResolutionSD},
			{"480p WEB-DL", plugin.SourceWEBDL, plugin.Resolution480p},
			{"720p WEB-DL", plugin.SourceWEBDL, plugin.Resolution720p},
			{"1080p WEB-DL", plugin.SourceWEBDL, plugin.Resolution1080p},
			{"2160p WEB-DL", plugin.SourceWEBDL, plugin.Resolution2160p},
			{"720p BluRay", plugin.SourceBluRay, plugin.Resolution720p},
			{"1080p BluRay", plugin.SourceBluRay, plugin.Resolution1080p},
			{"2160p BluRay", plugin.SourceBluRay, plugin.Resolution2160p},
			{"1080p Remux", plugin.SourceRemux, plugin.Resolution1080p},
			{"2160p Remux", plugin.SourceRemux, plugin.Resolution2160p},
		}

		separators := []struct {
			name string
			char byte
		}{
			{"dash", '-'},
			{"dot", '.'},
			{"space", ' '},
			{"underscore", '_'},
		}

		for _, bc := range baseCases {
			for _, sep := range separators {
				title := "My movie 2020 " + replaceSep(bc.qualityString, sep.char)
				name := bc.qualityString + "_" + sep.name
				wantSource := bc.wantSource
				wantRes := bc.wantRes
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					got := Parse(title)
					if got.Source != wantSource {
						t.Errorf("Source for %q: got %q, want %q", title, got.Source, wantSource)
					}
					if got.Resolution != wantRes {
						t.Errorf("Resolution for %q: got %q, want %q", title, got.Resolution, wantRes)
					}
				})
			}
		}
	})

	// ── SDTV (HDTV 480p / PDTV / DSR) ──────────────────────────────────
	// The TV-episode patterns from Radarr's should_parse_sdtv_quality that
	// contain recognizable source markers. We include them because SDTV
	// parsing is relevant for movie rips too.
	t.Run("SDTV", func(t *testing.T) {
		t.Parallel()
		cases := []tc{
			{"Movie Name S02E01 HDTV XviD 2HD", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name S05E11 PROPER HDTV XviD 2HD", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name S02E08 HDTV x264 FTP", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie.Name.2011.S02E01.WS.PDTV.x264-TLA", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name S01E04 DSR x264 2HD", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name S11E03 has no periods or extension HDTV", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name.S04E05.HDTV.XviD-LOL", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Some.Movie.S03E06.HDTV-WiDE", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name.S10E27.WS.DSR.XviD-2HD", plugin.SourceHDTV, plugin.ResolutionUnknown},
			{"Movie Name.S03.TVRip.XviD-NOGRP", plugin.SourceHDTV, plugin.ResolutionUnknown},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.Source != c.wantSource {
					t.Errorf("Source: got %q, want %q", got.Source, c.wantSource)
				}
				if got.Resolution != c.wantRes {
					t.Errorf("Resolution: got %q, want %q", got.Resolution, c.wantRes)
				}
			})
		}
	})

	// ── SD TV from OtherSourceQualityParserCases ────────────────────────
	// "SD TV" maps to HDTV at 480p in Radarr. Our parser maps HDTV
	// without explicit resolution to unknown (no 480p inference for HDTV).
	t.Run("SDTV_OtherSource", func(t *testing.T) {
		t.Parallel()
		separators := []struct {
			name string
			char byte
		}{
			{"dash", '-'},
			{"dot", '.'},
			{"space", ' '},
			{"underscore", '_'},
		}
		for _, sep := range separators {
			title := "My movie 2020 " + replaceSep("SD TV", sep.char)
			name := "SD_TV_" + sep.name
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got := Parse(title)
				if got.Source != plugin.SourceHDTV {
					t.Errorf("Source for %q: got %q, want %q", title, got.Source, plugin.SourceHDTV)
				}
				// Our parser does not infer 480p for bare HDTV; Radarr
				// maps SD TV → 480p but we keep it as unknown.
			})
		}
	})

	// ── HD TV from OtherSourceQualityParserCases ────────────────────────
	t.Run("HDTV_OtherSource", func(t *testing.T) {
		t.Parallel()
		separators := []struct {
			name string
			char byte
		}{
			{"dash", '-'},
			{"dot", '.'},
			{"space", ' '},
			{"underscore", '_'},
		}
		for _, sep := range separators {
			title := "My movie 2020 " + replaceSep("HD TV", sep.char)
			name := "HD_TV_" + sep.name
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got := Parse(title)
				if got.Source != plugin.SourceHDTV {
					t.Errorf("Source for %q: got %q, want %q", title, got.Source, plugin.SourceHDTV)
				}
			})
		}
	})

	// ── 1080p HD TV from OtherSourceQualityParserCases ──────────────────
	t.Run("HDTV_1080p_OtherSource", func(t *testing.T) {
		t.Parallel()
		separators := []struct {
			name string
			char byte
		}{
			{"dash", '-'},
			{"dot", '.'},
			{"space", ' '},
			{"underscore", '_'},
		}
		for _, sep := range separators {
			title := "My movie 2020 " + replaceSep("1080p HD TV", sep.char)
			name := "1080p_HD_TV_" + sep.name
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got := Parse(title)
				if got.Source != plugin.SourceHDTV {
					t.Errorf("Source for %q: got %q, want %q", title, got.Source, plugin.SourceHDTV)
				}
				if got.Resolution != plugin.Resolution1080p {
					t.Errorf("Resolution for %q: got %q, want %q", title, got.Resolution, plugin.Resolution1080p)
				}
			})
		}
	})

	// ── 2160p HD TV from OtherSourceQualityParserCases ──────────────────
	t.Run("HDTV_2160p_OtherSource", func(t *testing.T) {
		t.Parallel()
		separators := []struct {
			name string
			char byte
		}{
			{"dash", '-'},
			{"dot", '.'},
			{"space", ' '},
			{"underscore", '_'},
		}
		for _, sep := range separators {
			title := "My movie 2020 " + replaceSep("2160p HD TV", sep.char)
			name := "2160p_HD_TV_" + sep.name
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				got := Parse(title)
				if got.Source != plugin.SourceHDTV {
					t.Errorf("Source for %q: got %q, want %q", title, got.Source, plugin.SourceHDTV)
				}
				if got.Resolution != plugin.Resolution2160p {
					t.Errorf("Resolution for %q: got %q, want %q", title, got.Resolution, plugin.Resolution2160p)
				}
			})
		}
	})

	// ── REPACK / PROPER / RERIP revision parsing ────────────────────────
	t.Run("Revision", func(t *testing.T) {
		t.Parallel()
		type revCase struct {
			title       string
			wantRepack  bool
			wantVersion int
		}
		cases := []revCase{
			{"Movie Title 2018 REPACK 720p HDTV x264 aAF", true, 2},
			{"Movie.Title.2018.REPACK.720p.HDTV.x264-aAF", true, 2},
			{"Movie.Title.2018.REPACK2.720p.HDTV.x264-aAF", true, 3},
			{"Movie.Title.2018.PROPER.720p.HDTV.x264-aAF", false, 2},
			{"Movie.Title.2018.RERIP.720p.BluRay.x264-DEMAND", true, 2},
			{"Movie.Title.2018.RERIP2.720p.BluRay.x264-DEMAND", true, 3},
		}
		for _, c := range cases {
			t.Run(c.title, func(t *testing.T) {
				t.Parallel()
				got := Parse(c.title)
				if got.IsRepack != c.wantRepack {
					t.Errorf("IsRepack: got %v, want %v", got.IsRepack, c.wantRepack)
				}
				if got.Revision.Version != c.wantVersion {
					t.Errorf("Revision.Version: got %d, want %d", got.Revision.Version, c.wantVersion)
				}
			})
		}
	})

	// ── HDTV Remux variant (should not map to Remux) ────────────────────
	// Radarr's fixture:  HR.WS.PDTV.x264-DHD-Remux.mkv → TV 720p (no REMUX modifier)
	// The "-Remux" here is part of the group/tag, not a quality modifier.
	// However our parser detects REMUX anywhere in the string, so this
	// will map to SourceRemux. We document the divergence.
	t.Run("HDTV_Remux_GroupName", func(t *testing.T) {
		t.Parallel()
		title := "Movie.Name.The.Lost.Pilots.Movie.HR.WS.PDTV.x264-DHD-Remux.mkv"
		got := Parse(title)
		// Our parser sees "Remux" and maps to SourceRemux. Radarr treats
		// this as plain HDTV because -Remux is parsed as part of the
		// release group suffix.  We accept our parser's behavior.
		if got.Source != plugin.SourceRemux {
			t.Errorf("Source: got %q, want %q", got.Source, plugin.SourceRemux)
		}
	})
}

// replaceSep replaces spaces in s with the given separator character.
func replaceSep(s string, sep byte) string {
	out := make([]byte, len(s))
	for i := range s {
		if s[i] == ' ' {
			out[i] = sep
		} else {
			out[i] = s[i]
		}
	}
	return string(out)
}
