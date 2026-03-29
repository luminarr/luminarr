const DOCS_BASE = "https://luminarr.video/docs";

export const DOCS_URLS = {
  // Settings pages
  libraries: `${DOCS_BASE}/libraries`,
  qualityProfiles: `${DOCS_BASE}/quality-profiles`,
  qualityDefinitions: `${DOCS_BASE}/quality-definitions`,
  customFormats: `${DOCS_BASE}/custom-formats`,
  indexers: `${DOCS_BASE}/indexers`,
  downloadClients: `${DOCS_BASE}/download-clients`,
  notifications: `${DOCS_BASE}/notifications`,
  mediaServers: `${DOCS_BASE}/media-servers`,
  importLists: `${DOCS_BASE}/import-lists`,
  importExclusions: `${DOCS_BASE}/import-exclusions`,
  blocklist: `${DOCS_BASE}/blocklist`,
  mediaManagement: `${DOCS_BASE}/media-management`,
  mediaScanning: `${DOCS_BASE}/media-scanning`,
  appSettings: `${DOCS_BASE}/app-settings`,
  tags: `${DOCS_BASE}/tags`,
  system: `${DOCS_BASE}/system`,

  // Main pages
  discover: `${DOCS_BASE}/discover`,
  wanted: `${DOCS_BASE}/wanted`,
  activity: `${DOCS_BASE}/activity`,
  calendar: `${DOCS_BASE}/calendar`,
} as const;
