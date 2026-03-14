-- +goose Up
-- Add 17 missing quality tiers to cover all Radarr-equivalent quality levels.

-- Pre-release sources (lowest → highest within group).
INSERT INTO quality_definitions (id, name, resolution, source, codec, hdr, min_size, max_size, preferred_size, sort_order) VALUES
  ('unknown-workprint-unknown-none', 'Workprint',  'unknown', 'workprint', 'unknown', 'none', 0, 0, 0, 1),
  ('unknown-cam-unknown-none',      'CAM',         'unknown', 'cam',       'unknown', 'none', 0, 0, 0, 2),
  ('unknown-telesync-unknown-none', 'Telesync',    'unknown', 'telesync',  'unknown', 'none', 0, 0, 0, 3),
  ('unknown-telecine-unknown-none', 'Telecine',    'unknown', 'telecine',  'unknown', 'none', 0, 0, 0, 4),
  ('sd-dvdscr-unknown-none',        'DVDSCR',     'sd',      'dvdscr',    'unknown', 'none', 0, 3, 0, 5),
  ('sd-regional-unknown-none',      'Regional',   'sd',      'regional',  'unknown', 'none', 0, 3, 0, 6);

-- 480p/576p variants.
INSERT INTO quality_definitions (id, name, resolution, source, codec, hdr, min_size, max_size, preferred_size, sort_order) VALUES
  ('480p-webdl-x264-none',   '480p WEBDL',   '480p', 'webdl',  'x264', 'none', 0, 3,  3,  15),
  ('480p-webrip-x264-none',  '480p WEBRip',  '480p', 'webrip', 'x264', 'none', 0, 3,  3,  16),
  ('480p-bluray-x264-none',  '480p Bluray',  '480p', 'bluray', 'x264', 'none', 0, 5,  5,  17),
  ('576p-bluray-x264-none',  '576p Bluray',  '576p', 'bluray', 'x264', 'none', 0, 5,  5,  18);

-- Other missing tiers.
INSERT INTO quality_definitions (id, name, resolution, source, codec, hdr, min_size, max_size, preferred_size, sort_order) VALUES
  ('sd-dvdr-unknown-none',          'DVD-R',         'sd',      'dvdr',   'unknown', 'none',  0,  0,   0,  25),
  ('2160p-hdtv-x265-hdr10',        '2160p HDTV HDR','2160p',   'hdtv',   'x265',   'hdr10', 15, 250, 250, 115),
  ('2160p-webrip-x265-hdr10',      '2160p WEBRip HDR','2160p', 'webrip', 'x265',   'hdr10', 15, 250, 250, 125),
  ('unknown-brdisk-unknown-none',   'BR-DISK',       'unknown', 'brdisk', 'unknown', 'none',  0,  0,   0, 150),
  ('unknown-rawhd-unknown-none',    'Raw-HD',        'unknown', 'rawhd',  'unknown', 'none',  0,  0,   0, 160);

-- +goose Down
DELETE FROM quality_definitions WHERE id IN (
  'unknown-workprint-unknown-none',
  'unknown-cam-unknown-none',
  'unknown-telesync-unknown-none',
  'unknown-telecine-unknown-none',
  'sd-dvdscr-unknown-none',
  'sd-regional-unknown-none',
  '480p-webdl-x264-none',
  '480p-webrip-x264-none',
  '480p-bluray-x264-none',
  '576p-bluray-x264-none',
  'sd-dvdr-unknown-none',
  '2160p-hdtv-x265-hdr10',
  '2160p-webrip-x265-hdr10',
  'unknown-brdisk-unknown-none',
  'unknown-rawhd-unknown-none'
);
