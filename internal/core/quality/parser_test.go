package quality_test

import (
	"testing"

	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/pkg/plugin"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantRes    plugin.Resolution
		wantSource plugin.Source
		wantCodec  plugin.Codec
		wantHDR    plugin.HDRFormat
		wantName   string
	}{
		// ── 1. Standard BluRay 1080p ─────────────────────────────────────────
		{
			name:       "bluray 1080p x264",
			input:      "The.Dark.Knight.2008.1080p.BluRay.x264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray-1080p x264",
		},
		{
			name:       "bluray 1080p x265 dashed",
			input:      "Interstellar.2014.1080p.Blu-Ray.x265-YIFY",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray-1080p x265",
		},
		{
			name:       "bluray 1080p HEVC",
			input:      "Inception.2010.1080p.BLURAY.HEVC-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray-1080p x265",
		},
		{
			name:       "bluray 1080p no codec",
			input:      "Pulp.Fiction.1994.1080p.BluRay-SPARKS",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray-1080p",
		},

		// ── 2. Standard BluRay 2160p ─────────────────────────────────────────
		{
			name:       "bluray 2160p x265",
			input:      "Dune.2021.2160p.BluRay.x265.10bit-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray-2160p x265",
		},
		{
			name:       "UHD keyword maps to 2160p",
			input:      "Avatar.2009.UHD.BluRay.x265-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "4K keyword maps to 2160p",
			input:      "Top.Gun.Maverick.2022.4K.BluRay.x265-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},

		// ── 3. BluRay Remux ──────────────────────────────────────────────────
		{
			name:       "bluray remux 1080p",
			input:      "The.Godfather.1972.1080p.BluRay.REMUX.AVC-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceRemux,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray Remux-1080p x264",
		},
		{
			name:       "bluray remux 2160p x265",
			input:      "Mad.Max.Fury.Road.2015.2160p.BluRay.REMUX.HEVC-FGT",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceRemux,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
			wantName:   "Bluray Remux-2160p x265",
		},
		{
			name:       "BDREMUX keyword",
			input:      "Parasite.2019.1080p.BDREMUX.x265-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceRemux,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},

		// ── 4. WEB-DL ────────────────────────────────────────────────────────
		{
			name:       "WEB-DL 1080p x264",
			input:      "The.Crown.S01E01.2016.1080p.WEB-DL.DD5.1.H.264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "WEBDL-1080p x264",
		},
		{
			name:       "WEBDL no dash 720p",
			input:      "Breaking.Bad.S05E16.2013.720p.WEBDL.x264-GROUP",
			wantRes:    plugin.Resolution720p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "WEBDL-720p x264",
		},
		{
			name:       "WEB.DL dot-separated",
			input:      "Oppenheimer.2023.1080p.WEB.DL.x265-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "Netflix WEB-DL 2160p",
			input:      "Stranger.Things.S04E01.2022.2160p.NF.WEB-DL.x265-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "Disney+ WEB-DL 1080p",
			input:      "Andor.S01E03.2022.1080p.DSNP.WEB-DL.DDP5.1.H.264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "Amazon WEB-DL 1080p x265",
			input:      "The.Boys.S03E01.2022.1080p.AMZN.WEB-DL.H.265-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},

		// ── 5. WEBRip ────────────────────────────────────────────────────────
		{
			name:       "WEBRip 1080p x264",
			input:      "The.Mandalorian.S01E01.2019.1080p.WEBRip.x264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBRip,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "WEBRip-1080p x264",
		},
		{
			name:       "WEB-Rip dashed",
			input:      "Loki.S02E01.2023.720p.WEB-Rip.x265-GROUP",
			wantRes:    plugin.Resolution720p,
			wantSource: plugin.SourceWEBRip,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},

		// ── 6. HDTV ──────────────────────────────────────────────────────────
		{
			name:       "HDTV 720p x264",
			input:      "Game.of.Thrones.S08E06.2019.720p.HDTV.x264-GROUP",
			wantRes:    plugin.Resolution720p,
			wantSource: plugin.SourceHDTV,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "HDTV-720p x264",
		},
		{
			name:       "HDTV 1080p no codec",
			input:      "The.Wire.S01E01.2002.1080p.HDTV-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceHDTV,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},

		// ── 7. HDR variants ──────────────────────────────────────────────────
		{
			name:       "BluRay 2160p HDR10",
			input:      "Blade.Runner.2049.2017.2160p.BluRay.x265.HDR10-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRHDR10,
			wantName:   "Bluray-2160p x265 HDR10",
		},
		{
			name:       "BluRay 2160p plain HDR token",
			input:      "Everything.Everywhere.All.at.Once.2022.2160p.BluRay.HEVC.HDR-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRHDR10,
		},
		{
			name:       "Dolby Vision DV token",
			input:      "The.Batman.2022.2160p.BluRay.x265.DV-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRDolbyVision,
			wantName:   "Bluray-2160p x265 Dolby Vision",
		},
		{
			name:       "Dolby Vision DoVi token",
			input:      "Severance.S01E01.2022.2160p.ATVP.WEB-DL.DoVi.x265-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRDolbyVision,
		},
		{
			name:       "Dolby Vision full name",
			input:      "House.of.the.Dragon.S01E01.2022.2160p.WEB-DL.Dolby.Vision.x265-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRDolbyVision,
		},
		{
			name:       "HDR10Plus token",
			input:      "Dune.Part.Two.2024.2160p.BluRay.x265.HDR10Plus-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRHDR10Plus,
			wantName:   "Bluray-2160p x265 HDR10+",
		},
		{
			name:       "HLG token",
			input:      "One.Piece.Film.Red.2022.2160p.BluRay.x265.HLG-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRHLG,
			wantName:   "Bluray-2160p x265 HLG",
		},
		{
			name:       "Remux 2160p Dolby Vision",
			input:      "The.Northman.2022.2160p.BluRay.REMUX.HEVC.DoVi-FGT",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceRemux,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRDolbyVision,
			wantName:   "Bluray Remux-2160p x265 Dolby Vision",
		},

		// ── 8. Codec variants ────────────────────────────────────────────────
		{
			name:       "H.264 codec token",
			input:      "The.Shawshank.Redemption.1994.1080p.BluRay.H.264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "H.265 codec token",
			input:      "Tenet.2020.1080p.WEB-DL.H.265-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "AV1 codec",
			input:      "Killers.of.the.Flower.Moon.2023.1080p.WEB-DL.AV1-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecAV1,
			wantHDR:    plugin.HDRNone,
			wantName:   "WEBDL-1080p AV1",
		},
		{
			name:       "AVC maps to x264",
			input:      "Gladiator.2000.1080p.BluRay.AVC-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},

		// ── 9. SD / DVD ──────────────────────────────────────────────────────
		{
			name:       "DVDRip XviD",
			input:      "The.Matrix.1999.DVDRip.XviD-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVD,
			wantCodec:  plugin.CodecXVID,
			wantHDR:    plugin.HDRNone,
			wantName:   "DVD-SD XviD",
		},
		{
			name:       "DVD.Rip dot-separated with 480p",
			input:      "Schindlers.List.1993.480p.DVD.Rip.x264-GROUP",
			wantRes:    plugin.Resolution480p,
			wantSource: plugin.SourceDVD,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "DVD-480p x264",
		},
		{
			name:       "480p explicit resolution",
			input:      "Some.Movie.2005.480p.WEBRip.x264-GROUP",
			wantRes:    plugin.Resolution480p,
			wantSource: plugin.SourceWEBRip,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "WEBRip-480p x264",
		},
		{
			name:       "576p explicit resolution",
			input:      "Another.Movie.2003.576p.DVDRip.XviD-GROUP",
			wantRes:    plugin.Resolution576p,
			wantSource: plugin.SourceDVD,
			wantCodec:  plugin.CodecXVID,
			wantHDR:    plugin.HDRNone,
			wantName:   "DVD-576p XviD",
		},
		{
			name:       "DVDR keyword",
			input:      "Fargo.1996.DVDR-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDR,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
			wantName:   "DVD-R-SD",
		},
		{
			name:       "DivX codec maps to XviD",
			input:      "Se7en.1995.DVDRip.DivX-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVD,
			wantCodec:  plugin.CodecXVID,
			wantHDR:    plugin.HDRNone,
		},

		// ── 10. CAM / TS ─────────────────────────────────────────────────────
		{
			name:       "CAM release",
			input:      "Avengers.Endgame.2019.CAM.x264-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceCAM,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "CAM x264",
		},
		{
			name:       "HDCAM release",
			input:      "Spider-Man.No.Way.Home.2021.HDCAM-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceCAM,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "TS release (telesync)",
			input:      "Fast.X.2023.TS.x264-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTelesync,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "Telesync x264",
		},
		{
			name:       "TELESYNC keyword",
			input:      "Ant-Man.2015.TELESYNC-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTelesync,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
			wantName:   "Telesync",
		},
		{
			name:       "TELECINE keyword",
			input:      "Aquaman.2018.TELECINE-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTELECINE,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},

		// ── 10a. New pre-release sources ────────────────────────────────────
		{
			name:       "WORKPRINT keyword",
			input:      "Some.Movie.2024.WORKPRINT-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceWorkprint,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
			wantName:   "Workprint",
		},
		{
			name:       "WP keyword",
			input:      "Movie.2024.WP.x264-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceWorkprint,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "HDTS keyword (telesync)",
			input:      "Avengers.2019.HDTS.x264-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTelesync,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "PDVD keyword (telesync)",
			input:      "Movie.2023.PDVD.x264-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTelesync,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "TC keyword (telecine)",
			input:      "Some.Film.2024.TC-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTELECINE,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "HDTC keyword (telecine)",
			input:      "Film.2024.HDTC.x264-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceTELECINE,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "DVDSCR keyword",
			input:      "Awards.Movie.2024.DVDSCR.x264-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDSCR,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "DVDSCR-SD x264",
		},
		{
			name:       "SCREENER keyword",
			input:      "Oscar.Film.2024.SCREENER-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDSCR,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "SCR keyword",
			input:      "Movie.2024.SCR.x264-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDSCR,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "R5 keyword (regional)",
			input:      "Action.Movie.2024.R5.x264-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceRegional,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
			wantName:   "Regional-SD x264",
		},
		{
			name:       "REGIONAL keyword",
			input:      "Movie.2024.REGIONAL-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceRegional,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},

		// ── 10b. Disc image sources ─────────────────────────────────────────
		{
			name:       "DVD-R keyword",
			input:      "Classic.Movie.2000.DVD-R-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDR,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "DVD9 keyword",
			input:      "Movie.1999.DVD9-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDR,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "DVD5 keyword",
			input:      "Old.Film.2001.DVD5-GROUP",
			wantRes:    plugin.ResolutionSD,
			wantSource: plugin.SourceDVDR,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "BDMV keyword (BR-DISK)",
			input:      "Movie.2023.BDMV-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceBRDisk,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
			wantName:   "BR-DISK",
		},
		{
			name:       "BD25 keyword (BR-DISK)",
			input:      "Film.2022.BD25-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceBRDisk,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "BD50 keyword (BR-DISK)",
			input:      "Movie.2021.BD50-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceBRDisk,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "RAW-HD keyword",
			input:      "Concert.2022.RAW-HD-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceRawHD,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
			wantName:   "Raw-HD",
		},

		// ── 11. PROPER / REPACK flags (don't affect quality) ─────────────────
		{
			name:       "PROPER flag ignored",
			input:      "The.Dark.Knight.2008.1080p.BluRay.x264.PROPER-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "REPACK flag ignored",
			input:      "Interstellar.2014.1080p.WEB-DL.x265.REPACK-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "RERIP flag ignored",
			input:      "Gladiator.2000.1080p.BluRay.x264.RERIP-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},

		// ── 12. Edition info (does not affect quality) ────────────────────────
		{
			name:       "Extended edition",
			input:      "The.Lord.of.the.Rings.2001.Extended.1080p.BluRay.x264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "Directors Cut edition",
			input:      "Blade.Runner.1982.Directors.Cut.1080p.BluRay.x265-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "Theatrical Cut edition",
			input:      "Zack.Snyders.Justice.League.2021.Theatrical.2160p.WEB-DL.x265.HDR-GROUP",
			wantRes:    plugin.Resolution2160p,
			wantSource: plugin.SourceWEBDL,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRHDR10,
		},

		// ── 13. Multi-word titles with embedded years ─────────────────────────
		{
			name:       "title with year in middle",
			input:      "Once.Upon.a.Time.in.Hollywood.2019.1080p.BluRay.x264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "title with numbers",
			input:      "2001.A.Space.Odyssey.1968.1080p.BluRay.x264-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX264,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "spaces as separators",
			input:      "The Revenant 2015 1080p BluRay x265-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceBluRay,
			wantCodec:  plugin.CodecX265,
			wantHDR:    plugin.HDRNone,
		},

		// ── 14. Ambiguous / minimal info ─────────────────────────────────────
		{
			name:       "resolution only",
			input:      "SomeMovie.1080p-GROUP",
			wantRes:    plugin.Resolution1080p,
			wantSource: plugin.SourceUnknown,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "no quality info",
			input:      "SomeOldMovie-GROUP",
			wantRes:    plugin.ResolutionUnknown,
			wantSource: plugin.SourceUnknown,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
		{
			name:       "720p WEBRip no codec",
			input:      "Barry.S01E01.2018.720p.WEBRip-GROUP",
			wantRes:    plugin.Resolution720p,
			wantSource: plugin.SourceWEBRip,
			wantCodec:  plugin.CodecUnknown,
			wantHDR:    plugin.HDRNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := quality.Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", tc.input, err)
			}

			if got.Resolution != tc.wantRes {
				t.Errorf("Resolution: got %q, want %q", got.Resolution, tc.wantRes)
			}
			if got.Source != tc.wantSource {
				t.Errorf("Source: got %q, want %q", got.Source, tc.wantSource)
			}
			if got.Codec != tc.wantCodec {
				t.Errorf("Codec: got %q, want %q", got.Codec, tc.wantCodec)
			}
			if got.HDR != tc.wantHDR {
				t.Errorf("HDR: got %q, want %q", got.HDR, tc.wantHDR)
			}
			if tc.wantName != "" && got.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tc.wantName)
			}
		})
	}
}
