# Plan: Port Radarr Test Cases to Luminarr Parser

**Status**: Draft
**Scope**: Port ~550 test cases from Radarr's parser test suite to `internal/parser/`
**Source**: `/data/home/davidfic/dev/luminarr/Radarr/src/NzbDrone.Core.Test/ParserTests/`

---

## Summary

Radarr has ~550 parser test cases battle-tested against real-world release names. Many cover edge cases our current 475 tests don't touch. Porting these will significantly harden the parser, especially for:
- Unusual source/resolution combinations
- Anime naming conventions
- Release group extraction with bad suffix stripping
- German/international release naming
- BRDISK/ISO detection
- Edge cases in WEB-DL vs WEBRip vs HDTV detection

---

## What to Port

### 1. Quality Parser (328 cases → ~200 new)

**High value — patterns we don't test:**
- CAM variants: `HQCAM`, `NewCAM`, `NEWCAM`, `HDCAMRip`
- WEB-DL inferred from `iTunesHD`, `WebHD`, `Viva MKV WEB`
- Anime bracket format: `[HorribleSubs] Title [Web][MKV][h264][1080p]`
- `960p` mapping to 720p
- `BDMux`, `MBluRay` as BluRay variants
- `HDDVD`, `HDDVDRip`, `HD.DVD` as BluRay sources
- `UHDBD`, `UHDBDRip`, `UHD2BD` as BluRay variants
- `BDLight` as BluRay variant
- `TSRip`, `TeleSynch` as telesync variants
- `PDTV`, `DSR`, `TVRip` as SDTV/HDTV variants
- BRDISK detection: `BDISO`, `COMPLETE.BLURAY`, `Bluray ISO`, `BD25.ISO`, `HD.DVD`
- Hybrid-REMUX: `BluRay.Hybrid-REMUX`
- `4Kto1080p` indicating downscale from 4K
- `DP.WEB` (Disney+ indicator)
- German release patterns: `German.DTS.DL`, `German.Dubbed.DL`, `AC3D`, `EAC3D`
- `h266` codec detection (next-gen)
- RAWHD: `MPEG2` in 1080i HDTV context
- `WS` (widescreen) flag in various positions

**Skip (not relevant to Luminarr's movie parser):**
- TV show season/episode patterns (S01E01)
- SDTV 480p patterns (Luminarr only handles movies, not daily TV)

### 2. Release Group Extraction (157 cases → ~100 new)

**High value — patterns we don't test:**
- Bad suffix stripping: `-postbot`, `-xpost`, `-Rakuv`, `-Obfuscated`, `-NZBgeek`, `-BUYMORE`, `-WhiteRev`, `-AsRequested`, `-Chamele0n`, `-Z0iDS3N`, `-GEROV`, `-4P`, `-4Planet`, `-AlteZachen`, `-RePACKPOST`
- Tracker prefixes: `[ www.Torrenting.com ] -` before the release name
- Language suffixes: `-SKGTV English`, `-SKGTV_English`, `-SKGTV.English`
- Anime groups in brackets: `[FFF]`, `[HorribleSubs]`, `[Anime-Koi]`
- Complex bracket nesting: `(1080p BluRay x265 HEVC 10bit AAC 7.1 Tigole)`, `[QxR]`
- Groups with special characters: `D-Z0N3`, `Blu-bits`, `DX-TV`, `FTW-HS`, `VH-PROD`, `-ZR-`, `Koten_Gars`, `E.N.D`
- Groups in parens with nested brackets: `(Koten_Gars)`, `[HDO]`, `(TAoE)`
- Unicode group names: `ZØNEHD`
- Path-based extraction: `C:\Test\...archivist.mkv`
- No group when `DTS-MA.5.1`, `DTS-X.MA.5.1`, `DTS-ES.MA.5.1` end the title
- Groups with spaces: `Eml HDTeam`, `BEN THE MEN`
- `[YTS.MX]`, `[YTS.AG]`, `[YTS.LT]` bracket groups
- Group preceded by ` - ` (space-dash-space): `YIFY`, `HONE`, etc.

### 3. Edition Parser (59 cases → ~30 new)

**High value — patterns we don't test:**
- `Despecialized` (Star Wars fan edit)
- `2in1` edition
- `Restored` edition
- `Assembly Cut`
- `Ultimate Hunter Edition` (named variants)
- `Diamond Edition`
- `Signature Edition`
- `Imperial Edition`
- Edition in parentheses: `(Special.Edition.Remastered)`
- Edition before year: `Movie Extended 2012`
- Combined editions: `Extended Directors Cut Fan Edit`
- `Extended Theatrical Version IMAX` (multiple)
- Negative cases: `Holiday Special`, `Directors.Cut.German` (group name, not edition), `Rogue Movie` (not Rogue Cut), `Uncut.Movie` (movie title, not edition), `The.Christmas.Edition` (movie title)

### 4. Audio Codec (24 cases → ~10 new)

**High value — patterns we don't test:**
- `DTS-ES` variant
- `DTS-HD HRA` (High Resolution Audio, not Master Audio)
- `WMA` (wmav1, wmav2) audio
- `Vorbis` audio
- `MP2` audio
- `ADPCM` variants mapping to PCM
- Atmos detection from MediaInfo `thd+` profile

---

## Implementation Strategy

### Step 1: Create `radarr_compat_test.go`

A single large test file in `internal/parser/` with all ported test cases, organized by the Radarr fixture they came from. This keeps the ported tests clearly identified and separate from our original tests.

### Step 2: Adapt Test Cases

For each Radarr test case:
1. Translate the release title string directly (no changes needed)
2. Map Radarr's quality enum to our `plugin.Source` / `plugin.Resolution` / `plugin.Codec`
3. Map Radarr's `Modifier` (REMUX, BRDISK, RAWHD) to our `Source` values
4. Skip TV-only patterns (S01E01) that don't apply to movie parsing
5. Note: Radarr's "SDTV" source maps to our `SourceHDTV` or `SourceUnknown` depending on context

### Step 3: Handle Expected Failures

Some Radarr test cases may expose gaps in our parser. For those:
1. First add the test case with the Radarr-expected value
2. If our parser produces a different (but acceptable) result, document why
3. If our parser is wrong, fix the parser

### Step 4: Release Group Bad Suffix Stripping

Radarr strips 25+ known bad suffixes from release group names. We don't do this today. Options:
1. Add a `badSuffixes` list to `parseReleaseGroup()` that strips known tracker/post-processing suffixes
2. Or accept this as a known gap

### Step 5: Verify

Run the full test suite and ensure no regressions.

---

## Mapping: Radarr Quality → Luminarr Types

| Radarr Source | Radarr Modifier | Luminarr Source |
|---------------|-----------------|-----------------|
| BLURAY | (none) | `SourceBluRay` |
| BLURAY | REMUX | `SourceRemux` |
| BLURAY | BRDISK | `SourceBRDisk` |
| TV | RAWHD | `SourceRawHD` |
| TV | (none) | `SourceHDTV` |
| WEBDL | (none) | `SourceWEBDL` |
| WEBRIP | (none) | `SourceWEBRip` |
| DVD | (none) | `SourceDVD` |
| DVD | REMUX | `SourceDVDR` |
| CAM | (none) | `SourceCAM` |
| TELESYNC | (none) | `SourceTelesync` |
| UNKNOWN | (none) | `SourceUnknown` |

| Radarr Resolution | Luminarr Resolution |
|-------------------|---------------------|
| R480p | `Resolution480p` or `ResolutionSD` |
| R576p | `Resolution576p` |
| R720p | `Resolution720p` |
| R1080p | `Resolution1080p` |
| R2160p | `Resolution2160p` |
| Unknown | `ResolutionUnknown` |

---

## Estimated New Test Cases

| Area | Radarr Total | New to Port | Notes |
|------|-------------|-------------|-------|
| Quality/Source | 328 | ~200 | Skip TV-specific patterns |
| Release Group | 157 | ~100 | Include bad suffix stripping |
| Edition | 59 | ~30 | Include negative cases |
| Audio Codec | 24 | ~10 | MediaInfo-based tests need adaptation |
| **Total** | **568** | **~340** | |

After porting, total parser test count: ~475 (current) + ~340 (Radarr) = **~815 test cases**
