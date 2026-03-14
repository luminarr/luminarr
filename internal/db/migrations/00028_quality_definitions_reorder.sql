-- +goose Up
-- Reorder quality definitions: most popular/highest quality first, pre-release at the bottom.

-- 2160p tiers (highest quality first).
UPDATE quality_definitions SET sort_order = 10  WHERE id = '2160p-remux-x265-hdr10';
UPDATE quality_definitions SET sort_order = 20  WHERE id = '2160p-bluray-x265-hdr10';
UPDATE quality_definitions SET sort_order = 30  WHERE id = '2160p-webdl-x265-hdr10';
UPDATE quality_definitions SET sort_order = 40  WHERE id = '2160p-webrip-x265-hdr10';
UPDATE quality_definitions SET sort_order = 50  WHERE id = '2160p-hdtv-x265-hdr10';

-- 1080p tiers.
UPDATE quality_definitions SET sort_order = 60  WHERE id = '1080p-remux-x265-none';
UPDATE quality_definitions SET sort_order = 70  WHERE id = '1080p-bluray-x265-none';
UPDATE quality_definitions SET sort_order = 80  WHERE id = '1080p-webdl-x264-none';
UPDATE quality_definitions SET sort_order = 90  WHERE id = '1080p-webrip-x265-none';
UPDATE quality_definitions SET sort_order = 100 WHERE id = '1080p-hdtv-x264-none';

-- 720p tiers.
UPDATE quality_definitions SET sort_order = 110 WHERE id = '720p-bluray-x264-none';
UPDATE quality_definitions SET sort_order = 120 WHERE id = '720p-webdl-x264-none';
UPDATE quality_definitions SET sort_order = 130 WHERE id = '720p-webrip-x264-none';
UPDATE quality_definitions SET sort_order = 140 WHERE id = '720p-hdtv-x264-none';

-- 480p/576p tiers.
UPDATE quality_definitions SET sort_order = 150 WHERE id = '576p-bluray-x264-none';
UPDATE quality_definitions SET sort_order = 160 WHERE id = '480p-bluray-x264-none';
UPDATE quality_definitions SET sort_order = 170 WHERE id = '480p-webdl-x264-none';
UPDATE quality_definitions SET sort_order = 180 WHERE id = '480p-webrip-x264-none';

-- SD tiers.
UPDATE quality_definitions SET sort_order = 190 WHERE id = 'sd-dvdr-unknown-none';
UPDATE quality_definitions SET sort_order = 200 WHERE id = 'sd-dvd-xvid-none';
UPDATE quality_definitions SET sort_order = 210 WHERE id = 'sd-hdtv-x264-none';

-- Raw / disk formats.
UPDATE quality_definitions SET sort_order = 220 WHERE id = 'unknown-brdisk-unknown-none';
UPDATE quality_definitions SET sort_order = 230 WHERE id = 'unknown-rawhd-unknown-none';

-- Pre-release (least desirable, at the bottom).
UPDATE quality_definitions SET sort_order = 240 WHERE id = 'sd-dvdscr-unknown-none';
UPDATE quality_definitions SET sort_order = 250 WHERE id = 'sd-regional-unknown-none';
UPDATE quality_definitions SET sort_order = 260 WHERE id = 'unknown-telecine-unknown-none';
UPDATE quality_definitions SET sort_order = 270 WHERE id = 'unknown-telesync-unknown-none';
UPDATE quality_definitions SET sort_order = 280 WHERE id = 'unknown-cam-unknown-none';
UPDATE quality_definitions SET sort_order = 290 WHERE id = 'unknown-workprint-unknown-none';

-- +goose Down
-- Restore original sort orders.

-- From migration 00026 (pre-release + misc).
UPDATE quality_definitions SET sort_order = 1   WHERE id = 'unknown-workprint-unknown-none';
UPDATE quality_definitions SET sort_order = 2   WHERE id = 'unknown-cam-unknown-none';
UPDATE quality_definitions SET sort_order = 3   WHERE id = 'unknown-telesync-unknown-none';
UPDATE quality_definitions SET sort_order = 4   WHERE id = 'unknown-telecine-unknown-none';
UPDATE quality_definitions SET sort_order = 5   WHERE id = 'sd-dvdscr-unknown-none';
UPDATE quality_definitions SET sort_order = 6   WHERE id = 'sd-regional-unknown-none';
UPDATE quality_definitions SET sort_order = 15  WHERE id = '480p-webdl-x264-none';
UPDATE quality_definitions SET sort_order = 16  WHERE id = '480p-webrip-x264-none';
UPDATE quality_definitions SET sort_order = 17  WHERE id = '480p-bluray-x264-none';
UPDATE quality_definitions SET sort_order = 18  WHERE id = '576p-bluray-x264-none';
UPDATE quality_definitions SET sort_order = 25  WHERE id = 'sd-dvdr-unknown-none';
UPDATE quality_definitions SET sort_order = 115 WHERE id = '2160p-hdtv-x265-hdr10';
UPDATE quality_definitions SET sort_order = 125 WHERE id = '2160p-webrip-x265-hdr10';
UPDATE quality_definitions SET sort_order = 150 WHERE id = 'unknown-brdisk-unknown-none';
UPDATE quality_definitions SET sort_order = 160 WHERE id = 'unknown-rawhd-unknown-none';

-- From migration 00011 (original 14).
UPDATE quality_definitions SET sort_order = 10  WHERE id = 'sd-dvd-xvid-none';
UPDATE quality_definitions SET sort_order = 20  WHERE id = 'sd-hdtv-x264-none';
UPDATE quality_definitions SET sort_order = 30  WHERE id = '720p-hdtv-x264-none';
UPDATE quality_definitions SET sort_order = 40  WHERE id = '720p-webdl-x264-none';
UPDATE quality_definitions SET sort_order = 50  WHERE id = '720p-webrip-x264-none';
UPDATE quality_definitions SET sort_order = 60  WHERE id = '720p-bluray-x264-none';
UPDATE quality_definitions SET sort_order = 70  WHERE id = '1080p-hdtv-x264-none';
UPDATE quality_definitions SET sort_order = 80  WHERE id = '1080p-webdl-x264-none';
UPDATE quality_definitions SET sort_order = 90  WHERE id = '1080p-webrip-x265-none';
UPDATE quality_definitions SET sort_order = 100 WHERE id = '1080p-bluray-x265-none';
UPDATE quality_definitions SET sort_order = 110 WHERE id = '1080p-remux-x265-none';
UPDATE quality_definitions SET sort_order = 120 WHERE id = '2160p-webdl-x265-hdr10';
UPDATE quality_definitions SET sort_order = 130 WHERE id = '2160p-bluray-x265-hdr10';
UPDATE quality_definitions SET sort_order = 140 WHERE id = '2160p-remux-x265-hdr10';
