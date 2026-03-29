import { useState, useRef, useEffect } from "react";
import Modal from "@/components/Modal";
import PageHeader from "@/components/PageHeader";
import { DOCS_URLS } from "@/lib/docsUrls";
import {
  useImportLists,
  useCreateImportList,
  useUpdateImportList,
  useDeleteImportList,
  useTestImportList,
  useSyncAllImportLists,
  useCreatePlexPin,
  checkPlexPin,
  useImportListPreview,
} from "@/api/importlists";
import type { ImportListPreviewItem } from "@/api/importlists";
import { useLibraries } from "@/api/libraries";
import { useQualityProfiles } from "@/api/quality-profiles";
import { useMediaServers } from "@/api/mediaservers";
import { useSearchPeople, useSearchAll } from "@/api/collections";
import type { ImportListConfig, ImportListRequest } from "@/types";

// ── Constants ──────────────────────────────────────────────────────────────────

interface KindDef {
  value: string;
  label: string;
  group: string;
  desc: string;
  zeroConfig: boolean;
  requiresPlex?: boolean;
  requiresSignup?: boolean;
}

const KINDS: KindDef[] = [
  { value: "tmdb_popular",     label: "Popular",         group: "TMDb",  desc: "Most popular movies right now",           zeroConfig: true },
  { value: "tmdb_upcoming",    label: "Upcoming",        group: "TMDb",  desc: "Movies coming soon to theaters",          zeroConfig: true },
  { value: "tmdb_now_playing", label: "Now Playing",     group: "TMDb",  desc: "Currently in theaters",                   zeroConfig: true },
  { value: "tmdb_top_rated",   label: "Top Rated",       group: "TMDb",  desc: "All-time highest rated movies",           zeroConfig: true },
  { value: "tmdb_trending",    label: "Trending",        group: "TMDb",  desc: "Trending today or this week",             zeroConfig: true },
  { value: "tmdb_list",        label: "User List",       group: "TMDb",  desc: "A specific TMDb list by ID",              zeroConfig: false },
  { value: "tmdb_person",      label: "Person",          group: "TMDb",  desc: "Filmography of an actor or director",     zeroConfig: false },
  { value: "tmdb_collection",  label: "Collection",      group: "TMDb",  desc: "A movie franchise or collection",         zeroConfig: false },
  { value: "trakt_popular",    label: "Popular",         group: "Trakt", desc: "Most popular movies on Trakt",            zeroConfig: true },
  { value: "trakt_trending",   label: "Trending",        group: "Trakt", desc: "Most watched right now",                  zeroConfig: true },
  { value: "trakt_anticipated",label: "Anticipated",     group: "Trakt", desc: "Most anticipated upcoming movies",        zeroConfig: true },
  { value: "trakt_box_office", label: "Box Office",      group: "Trakt", desc: "Current box office hits",                 zeroConfig: true },
  { value: "trakt_list",       label: "User List",       group: "Trakt", desc: "A Trakt user's watchlist or custom list", zeroConfig: false, requiresSignup: true },
  { value: "plex_watchlist",   label: "Watchlist",       group: "Plex",  desc: "Your Plex account watchlist",             zeroConfig: false, requiresPlex: true },
  { value: "stevenlu",         label: "StevenLu",        group: "Other", desc: "Curated popular movies list",             zeroConfig: true },
  { value: "mdblist",          label: "MDBList",         group: "Other", desc: "Lists from mdblist.com",                  zeroConfig: false, requiresSignup: true },
  { value: "custom_list",      label: "Custom JSON",     group: "Other", desc: "Any URL returning JSON with TMDb IDs",    zeroConfig: false },
];

const MIN_AVAIL_OPTIONS = [
  { value: "announced", label: "Announced" },
  { value: "in_cinemas", label: "In Cinemas" },
  { value: "released", label: "Released" },
];

function kindLabel(kind: string): string {
  return KINDS.find((k) => k.value === kind)?.label ?? kind;
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function strSetting(settings: Record<string, unknown>, key: string): string {
  const v = settings[key];
  return typeof v === "string" ? v : "";
}

function numSetting(settings: Record<string, unknown>, key: string, fallback: number): number {
  const v = settings[key];
  return typeof v === "number" ? v : fallback;
}

// ── Shared styles ──────────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  width: "100%",
  background: "var(--color-bg-elevated)",
  border: "1px solid var(--color-border-default)",
  borderRadius: 6,
  padding: "8px 12px",
  fontSize: 13,
  color: "var(--color-text-primary)",
  outline: "none",
  boxSizing: "border-box",
};

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 12,
  fontWeight: 500,
  color: "var(--color-text-secondary)",
  marginBottom: 6,
};

const fieldStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 0,
};

function actionBtn(color: string, bg: string): React.CSSProperties {
  return {
    background: bg,
    border: "1px solid var(--color-border-default)",
    borderRadius: 5,
    padding: "3px 10px",
    fontSize: 12,
    color,
    cursor: "pointer",
    whiteSpace: "nowrap",
  };
}

// ── Form state ─────────────────────────────────────────────────────────────────

interface FormState {
  name: string;
  kind: string;
  enabled: boolean;
  search_on_add: boolean;
  monitor: boolean;
  min_availability: string;
  quality_profile_id: string;
  library_id: string;
  // tmdb_list
  tmdb_list_id: string;
  // tmdb_person
  tmdb_person_id: string;
  tmdb_person_type: string;
  tmdb_person_name: string;
  // tmdb_collection
  tmdb_collection_id: string;
  tmdb_collection_name: string;
  // tmdb_trending
  tmdb_trending_window: string;
  // trakt_list
  trakt_username: string;
  trakt_list_type: string;
  trakt_list_slug: string;
  // plex
  plex_access_token: string;
  // mdblist
  mdblist_api_key: string;
  mdblist_list_id: string;
  // custom_list
  custom_list_url: string;
}

function emptyForm(): FormState {
  return {
    name: "",
    kind: "tmdb_popular",
    enabled: true,
    search_on_add: false,
    monitor: true,
    min_availability: "released",
    quality_profile_id: "",
    library_id: "",
    tmdb_list_id: "",
    tmdb_person_id: "",
    tmdb_person_type: "actor",
    tmdb_person_name: "",
    tmdb_collection_id: "",
    tmdb_collection_name: "",
    tmdb_trending_window: "week",
    trakt_username: "",
    trakt_list_type: "watchlist",
    trakt_list_slug: "",
    plex_access_token: "",
    mdblist_api_key: "",
    mdblist_list_id: "",
    custom_list_url: "",
  };
}

function configToForm(cfg: ImportListConfig): FormState {
  const s = cfg.settings;
  return {
    name: cfg.name,
    kind: cfg.kind,
    enabled: cfg.enabled,
    search_on_add: cfg.search_on_add,
    monitor: cfg.monitor,
    min_availability: cfg.min_availability || "released",
    quality_profile_id: cfg.quality_profile_id,
    library_id: cfg.library_id,
    tmdb_list_id: cfg.kind === "tmdb_list" ? strSetting(s, "list_id") : "",
    tmdb_person_id: cfg.kind === "tmdb_person" ? String(numSetting(s, "person_id", 0)) : "",
    tmdb_person_type: cfg.kind === "tmdb_person" ? (strSetting(s, "person_type") || "actor") : "actor",
    tmdb_person_name: cfg.kind === "tmdb_person" ? strSetting(s, "person_name") : "",
    tmdb_collection_id: cfg.kind === "tmdb_collection" ? String(numSetting(s, "collection_id", 0)) : "",
    tmdb_collection_name: cfg.kind === "tmdb_collection" ? strSetting(s, "collection_name") : "",
    tmdb_trending_window: cfg.kind === "tmdb_trending" ? (strSetting(s, "window") || "week") : "week",
    trakt_username: cfg.kind === "trakt_list" ? strSetting(s, "username") : "",
    trakt_list_type: cfg.kind === "trakt_list" ? (strSetting(s, "list_type") || "watchlist") : "watchlist",
    trakt_list_slug: cfg.kind === "trakt_list" ? strSetting(s, "list_slug") : "",
    plex_access_token: cfg.kind === "plex_watchlist" ? strSetting(s, "access_token") : "",
    mdblist_api_key: cfg.kind === "mdblist" ? strSetting(s, "api_key") : "",
    mdblist_list_id: cfg.kind === "mdblist" ? strSetting(s, "list_id") : "",
    custom_list_url: cfg.kind === "custom_list" ? strSetting(s, "url") : "",
  };
}

function formToRequest(f: FormState): ImportListRequest {
  let settings: Record<string, unknown> = {};

  switch (f.kind) {
    case "tmdb_popular":
    case "tmdb_upcoming":
    case "tmdb_now_playing":
    case "tmdb_top_rated":
      break;
    case "tmdb_trending":
      settings = { window: f.tmdb_trending_window };
      break;
    case "tmdb_list":
      settings = { list_id: f.tmdb_list_id.trim() };
      break;
    case "tmdb_person":
      settings = { person_id: parseInt(f.tmdb_person_id, 10) || 0, person_type: f.tmdb_person_type, person_name: f.tmdb_person_name };
      break;
    case "tmdb_collection":
      settings = { collection_id: parseInt(f.tmdb_collection_id, 10) || 0, collection_name: f.tmdb_collection_name };
      break;
    case "trakt_popular":
    case "trakt_trending":
    case "trakt_anticipated":
    case "trakt_box_office":
    case "stevenlu":
      break;
    case "trakt_list":
      settings.username = f.trakt_username.trim();
      settings.list_type = f.trakt_list_type;
      if (f.trakt_list_type === "custom" && f.trakt_list_slug.trim()) {
        settings.list_slug = f.trakt_list_slug.trim();
      }
      break;
    case "plex_watchlist":
      if (f.plex_access_token.trim()) settings.access_token = f.plex_access_token.trim();
      break;
    case "mdblist":
      if (f.mdblist_api_key.trim()) settings.api_key = f.mdblist_api_key.trim();
      settings.list_id = f.mdblist_list_id.trim();
      break;
    case "custom_list":
      settings.url = f.custom_list_url.trim();
      break;
  }

  return {
    name: f.name.trim(),
    kind: f.kind,
    enabled: f.enabled,
    settings,
    search_on_add: f.search_on_add,
    monitor: f.monitor,
    min_availability: f.min_availability,
    quality_profile_id: f.quality_profile_id,
    library_id: f.library_id,
  };
}

// ── Settings sub-forms ─────────────────────────────────────────────────────────

interface SubFormProps {
  form: FormState;
  set: <K extends keyof FormState>(field: K, value: FormState[K]) => void;
  editing: boolean;
  focusBorder: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => void;
  blurBorder: (e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) => void;
}

function TMDbListSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>TMDb List ID *</label>
      <input
        style={inputStyle}
        value={form.tmdb_list_id}
        onChange={(e) => set("tmdb_list_id", e.currentTarget.value)}
        onFocus={focusBorder}
        onBlur={blurBorder}
        placeholder="e.g. 8136823"
      />
      <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
        The numeric ID from a TMDb list URL.
      </p>
    </div>
  );
}

function TMDbPersonSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const { data: results } = useSearchPeople(query);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const hasSelection = form.tmdb_person_id && form.tmdb_person_id !== "0";

  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>Person *</label>
        {hasSelection ? (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              padding: "8px 12px",
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              borderRadius: 6,
            }}
          >
            <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
              {form.tmdb_person_name || `TMDb #${form.tmdb_person_id}`}
              <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginLeft: 8 }}>
                ID: {form.tmdb_person_id}
              </span>
            </span>
            <button
              type="button"
              style={{
                background: "none",
                border: "none",
                color: "var(--color-accent)",
                cursor: "pointer",
                fontSize: 12,
                padding: 0,
              }}
              onClick={() => {
                set("tmdb_person_id", "");
                set("tmdb_person_name", "");
                setQuery("");
              }}
            >
              Change
            </button>
          </div>
        ) : (
          <div ref={wrapRef} style={{ position: "relative" }}>
            <input
              style={inputStyle}
              value={query}
              onChange={(e) => {
                setQuery(e.currentTarget.value);
                setOpen(true);
              }}
              onFocus={(e) => {
                focusBorder(e);
                if (query.trim().length >= 2) setOpen(true);
              }}
              onBlur={blurBorder}
              placeholder="Search for a person..."
            />
            {open && results && results.length > 0 && (
              <div
                style={{
                  position: "absolute",
                  top: "100%",
                  left: 0,
                  right: 0,
                  background: "var(--color-bg-elevated)",
                  border: "1px solid var(--color-border-default)",
                  borderRadius: 6,
                  marginTop: 4,
                  maxHeight: 220,
                  overflowY: "auto",
                  zIndex: 10,
                  boxShadow: "0 4px 12px rgba(0,0,0,0.3)",
                }}
              >
                {results.map((p) => (
                  <PersonDropdownItem
                    key={p.person_id}
                    name={p.name}
                    dept={p.known_for_department}
                    imgPath={p.profile_path}
                    onSelect={() => {
                      set("tmdb_person_id", String(p.person_id));
                      set("tmdb_person_name", p.name);
                      setOpen(false);
                      setQuery("");
                    }}
                  />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>Credit Type</label>
        <select
          style={{ ...inputStyle, cursor: "pointer" }}
          value={form.tmdb_person_type}
          onChange={(e) => set("tmdb_person_type", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
        >
          <option value="actor">Actor (Cast)</option>
          <option value="director">Director</option>
          <option value="writer">Writer</option>
          <option value="producer">Producer</option>
        </select>
      </div>
    </>
  );
}

function PersonDropdownItem({
  name,
  dept,
  imgPath,
  onSelect,
}: {
  name: string;
  dept: string;
  imgPath: string;
  onSelect: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const img = imgPath ? `https://image.tmdb.org/t/p/w45${imgPath}` : "";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 12px",
        cursor: "pointer",
        background: hovered ? "var(--color-bg-surface)" : "transparent",
        transition: "background 0.1s",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onMouseDown={(e) => e.preventDefault()}
      onClick={onSelect}
    >
      {img ? (
        <img
          src={img}
          alt=""
          style={{ width: 30, height: 30, borderRadius: "50%", objectFit: "cover", flexShrink: 0 }}
        />
      ) : (
        <div
          style={{
            width: 30,
            height: 30,
            borderRadius: "50%",
            background: "var(--color-bg-surface)",
            flexShrink: 0,
          }}
        />
      )}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>{name}</div>
        {dept && (
          <div style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{dept}</div>
        )}
      </div>
    </div>
  );
}

function TMDbCollectionSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const { data: allResults } = useSearchAll(query);
  const results = allResults?.filter((r) => r.result_type === "franchise");
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const hasSelection = form.tmdb_collection_id && form.tmdb_collection_id !== "0";

  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Collection *</label>
      {hasSelection ? (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "8px 12px",
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
          }}
        >
          <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
            {form.tmdb_collection_name || `TMDb #${form.tmdb_collection_id}`}
            <span style={{ fontSize: 11, color: "var(--color-text-muted)", marginLeft: 8 }}>
              ID: {form.tmdb_collection_id}
            </span>
          </span>
          <button
            type="button"
            style={{
              background: "none",
              border: "none",
              color: "var(--color-accent)",
              cursor: "pointer",
              fontSize: 12,
              padding: 0,
            }}
            onClick={() => {
              set("tmdb_collection_id", "");
              set("tmdb_collection_name", "");
              setQuery("");
            }}
          >
            Change
          </button>
        </div>
      ) : (
        <div ref={wrapRef} style={{ position: "relative" }}>
          <input
            style={inputStyle}
            value={query}
            onChange={(e) => {
              setQuery(e.currentTarget.value);
              setOpen(true);
            }}
            onFocus={(e) => {
              focusBorder(e);
              if (query.trim().length >= 2) setOpen(true);
            }}
            onBlur={blurBorder}
            placeholder="Search for a collection..."
          />
          {open && results && results.length > 0 && (
            <div
              style={{
                position: "absolute",
                top: "100%",
                left: 0,
                right: 0,
                background: "var(--color-bg-elevated)",
                border: "1px solid var(--color-border-default)",
                borderRadius: 6,
                marginTop: 4,
                maxHeight: 220,
                overflowY: "auto",
                zIndex: 10,
                boxShadow: "0 4px 12px rgba(0,0,0,0.3)",
              }}
            >
              {results.map((c) => (
                <CollectionDropdownItem
                  key={c.id}
                  name={c.name}
                  subtitle={c.subtitle}
                  imgPath={c.image_path}
                  onSelect={() => {
                    set("tmdb_collection_id", String(c.id));
                    set("tmdb_collection_name", c.name);
                    setOpen(false);
                    setQuery("");
                  }}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function CollectionDropdownItem({
  name,
  subtitle,
  imgPath,
  onSelect,
}: {
  name: string;
  subtitle: string;
  imgPath: string;
  onSelect: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const img = imgPath ? `https://image.tmdb.org/t/p/w92${imgPath}` : "";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 12px",
        cursor: "pointer",
        background: hovered ? "var(--color-bg-surface)" : "transparent",
        transition: "background 0.1s",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onMouseDown={(e) => e.preventDefault()}
      onClick={onSelect}
    >
      {img ? (
        <img
          src={img}
          alt=""
          style={{ width: 36, height: 36, borderRadius: 4, objectFit: "cover", flexShrink: 0 }}
        />
      ) : (
        <div
          style={{
            width: 36,
            height: 36,
            borderRadius: 4,
            background: "var(--color-bg-surface)",
            flexShrink: 0,
          }}
        />
      )}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 13, color: "var(--color-text-primary)", fontWeight: 500 }}>{name}</div>
        {subtitle && (
          <div style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{subtitle}</div>
        )}
      </div>
    </div>
  );
}

function TraktListSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>Username *</label>
        <input
          style={inputStyle}
          value={form.trakt_username}
          onChange={(e) => set("trakt_username", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="Trakt username"
        />
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>List Type</label>
        <select
          style={{ ...inputStyle, cursor: "pointer" }}
          value={form.trakt_list_type}
          onChange={(e) => set("trakt_list_type", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
        >
          <option value="watchlist">Watchlist</option>
          <option value="custom">Custom List</option>
        </select>
      </div>
      {form.trakt_list_type === "custom" && (
        <div style={fieldStyle}>
          <label style={labelStyle}>List Slug *</label>
          <input
            style={inputStyle}
            value={form.trakt_list_slug}
            onChange={(e) => set("trakt_list_slug", e.currentTarget.value)}
            onFocus={focusBorder}
            onBlur={blurBorder}
            placeholder="e.g. my-favorite-movies"
          />
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
            The slug from the Trakt list URL.
          </p>
        </div>
      )}
    </>
  );
}

function PlexWatchlistSettings({ form, set, editing }: SubFormProps) {
  const createPin = useCreatePlexPin();
  const [polling, setPolling] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const hasToken = form.plex_access_token.length > 0;

  function startPlexAuth() {
    createPin.mutate(undefined, {
      onSuccess: (pin) => {
        window.open(pin.auth_url, "_blank", "noopener");
        setPolling(true);
        let attempts = 0;
        pollRef.current = setInterval(async () => {
          attempts++;
          try {
            const status = await checkPlexPin(pin.id);
            if (status.claimed && status.token) {
              set("plex_access_token", status.token);
              setPolling(false);
              if (pollRef.current) clearInterval(pollRef.current);
            } else if (attempts >= 60) {
              setPolling(false);
              if (pollRef.current) clearInterval(pollRef.current);
            }
          } catch {
            setPolling(false);
            if (pollRef.current) clearInterval(pollRef.current);
          }
        }, 2000);
      },
    });
  }

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  if (hasToken && !polling) {
    return (
      <div style={fieldStyle}>
        <label style={labelStyle}>Plex Account</label>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "8px 12px",
            background: "var(--color-bg-elevated)",
            border: "1px solid var(--color-border-default)",
            borderRadius: 6,
          }}
        >
          <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>
            {editing ? "Plex account linked (token saved)" : "Plex account linked"}
          </span>
          <button
            type="button"
            style={{
              background: "none",
              border: "none",
              color: "var(--color-accent)",
              cursor: "pointer",
              fontSize: 12,
              padding: 0,
            }}
            onClick={() => set("plex_access_token", "")}
          >
            Re-authenticate
          </button>
        </div>
      </div>
    );
  }

  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Plex Account *</label>
      <button
        type="button"
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 8,
          width: "100%",
          padding: "10px 16px",
          background: polling ? "var(--color-bg-elevated)" : "#e5a00d",
          color: polling ? "var(--color-text-secondary)" : "#1a1a1a",
          border: "1px solid var(--color-border-default)",
          borderRadius: 6,
          fontSize: 13,
          fontWeight: 600,
          cursor: polling || createPin.isPending ? "default" : "pointer",
          opacity: createPin.isPending ? 0.6 : 1,
        }}
        onClick={startPlexAuth}
        disabled={polling || createPin.isPending}
      >
        {polling
          ? "Waiting for Plex authorization..."
          : createPin.isPending
            ? "Creating link..."
            : "Sign in with Plex"}
      </button>
      {polling && (
        <p style={{ margin: "6px 0 0", fontSize: 11, color: "var(--color-text-muted)", textAlign: "center" }}>
          A Plex sign-in window has opened. Complete the sign-in there, then return here.
        </p>
      )}
    </div>
  );
}

function MDBListSettings({ form, set, editing, focusBorder, blurBorder }: SubFormProps) {
  return (
    <>
      <div style={fieldStyle}>
        <label style={labelStyle}>API Key *</label>
        <input
          style={inputStyle}
          type="password"
          value={form.mdblist_api_key}
          onChange={(e) => set("mdblist_api_key", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder={editing ? "enter to change" : "Your MDBList API key"}
          autoComplete="new-password"
        />
        {editing && (
          <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
            API key is masked. Enter a new value to update.
          </p>
        )}
        <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
          Get your API key from mdblist.com/preferences
        </p>
      </div>
      <div style={fieldStyle}>
        <label style={labelStyle}>List ID *</label>
        <input
          style={inputStyle}
          value={form.mdblist_list_id}
          onChange={(e) => set("mdblist_list_id", e.currentTarget.value)}
          onFocus={focusBorder}
          onBlur={blurBorder}
          placeholder="e.g. 12345"
        />
        <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
          The numeric ID from the MDBList list URL.
        </p>
      </div>
    </>
  );
}

function CustomListSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>URL *</label>
      <input
        style={inputStyle}
        value={form.custom_list_url}
        onChange={(e) => set("custom_list_url", e.currentTarget.value)}
        onFocus={focusBorder}
        onBlur={blurBorder}
        placeholder='https://example.com/movies.json'
      />
      <p style={{ margin: "4px 0 0", fontSize: 11, color: "var(--color-text-muted)" }}>
        URL returning a JSON array. Each object needs a <code style={{ fontSize: 11 }}>tmdb</code> or <code style={{ fontSize: 11 }}>tmdb_id</code> field.
      </p>
    </div>
  );
}

function TMDbTrendingSettings({ form, set, focusBorder, blurBorder }: SubFormProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>Time Window</label>
      <select
        style={{ ...inputStyle, cursor: "pointer" }}
        value={form.tmdb_trending_window}
        onChange={(e) => set("tmdb_trending_window", e.currentTarget.value)}
        onFocus={focusBorder}
        onBlur={blurBorder}
      >
        <option value="day">Today</option>
        <option value="week">This Week</option>
      </select>
    </div>
  );
}

function PreviewGrid({ items, loading }: { items: ImportListPreviewItem[]; loading?: boolean }) {
  if (loading) {
    return (
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(90px, 1fr))",
          gap: 10,
          padding: 2,
        }}
      >
        {Array.from({ length: 10 }, (_, i) => (
          <div key={i}>
            <div className="skeleton" style={{ width: "100%", aspectRatio: "2/3", borderRadius: 6 }} />
            <div className="skeleton" style={{ height: 10, borderRadius: 3, marginTop: 6, width: "70%" }} />
          </div>
        ))}
      </div>
    );
  }

  if (items.length === 0) return null;

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(90px, 1fr))",
        gap: 10,
        padding: 2,
      }}
    >
      {items.slice(0, 20).map((item) => (
        <div key={item.tmdb_id} style={{ textAlign: "center" }}>
          {item.poster_path ? (
            <img
              src={`https://image.tmdb.org/t/p/w154${item.poster_path}`}
              alt={item.title}
              style={{
                width: "100%",
                aspectRatio: "2/3",
                objectFit: "cover",
                borderRadius: 6,
                background: "var(--color-bg-elevated)",
              }}
            />
          ) : (
            <div
              style={{
                width: "100%",
                aspectRatio: "2/3",
                borderRadius: 6,
                background: "var(--color-bg-elevated)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 10,
                color: "var(--color-text-muted)",
                padding: 4,
              }}
            >
              {item.title}
            </div>
          )}
          <p
            style={{
              margin: "4px 0 0",
              fontSize: 10,
              color: "var(--color-text-muted)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {item.title} {item.year ? `(${item.year})` : ""}
          </p>
        </div>
      ))}
    </div>
  );
}

// ── Source picker card grid ─────────────────────────────────────────────────

function SourceCard({
  kind,
  selected,
  disabled,
  disabledReason,
  onClick,
}: {
  kind: KindDef;
  selected: boolean;
  disabled: boolean;
  disabledReason?: string;
  onClick: () => void;
}) {
  const [hovered, setHovered] = useState(false);

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 4,
        padding: "10px 12px",
        background: selected
          ? "color-mix(in srgb, var(--color-accent) 15%, transparent)"
          : hovered && !disabled
            ? "var(--color-bg-elevated)"
            : "transparent",
        border: selected
          ? "1px solid var(--color-accent)"
          : "1px solid var(--color-border-default)",
        borderRadius: 8,
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.4 : 1,
        textAlign: "left",
        minWidth: 0,
        transition: "background 0.1s, border-color 0.1s",
      }}
      title={disabledReason}
    >
      <span style={{ fontSize: 13, fontWeight: 500, color: "var(--color-text-primary)" }}>
        {kind.label}
      </span>
      <span style={{ fontSize: 11, color: "var(--color-text-muted)", lineHeight: 1.3 }}>
        {kind.desc}
      </span>
      {kind.requiresSignup && (
        <span style={{ fontSize: 10, color: "var(--color-warning)", marginTop: 2 }}>
          Requires external account
        </span>
      )}
    </button>
  );
}

function SourcePicker({
  selected,
  onSelect,
  hasPlexServer,
}: {
  selected: string;
  onSelect: (kind: string) => void;
  hasPlexServer: boolean;
}) {
  const groups = ["TMDb", "Trakt", "Plex", "Other"] as const;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      {groups.map((group) => {
        const kinds = KINDS.filter((k) => k.group === group);
        if (kinds.length === 0) return null;

        return (
          <div key={group}>
            <div
              style={{
                fontSize: 11,
                fontWeight: 600,
                color: "var(--color-text-muted)",
                textTransform: "uppercase",
                letterSpacing: "0.05em",
                marginBottom: 8,
              }}
            >
              {group}
            </div>
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
                gap: 8,
              }}
            >
              {kinds.map((k) => {
                const plexDisabled = !!k.requiresPlex && !hasPlexServer;
                const disabled = plexDisabled;
                const disabledReason = plexDisabled
                  ? "Connect a Plex media server first (Settings > Media Servers)"
                  : undefined;

                return (
                  <SourceCard
                    key={k.value}
                    kind={k}
                    selected={selected === k.value}
                    disabled={disabled}
                    disabledReason={disabledReason}
                    onClick={() => onSelect(k.value)}
                  />
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function KindSettings(props: SubFormProps) {
  switch (props.form.kind) {
    case "tmdb_popular":
    case "tmdb_upcoming":
    case "tmdb_now_playing":
    case "tmdb_top_rated":
      return null;
    case "tmdb_trending":
      return <TMDbTrendingSettings {...props} />;
    case "tmdb_list":
      return <TMDbListSettings {...props} />;
    case "tmdb_person":
      return <TMDbPersonSettings {...props} />;
    case "tmdb_collection":
      return <TMDbCollectionSettings {...props} />;
    case "trakt_popular":
    case "trakt_trending":
    case "trakt_anticipated":
    case "trakt_box_office":
    case "stevenlu":
      return null;
    case "trakt_list":
      return <TraktListSettings {...props} />;
    case "plex_watchlist":
      return <PlexWatchlistSettings {...props} />;
    case "mdblist":
      return <MDBListSettings {...props} />;
    case "custom_list":
      return <CustomListSettings {...props} />;
    default:
      return null;
  }
}

// ── Main component ─────────────────────────────────────────────────────────────

export default function ImportListList() {
  const { data: lists, isLoading, error } = useImportLists();
  const { data: libraries } = useLibraries();
  const { data: profiles } = useQualityProfiles();
  const { data: mediaServers } = useMediaServers();
  const createMut = useCreateImportList();
  const updateMut = useUpdateImportList();
  const deleteMut = useDeleteImportList();
  const testMut = useTestImportList();
  const syncMut = useSyncAllImportLists();
  const previewMut = useImportListPreview();

  const [editId, setEditId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [showModal, setShowModal] = useState(false);
  // "pick" = source selection card grid, "config" = form + preview
  const [modalStep, setModalStep] = useState<"pick" | "config">("pick");
  const [previewItems, setPreviewItems] = useState<ImportListPreviewItem[]>([]);

  const hasPlexServer = (mediaServers ?? []).some((s) => s.kind === "plex" && s.enabled);

  function set<K extends keyof FormState>(field: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  function loadPreview(kind: string, settings?: Record<string, unknown>) {
    setPreviewItems([]);
    previewMut.mutate({ kind, settings }, {
      onSuccess: (items) => setPreviewItems(items ?? []),
    });
  }

  function openCreate() {
    setEditId(null);
    const f = emptyForm();
    if (libraries?.length) f.library_id = libraries[0].id;
    if (profiles?.length) f.quality_profile_id = profiles[0].id;
    setForm(f);
    setPreviewItems([]);
    setModalStep("pick");
    setShowModal(true);
  }

  function openEdit(cfg: ImportListConfig) {
    setEditId(cfg.id);
    setForm(configToForm(cfg));
    setPreviewItems([]);
    setModalStep("config");
    setShowModal(true);
  }

  function closeModal() {
    setShowModal(false);
    setEditId(null);
    setPreviewItems([]);
  }

  function handleSourceSelect(kind: string) {
    set("kind", kind);
    const kindDef = KINDS.find((k) => k.value === kind);
    // Auto-set name based on kind if empty
    if (!form.name || form.name === "" || KINDS.some((k) => k.label === form.name || `${k.group} ${k.label}` === form.name)) {
      const autoName = kindDef ? `${kindDef.group} ${kindDef.label}` : kind;
      set("name", autoName);
    }
    setModalStep("config");
    // Auto-load preview for zero-config sources
    if (kindDef?.zeroConfig) {
      const settings = kind === "tmdb_trending" ? { window: form.tmdb_trending_window } : undefined;
      loadPreview(kind, settings);
    }
  }

  function handleSave() {
    const req = formToRequest(form);
    if (editId) {
      updateMut.mutate({ id: editId, ...req }, { onSuccess: closeModal });
    } else {
      createMut.mutate(req, { onSuccess: closeModal });
    }
  }

  function focusBorder(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    e.currentTarget.style.borderColor = "var(--color-accent)";
  }
  function blurBorder(e: React.FocusEvent<HTMLInputElement | HTMLSelectElement>) {
    e.currentTarget.style.borderColor = "var(--color-border-default)";
  }

  const saving = createMut.isPending || updateMut.isPending;
  const currentKindDef = KINDS.find((k) => k.value === form.kind);

  // ── Loading / error states ─────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div style={{ padding: "32px 40px", maxWidth: 800 }}>
        <h2 style={{ fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", margin: 0 }}>Import Lists</h2>
        <div style={{ display: "flex", flexDirection: "column", gap: 12, marginTop: 20 }}>
          {[1, 2, 3].map((i) => (
            <div key={i} className="skeleton" style={{ height: 56, borderRadius: 8 }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "32px 40px", maxWidth: 800 }}>
        <h2 style={{ fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", margin: 0 }}>Import Lists</h2>
        <p style={{ color: "var(--color-text-danger)", marginTop: 16 }}>
          Failed to load import lists: {(error as Error).message}
        </p>
      </div>
    );
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div style={{ padding: "32px 40px", maxWidth: 800 }}>
      <PageHeader
        title="Import Lists"
        description="Automatically add movies from external lists and sources."
        docsUrl={DOCS_URLS.importLists}
        action={
          <div style={{ display: "flex", gap: 8 }}>
            <button
              style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
              onClick={() => syncMut.mutate()}
              disabled={syncMut.isPending}
            >
              {syncMut.isPending ? "Syncing..." : "Sync All"}
            </button>
            <button
              style={{
                ...actionBtn("white", "var(--color-accent)"),
                border: "none",
                fontWeight: 500,
              }}
              onClick={openCreate}
            >
              + Add
            </button>
          </div>
        }
      />

      {/* Empty state */}
      {(!lists || lists.length === 0) && (
        <div
          style={{
            textAlign: "center",
            padding: "48px 24px",
            color: "var(--color-text-muted)",
            background: "var(--color-bg-elevated)",
            border: "1px dashed var(--color-border-default)",
            borderRadius: 8,
          }}
        >
          <p style={{ fontSize: 14, margin: 0 }}>No import lists configured.</p>
          <p style={{ fontSize: 12, marginTop: 6 }}>
            Import lists automatically add movies from external sources like TMDb, Trakt, and Plex.
          </p>
        </div>
      )}

      {/* List */}
      {lists && lists.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {lists.map((cfg) => (
            <ImportListRow
              key={cfg.id}
              cfg={cfg}
              onEdit={() => openEdit(cfg)}
              onDelete={() => { if (confirm(`Delete "${cfg.name}"?`)) deleteMut.mutate(cfg.id); }}
              onTest={() => testMut.mutate(cfg.id)}
              testing={testMut.isPending && testMut.variables === cfg.id}
            />
          ))}
        </div>
      )}

      {/* Modal */}
      {showModal && (
        <Modal onClose={closeModal} width={modalStep === "pick" && !editId ? 640 : 580}>
          {/* ── Step 1: Source picker ─────────────────────────────────────── */}
          {modalStep === "pick" && !editId && (
            <>
              <div style={{ padding: "20px 24px 0", borderBottom: "1px solid var(--color-border-subtle)" }}>
                <h3 style={{ margin: "0 0 16px", fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
                  Choose a Source
                </h3>
              </div>
              <div
                style={{
                  padding: "20px 24px",
                  overflowY: "auto",
                  maxHeight: "calc(100vh - 220px)",
                }}
              >
                <SourcePicker
                  selected={form.kind}
                  onSelect={handleSourceSelect}
                  hasPlexServer={hasPlexServer}
                />
              </div>
            </>
          )}

          {/* ── Step 2: Config form + preview ────────────────────────────── */}
          {(modalStep === "config" || editId) && (
            <>
              <div style={{ padding: "20px 24px 0", borderBottom: "1px solid var(--color-border-subtle)" }}>
                <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 16 }}>
                  {!editId && (
                    <button
                      type="button"
                      onClick={() => { setModalStep("pick"); setPreviewItems([]); }}
                      style={{
                        background: "none",
                        border: "none",
                        color: "var(--color-text-muted)",
                        cursor: "pointer",
                        fontSize: 16,
                        padding: 0,
                        lineHeight: 1,
                      }}
                      title="Back to source selection"
                    >
                      &larr;
                    </button>
                  )}
                  <h3 style={{ margin: 0, fontSize: 16, fontWeight: 600, color: "var(--color-text-primary)" }}>
                    {editId ? "Edit Import List" : currentKindDef ? `${currentKindDef.group} ${currentKindDef.label}` : "Configure List"}
                  </h3>
                  {currentKindDef && (
                    <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>
                      {currentKindDef.desc}
                    </span>
                  )}
                </div>
              </div>
              <div
                style={{
                  padding: "20px 24px",
                  display: "flex",
                  flexDirection: "column",
                  gap: 16,
                  overflowY: "auto",
                  maxHeight: "calc(100vh - 220px)",
                }}
              >
                {/* Preview for zero-config sources */}
                {(previewItems.length > 0 || previewMut.isPending) && (
                  <div>
                    <div style={{ fontSize: 12, fontWeight: 500, color: "var(--color-text-secondary)", marginBottom: 8 }}>
                      Preview — movies that will be imported
                    </div>
                    <PreviewGrid items={previewItems} loading={previewMut.isPending} />
                  </div>
                )}

                {/* Name */}
                <div style={fieldStyle}>
                  <label style={labelStyle}>Name *</label>
                  <input
                    style={inputStyle}
                    value={form.name}
                    onChange={(e) => set("name", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                    placeholder="e.g. TMDb Popular"
                  />
                </div>

                {/* Plugin-specific settings */}
                <KindSettings
                  form={form}
                  set={set}
                  editing={!!editId}
                  focusBorder={focusBorder}
                  blurBorder={blurBorder}
                />

                {/* Separator */}
                <hr style={{ border: "none", borderTop: "1px solid var(--color-border-subtle)", margin: "4px 0" }} />

                {/* Library */}
                <div style={fieldStyle}>
                  <label style={labelStyle}>Library *</label>
                  <select
                    style={{ ...inputStyle, cursor: "pointer" }}
                    value={form.library_id}
                    onChange={(e) => set("library_id", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                  >
                    <option value="">Select a library...</option>
                    {libraries?.map((lib) => (
                      <option key={lib.id} value={lib.id}>{lib.name}</option>
                    ))}
                  </select>
                </div>

                {/* Quality Profile */}
                <div style={fieldStyle}>
                  <label style={labelStyle}>Quality Profile *</label>
                  <select
                    style={{ ...inputStyle, cursor: "pointer" }}
                    value={form.quality_profile_id}
                    onChange={(e) => set("quality_profile_id", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                  >
                    <option value="">Select a profile...</option>
                    {profiles?.map((p) => (
                      <option key={p.id} value={p.id}>{p.name}</option>
                    ))}
                  </select>
                </div>

                {/* Minimum Availability */}
                <div style={fieldStyle}>
                  <label style={labelStyle}>Minimum Availability</label>
                  <select
                    style={{ ...inputStyle, cursor: "pointer" }}
                    value={form.min_availability}
                    onChange={(e) => set("min_availability", e.currentTarget.value)}
                    onFocus={focusBorder}
                    onBlur={blurBorder}
                  >
                    {MIN_AVAIL_OPTIONS.map((o) => (
                      <option key={o.value} value={o.value}>{o.label}</option>
                    ))}
                  </select>
                </div>

                {/* Checkboxes */}
                <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
                  <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
                    <input
                      type="checkbox"
                      checked={form.enabled}
                      onChange={(e) => set("enabled", e.currentTarget.checked)}
                      style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
                    />
                    <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>Enabled</span>
                  </label>
                  <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
                    <input
                      type="checkbox"
                      checked={form.monitor}
                      onChange={(e) => set("monitor", e.currentTarget.checked)}
                      style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
                    />
                    <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>Monitor added movies</span>
                  </label>
                  <label style={{ display: "flex", alignItems: "center", gap: 8, cursor: "pointer", userSelect: "none" }}>
                    <input
                      type="checkbox"
                      checked={form.search_on_add}
                      onChange={(e) => set("search_on_add", e.currentTarget.checked)}
                      style={{ width: 16, height: 16, cursor: "pointer", accentColor: "var(--color-accent)" }}
                    />
                    <span style={{ fontSize: 13, color: "var(--color-text-primary)" }}>Search on add</span>
                    <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
                      — auto-search for releases when movies are added
                    </span>
                  </label>
                </div>
              </div>

              {/* Footer */}
              <div
                style={{
                  display: "flex",
                  justifyContent: "flex-end",
                  gap: 8,
                  padding: "16px 24px",
                  borderTop: "1px solid var(--color-border-subtle)",
                }}
              >
                <button
                  style={actionBtn("var(--color-text-secondary)", "transparent")}
                  onClick={closeModal}
                >
                  Cancel
                </button>
                <button
                  style={{
                    ...actionBtn("white", "var(--color-accent)"),
                    border: "none",
                    fontWeight: 500,
                    opacity: saving ? 0.6 : 1,
                  }}
                  onClick={handleSave}
                  disabled={saving || !form.name.trim()}
                >
                  {saving ? "Saving..." : "Save"}
                </button>
              </div>
            </>
          )}
        </Modal>
      )}
    </div>
  );
}

// ── Row component ──────────────────────────────────────────────────────────────

function ImportListRow({
  cfg,
  onEdit,
  onDelete,
  onTest,
  testing,
}: {
  cfg: ImportListConfig;
  onEdit: () => void;
  onDelete: () => void;
  onTest: () => void;
  testing: boolean;
}) {
  const [hovered, setHovered] = useState(false);

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: "12px 16px",
        background: hovered ? "var(--color-bg-elevated)" : "var(--color-bg-surface)",
        border: "1px solid var(--color-border-default)",
        borderRadius: 8,
        transition: "background 0.15s",
        cursor: "pointer",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={onEdit}
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 2, minWidth: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span
            style={{
              display: "inline-block",
              width: 8,
              height: 8,
              borderRadius: "50%",
              background: cfg.enabled ? "var(--color-success)" : "var(--color-text-muted)",
              flexShrink: 0,
            }}
          />
          <span style={{ fontSize: 14, fontWeight: 500, color: "var(--color-text-primary)" }}>
            {cfg.name}
          </span>
          <span
            style={{
              fontSize: 11,
              color: "var(--color-text-muted)",
              background: "var(--color-bg-elevated)",
              padding: "1px 6px",
              borderRadius: 4,
              flexShrink: 0,
            }}
          >
            {kindLabel(cfg.kind)}
          </span>
        </div>
        <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>
          {cfg.monitor ? "Monitored" : "Unmonitored"}
          {cfg.search_on_add ? " \u00b7 Search on add" : ""}
        </span>
      </div>

      <div
        style={{ display: "flex", gap: 6, flexShrink: 0 }}
        onClick={(e) => e.stopPropagation()}
      >
        <button
          style={actionBtn("var(--color-text-secondary)", "var(--color-bg-elevated)")}
          onClick={onTest}
          disabled={testing}
        >
          {testing ? "Testing..." : "Test"}
        </button>
        <button
          style={actionBtn("var(--color-text-danger)", "transparent")}
          onClick={onDelete}
        >
          Delete
        </button>
      </div>
    </div>
  );
}
