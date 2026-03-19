import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import {
  useCollectionStats,
  useQualityStats,
  useStorageStats,
  useGrabStats,
  useDecadeStats,
  useGrowthStats,
  useGenreStats,
} from "@/api/stats";
import type {
  CollectionStats,
  QualityBucket,
  StorageStats,
  GrabStats,
  DecadeBucket,
  GrowthPoint,
  GenreBucket,
} from "@/api/stats";
import { formatBytes } from "@/lib/utils";
const tooltipStyle = {
  contentStyle: {
    background: "var(--color-bg-elevated)",
    border: "1px solid var(--color-border-subtle)",
    borderRadius: 8,
    fontSize: 12,
    color: "var(--color-text-primary)",
  },
  // Disable the CSS transition so the tooltip snaps to cursor position
  // instead of animating/chasing it, which causes visible jitter.
  wrapperStyle: { transition: "none" },
  cursor: { fill: "color-mix(in srgb, var(--color-accent) 8%, transparent)" },
};

const axisStyle = { fontSize: 11, fill: "var(--color-text-muted)" };

// ── Skeleton ──────────────────────────────────────────────────────────────────

function CardSkeleton({ height = 220 }: { height?: number }) {
  return (
    <div
      className="skeleton"
      style={{ borderRadius: 12, height, background: "var(--color-bg-elevated)" }}
    />
  );
}

// ── Card shell ────────────────────────────────────────────────────────────────

function Card({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        background: "var(--color-bg-elevated)",
        borderRadius: 12,
        border: "1px solid var(--color-border-subtle)",
        padding: "20px 24px",
      }}
    >
      <h2
        style={{
          margin: "0 0 18px",
          fontSize: 13,
          fontWeight: 600,
          color: "var(--color-text-muted)",
          textTransform: "uppercase",
          letterSpacing: "0.08em",
        }}
      >
        {title}
      </h2>
      {children}
    </div>
  );
}

// ── Stat block ────────────────────────────────────────────────────────────────

function StatBlock({
  label,
  value,
  accent,
}: {
  label: string;
  value: string | number;
  accent?: string;
}) {
  return (
    <div style={{ flex: 1, minWidth: 100 }}>
      <div
        style={{
          fontSize: 28,
          fontWeight: 700,
          color: accent ?? "var(--color-text-primary)",
          lineHeight: 1,
          marginBottom: 6,
        }}
      >
        {value}
      </div>
      <div style={{ fontSize: 12, color: "var(--color-text-muted)", fontWeight: 500 }}>
        {label}
      </div>
    </div>
  );
}

// ── Empty state ───────────────────────────────────────────────────────────────

function EmptyChart({ message }: { message: string }) {
  return (
    <p style={{ color: "var(--color-text-muted)", fontSize: 13, margin: "8px 0 0" }}>
      {message}
    </p>
  );
}

// ── Collection card ───────────────────────────────────────────────────────────

function CollectionCard({ data }: { data: CollectionStats }) {
  return (
    <Card title="Collection">
      <div style={{ display: "flex", gap: 24, flexWrap: "wrap" }}>
        <StatBlock label="Total Movies" value={data.total_movies.toLocaleString()} />
        <StatBlock label="Monitored" value={data.monitored.toLocaleString()} />
        <StatBlock label="Have File" value={data.with_file.toLocaleString()} />
        <StatBlock
          label="Missing"
          value={data.missing.toLocaleString()}
          accent={data.missing > 0 ? "var(--color-warning)" : undefined}
        />
        <StatBlock
          label="Needs Upgrade"
          value={data.needs_upgrade.toLocaleString()}
          accent={data.needs_upgrade > 0 ? "var(--color-warning)" : undefined}
        />
        <StatBlock
          label="Added Last 30d"
          value={data.recently_added.toLocaleString()}
          accent={data.recently_added > 0 ? "var(--color-success)" : undefined}
        />
      </div>
    </Card>
  );
}

// ── Decades card ──────────────────────────────────────────────────────────────

function DecadesCard({ data }: { data: DecadeBucket[] }) {
  if (data.length === 0) {
    return (
      <Card title="Movies by Decade">
        <EmptyChart message="Add movies to see your decade breakdown." />
      </Card>
    );
  }
  return (
    <Card title="Movies by Decade">
      <ResponsiveContainer width="100%" height={180}>
        <BarChart data={data} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="var(--color-border-subtle)"
            vertical={false}
          />
          <XAxis dataKey="decade" tick={axisStyle} axisLine={false} tickLine={false} />
          <YAxis tick={axisStyle} axisLine={false} tickLine={false} allowDecimals={false} />
          <Tooltip
            contentStyle={tooltipStyle.contentStyle}
            wrapperStyle={tooltipStyle.wrapperStyle}
            cursor={tooltipStyle.cursor}
            formatter={(v: number | undefined) => [(v ?? 0).toLocaleString(), "Movies"]}
          />
          <Bar dataKey="count" fill="var(--color-accent)" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </Card>
  );
}

// ── Growth card ───────────────────────────────────────────────────────────────

function GrowthCard({ data }: { data: GrowthPoint[] }) {
  if (data.length < 2) {
    return (
      <Card title="Library Growth">
        <EmptyChart message="Keep adding movies — growth chart will appear here." />
      </Card>
    );
  }

  // Format month labels as "Jan 25"
  const chartData = data.map((p) => ({
    ...p,
    label: p.month
      ? new Date(p.month + "-01").toLocaleDateString(undefined, {
          month: "short",
          year: "2-digit",
        })
      : p.month,
  }));

  return (
    <Card title="Library Growth">
      <ResponsiveContainer width="100%" height={180}>
        <AreaChart data={chartData} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
          <defs>
            <linearGradient id="growthGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--color-accent)" stopOpacity={0.25} />
              <stop offset="95%" stopColor="var(--color-accent)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="var(--color-border-subtle)"
            vertical={false}
          />
          <XAxis
            dataKey="label"
            tick={axisStyle}
            axisLine={false}
            tickLine={false}
            interval="preserveStartEnd"
          />
          <YAxis tick={axisStyle} axisLine={false} tickLine={false} allowDecimals={false} />
          <Tooltip
            contentStyle={tooltipStyle.contentStyle}
            wrapperStyle={tooltipStyle.wrapperStyle}
            formatter={(v: number | undefined, name: string | undefined) => [
              (v ?? 0).toLocaleString(),
              name === "cumulative" ? "Total" : "Added",
            ]}
            labelFormatter={(label) => label}
          />
          <Area
            type="monotone"
            dataKey="cumulative"
            stroke="var(--color-accent)"
            strokeWidth={2}
            fill="url(#growthGrad)"
            dot={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </Card>
  );
}

// ── Quality card ──────────────────────────────────────────────────────────────

const RESOLUTION_ORDER = ["2160p", "1080p", "720p", "SD", "unknown"];
const SOURCE_ORDER = ["Remux", "Bluray", "WebDL", "WEBRip", "HDTV", "unknown"];
const CODEC_ORDER = ["AV1", "x265", "x264", "unknown"];
const HDR_ORDER = ["DolbyVision", "HDR10", "HDR10+", "HLG", "none", "unknown"];

function aggregateBy(buckets: QualityBucket[], key: keyof QualityBucket) {
  const map: Record<string, number> = {};
  for (const b of buckets) {
    const k = b[key] as string;
    map[k] = (map[k] ?? 0) + b.count;
  }
  return Object.entries(map).map(([label, count]) => ({ label, count }));
}

function sortedGroup(
  buckets: QualityBucket[],
  key: keyof QualityBucket,
  order: string[]
) {
  const items = aggregateBy(buckets, key);
  return items
    .sort((a, b) => {
      const ai = order.indexOf(a.label);
      const bi = order.indexOf(b.label);
      if (ai === -1 && bi === -1) return b.count - a.count;
      if (ai === -1) return 1;
      if (bi === -1) return -1;
      return ai - bi;
    })
    .filter((it) => it.count > 0);
}

// Tier = resolution + source combination
function buildTiers(buckets: QualityBucket[]) {
  const map: Record<string, { resolution: string; source: string; count: number }> = {};
  for (const b of buckets) {
    const key = `${b.resolution}|${b.source}`;
    if (!map[key]) map[key] = { resolution: b.resolution, source: b.source, count: 0 };
    map[key].count += b.count;
  }
  return Object.values(map)
    .filter((t) => t.count > 0)
    .sort((a, b) => {
      const ra = RESOLUTION_ORDER.indexOf(a.resolution);
      const rb = RESOLUTION_ORDER.indexOf(b.resolution);
      if (ra !== rb) {
        if (ra === -1) return 1;
        if (rb === -1) return -1;
        return ra - rb;
      }
      const sa = SOURCE_ORDER.indexOf(a.source);
      const sb = SOURCE_ORDER.indexOf(b.source);
      if (sa === -1 && sb === -1) return b.count - a.count;
      if (sa === -1) return 1;
      if (sb === -1) return -1;
      return sa - sb;
    })
    .map((t) => ({ label: `${t.resolution} ${t.source}`, count: t.count, resolution: t.resolution, source: t.source }));
}

type QualityDimension = "dimension" | "tier";

function QualityMiniChart({
  title,
  data,
  onBarClick,
}: {
  title: string;
  data: { label: string; count: number }[];
  onBarClick?: (label: string) => void;
}) {
  if (data.length === 0) return null;
  return (
    <div>
      <div
        style={{
          fontSize: 11,
          fontWeight: 600,
          color: "var(--color-text-muted)",
          textTransform: "uppercase",
          letterSpacing: "0.07em",
          marginBottom: 8,
        }}
      >
        {title}
      </div>
      <ResponsiveContainer width="100%" height={data.length * 28 + 8}>
        <BarChart
          data={data}
          layout="vertical"
          margin={{ top: 0, right: 40, left: 0, bottom: 0 }}
          style={onBarClick ? { cursor: "pointer" } : undefined}
          onClick={onBarClick ? (payload) => {
            const label = payload?.activePayload?.[0]?.payload?.label as string | undefined;
            if (label) onBarClick(label);
          } : undefined}
        >
          <XAxis type="number" hide />
          <YAxis
            type="category"
            dataKey="label"
            tick={axisStyle}
            axisLine={false}
            tickLine={false}
            width={72}
          />
          <Tooltip
            contentStyle={tooltipStyle.contentStyle}
            wrapperStyle={tooltipStyle.wrapperStyle}
            cursor={tooltipStyle.cursor}
            formatter={(v: number | undefined) => [(v ?? 0).toLocaleString(), "Files"]}
          />
          <Bar dataKey="count" fill="var(--color-accent)" radius={[0, 4, 4, 0]}>
            {data.map((_, i) => (
              <Cell
                key={i}
                fill="var(--color-accent)"
                fillOpacity={1 - i * (0.5 / Math.max(data.length - 1, 1))}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

function QualityCard({ data }: { data: QualityBucket[] }) {
  const navigate = useNavigate();
  const [view, setView] = useState<QualityDimension>("dimension");

  const total = data.reduce((s, b) => s + b.count, 0);
  if (total === 0) {
    return (
      <Card title="Quality Distribution">
        <EmptyChart message="No movie files yet." />
      </Card>
    );
  }

  const resolutions = sortedGroup(data, "resolution", RESOLUTION_ORDER);
  const sources = sortedGroup(data, "source", SOURCE_ORDER);
  const codecs = sortedGroup(data, "codec", CODEC_ORDER);
  const hdrs = sortedGroup(data, "hdr", HDR_ORDER);
  const tiers = buildTiers(data);

  function handleTierClick(label: string) {
    const tier = tiers.find((t) => t.label === label);
    if (!tier) return;
    const params = new URLSearchParams();
    params.set("quality_resolution", tier.resolution);
    params.set("quality_source", tier.source);
    navigate(`/?${params.toString()}`);
  }

  const toggleStyle = (active: boolean): React.CSSProperties => ({
    background: active ? "var(--color-bg-elevated)" : "transparent",
    border: active ? "1px solid var(--color-border-default)" : "1px solid transparent",
    borderRadius: 5,
    padding: "3px 10px",
    fontSize: 11,
    fontWeight: active ? 600 : 400,
    color: active ? "var(--color-text-primary)" : "var(--color-text-muted)",
    cursor: "pointer",
  });

  return (
    <Card title="Quality Distribution">
      {/* View toggle */}
      <div style={{ display: "flex", gap: 4, marginBottom: 20 }}>
        <button style={toggleStyle(view === "dimension")} onClick={() => setView("dimension")}>
          By Dimension
        </button>
        <button style={toggleStyle(view === "tier")} onClick={() => setView("tier")}>
          By Tier
        </button>
      </div>

      {view === "dimension" ? (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))",
            gap: 24,
          }}
        >
          <QualityMiniChart title="Resolution" data={resolutions} />
          <QualityMiniChart title="Source" data={sources} />
          <QualityMiniChart title="Codec" data={codecs} />
          <QualityMiniChart title="HDR" data={hdrs} />
        </div>
      ) : (
        <div>
          <p style={{ margin: "0 0 10px", fontSize: 11, color: "var(--color-text-muted)" }}>
            Click a tier to filter the movie library.
          </p>
          <QualityMiniChart
            title="Resolution + Source"
            data={tiers}
            onBarClick={handleTierClick}
          />
        </div>
      )}
    </Card>
  );
}

// ── Storage card ──────────────────────────────────────────────────────────────

function StorageCard({ data }: { data: StorageStats }) {
  const trendData = (data.trend ?? []).map((p) => ({
    bytes: p.total_bytes,
    label: new Date(p.captured_at).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    }),
  }));

  return (
    <Card title="Storage">
      <div style={{ display: "flex", gap: 32, flexWrap: "wrap", marginBottom: 16 }}>
        <StatBlock label="Total Used" value={formatBytes(data.total_bytes)} />
        <StatBlock label="Files" value={data.file_count.toLocaleString()} />
        {data.file_count > 0 && (
          <StatBlock
            label="Avg per File"
            value={formatBytes(Math.round(data.total_bytes / data.file_count))}
          />
        )}
      </div>

      {trendData.length >= 2 ? (
        <>
          <div
            style={{ fontSize: 11, color: "var(--color-text-muted)", marginBottom: 6 }}
          >
            Storage over time
          </div>
          <ResponsiveContainer width="100%" height={100}>
            <AreaChart data={trendData} margin={{ top: 4, right: 0, left: -32, bottom: 0 }}>
              <defs>
                <linearGradient id="storageGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-accent)" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="var(--color-accent)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis
                dataKey="label"
                tick={axisStyle}
                axisLine={false}
                tickLine={false}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={axisStyle}
                axisLine={false}
                tickLine={false}
                tickFormatter={formatBytes}
              />
              <Tooltip
                contentStyle={tooltipStyle.contentStyle}
                wrapperStyle={tooltipStyle.wrapperStyle}
                formatter={(v: number | undefined) => [formatBytes(v ?? 0), "Storage"]}
              />
              <Area
                type="monotone"
                dataKey="bytes"
                stroke="var(--color-accent)"
                strokeWidth={2}
                fill="url(#storageGrad)"
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </>
      ) : (
        <EmptyChart message="Trend data is collecting — check back tomorrow." />
      )}
    </Card>
  );
}

// ── Grabs card ────────────────────────────────────────────────────────────────

const GRAB_COLORS = {
  success: "var(--color-success)",
  failed: "var(--color-danger, #ef4444)",
};

function GrabsCard({ data }: { data: GrabStats }) {
  const successPct = Math.round(data.success_rate * 100);
  const pieData =
    data.total_grabs > 0
      ? [
          { name: "Successful", value: data.successful },
          { name: "Failed", value: data.failed },
        ]
      : [];

  return (
    <Card title="Grab Performance">
      <div style={{ display: "flex", gap: 16, alignItems: "flex-start", flexWrap: "wrap" }}>
        {/* Donut chart */}
        {pieData.length > 0 && (
          <div style={{ flexShrink: 0, display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
            <PieChart width={120} height={120}>
              <Pie
                data={pieData}
                cx={55}
                cy={55}
                innerRadius={36}
                outerRadius={54}
                dataKey="value"
                strokeWidth={0}
              >
                <Cell fill={GRAB_COLORS.success} />
                <Cell fill={GRAB_COLORS.failed} />
              </Pie>
            </PieChart>
            <div style={{ display: "flex", gap: 12, fontSize: 11, color: "var(--color-text-muted)" }}>
              <span>
                <span style={{ color: GRAB_COLORS.success }}>● </span>OK
              </span>
              <span>
                <span style={{ color: GRAB_COLORS.failed }}>● </span>Failed
              </span>
            </div>
          </div>
        )}

        {/* Number blocks */}
        <div style={{ display: "flex", gap: 24, flexWrap: "wrap", flex: 1 }}>
          <StatBlock label="Total Grabs" value={data.total_grabs.toLocaleString()} />
          <StatBlock label="Successful" value={data.successful.toLocaleString()} />
          <StatBlock
            label="Failed"
            value={data.failed.toLocaleString()}
            accent={data.failed > 0 ? "var(--color-danger, #ef4444)" : undefined}
          />
          <StatBlock
            label="Success Rate"
            value={`${successPct}%`}
            accent={
              successPct >= 90
                ? "var(--color-success)"
                : successPct >= 70
                ? "var(--color-warning)"
                : "var(--color-danger, #ef4444)"
            }
          />
        </div>
      </div>

      {(data.top_indexers ?? []).length > 0 && (
        <div style={{ marginTop: 20 }}>
          <div
            style={{
              fontSize: 11,
              fontWeight: 600,
              color: "var(--color-text-muted)",
              textTransform: "uppercase",
              letterSpacing: "0.07em",
              marginBottom: 10,
            }}
          >
            Top Indexers
          </div>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr>
                {["Indexer", "Grabs", "Success Rate"].map((h) => (
                  <th
                    key={h}
                    style={{
                      textAlign: h === "Indexer" ? "left" : "right",
                      color: "var(--color-text-muted)",
                      fontWeight: 500,
                      paddingBottom: 8,
                      fontSize: 12,
                    }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.top_indexers.map((idx) => (
                <tr
                  key={idx.indexer_id}
                  style={{ borderTop: "1px solid var(--color-border-subtle)" }}
                >
                  <td
                    style={{
                      padding: "8px 0",
                      color: "var(--color-text-primary)",
                      fontWeight: 500,
                    }}
                  >
                    {idx.indexer_name}
                  </td>
                  <td
                    style={{
                      padding: "8px 0",
                      textAlign: "right",
                      color: "var(--color-text-secondary)",
                    }}
                  >
                    {idx.grab_count.toLocaleString()}
                  </td>
                  <td
                    style={{
                      padding: "8px 0",
                      textAlign: "right",
                      color:
                        idx.success_rate >= 0.9
                          ? "var(--color-success)"
                          : idx.success_rate >= 0.7
                          ? "var(--color-warning)"
                          : "var(--color-danger, #ef4444)",
                      fontWeight: 600,
                    }}
                  >
                    {Math.round(idx.success_rate * 100)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

// ── Genres card ───────────────────────────────────────────────────────────────

function GenresCard({ data }: { data: GenreBucket[] }) {
  if (data.length === 0) {
    return (
      <Card title="Top Genres">
        <EmptyChart message="No genre data yet." />
      </Card>
    );
  }

  const max = data[0]?.count ?? 1;

  return (
    <Card title="Top Genres">
      <ResponsiveContainer width="100%" height={data.length * 30 + 8}>
        <BarChart
          data={data}
          layout="vertical"
          margin={{ top: 0, right: 48, left: 0, bottom: 0 }}
        >
          <XAxis type="number" hide domain={[0, max]} />
          <YAxis
            type="category"
            dataKey="genre"
            tick={axisStyle}
            axisLine={false}
            tickLine={false}
            width={100}
          />
          <Tooltip
            contentStyle={tooltipStyle.contentStyle}
            wrapperStyle={tooltipStyle.wrapperStyle}
            cursor={tooltipStyle.cursor}
            formatter={(v: number | undefined) => [(v ?? 0).toLocaleString(), "Movies"]}
          />
          <Bar dataKey="count" fill="var(--color-accent)" radius={[0, 4, 4, 0]}>
            {data.map((_, i) => (
              <Cell
                key={i}
                fill="var(--color-accent)"
                fillOpacity={1 - i * (0.45 / Math.max(data.length - 1, 1))}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </Card>
  );
}

// ── Error card ────────────────────────────────────────────────────────────────

function ErrorCard({ title }: { title: string }) {
  return (
    <Card title={title}>
      <p style={{ color: "var(--color-danger, #ef4444)", margin: 0, fontSize: 13 }}>
        Failed to load.
      </p>
    </Card>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function StatsPage() {
  const collection = useCollectionStats();
  const quality = useQualityStats();
  const storage = useStorageStats();
  const grabs = useGrabStats();
  const decades = useDecadeStats();
  const growth = useGrowthStats();
  const genres = useGenreStats();

  const twoCol: React.CSSProperties = {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fill, minmax(380px, 1fr))",
    gap: 20,
  };

  return (
    <div style={{ padding: "32px 32px 64px", maxWidth: 1200, margin: "0 auto" }}>
      <h1
        style={{
          fontSize: 24,
          fontWeight: 700,
          color: "var(--color-text-primary)",
          marginBottom: 24,
        }}
      >
        Statistics
      </h1>

      <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
        {/* Collection summary */}
        {collection.isLoading ? (
          <CardSkeleton height={110} />
        ) : collection.error ? (
          <ErrorCard title="Collection" />
        ) : collection.data ? (
          <CollectionCard data={collection.data} />
        ) : null}

        {/* Decades | Growth */}
        <div style={twoCol}>
          {decades.isLoading ? (
            <CardSkeleton height={250} />
          ) : decades.error ? (
            <ErrorCard title="Movies by Decade" />
          ) : decades.data ? (
            <DecadesCard data={decades.data} />
          ) : null}

          {growth.isLoading ? (
            <CardSkeleton height={250} />
          ) : growth.error ? (
            <ErrorCard title="Library Growth" />
          ) : growth.data ? (
            <GrowthCard data={growth.data} />
          ) : null}
        </div>

        {/* Quality distribution — full width */}
        {quality.isLoading ? (
          <CardSkeleton height={200} />
        ) : quality.error ? (
          <ErrorCard title="Quality Distribution" />
        ) : quality.data ? (
          <QualityCard data={quality.data} />
        ) : null}

        {/* Storage | Grabs */}
        <div style={twoCol}>
          {storage.isLoading ? (
            <CardSkeleton height={220} />
          ) : storage.error ? (
            <ErrorCard title="Storage" />
          ) : storage.data ? (
            <StorageCard data={storage.data} />
          ) : null}

          {grabs.isLoading ? (
            <CardSkeleton height={220} />
          ) : grabs.error ? (
            <ErrorCard title="Grab Performance" />
          ) : grabs.data ? (
            <GrabsCard data={grabs.data} />
          ) : null}
        </div>

        {/* Genres — full width */}
        {genres.isLoading ? (
          <CardSkeleton height={300} />
        ) : genres.error ? (
          <ErrorCard title="Top Genres" />
        ) : genres.data ? (
          <GenresCard data={genres.data} />
        ) : null}
      </div>
    </div>
  );
}

