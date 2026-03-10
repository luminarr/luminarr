import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useCollectionStats,
  useQualityStats,
  useStorageStats,
  useGrabStats,
} from "./stats";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useCollectionStats", () => {
  it("fetches collection stats", async () => {
    const stats = {
      total_movies: 100,
      monitored: 80,
      with_file: 75,
      missing: 25,
      needs_upgrade: 5,
      recently_added: 3,
    };
    server.use(
      http.get("/api/v1/stats/collection", () => HttpResponse.json(stats))
    );

    const { result } = renderHook(() => useCollectionStats(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.total_movies).toBe(100);
    expect(result.current.data?.with_file).toBe(75);
  });
});

describe("useQualityStats", () => {
  it("fetches quality distribution", async () => {
    const buckets = [
      { resolution: "1080p", source: "bluray", codec: "x264", hdr: "", count: 40 },
      { resolution: "2160p", source: "webdl", codec: "x265", hdr: "HDR10", count: 20 },
    ];
    server.use(
      http.get("/api/v1/stats/quality", () => HttpResponse.json(buckets))
    );

    const { result } = renderHook(() => useQualityStats(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(2);
  });
});

describe("useStorageStats", () => {
  it("fetches storage stats with trend", async () => {
    const storage = {
      total_bytes: 500_000_000_000,
      file_count: 75,
      trend: [
        { captured_at: "2025-01-01T00:00:00Z", total_bytes: 400_000_000_000, file_count: 60 },
      ],
    };
    server.use(
      http.get("/api/v1/stats/storage", () => HttpResponse.json(storage))
    );

    const { result } = renderHook(() => useStorageStats(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.total_bytes).toBe(500_000_000_000);
    expect(result.current.data?.trend).toHaveLength(1);
  });
});

describe("useGrabStats", () => {
  it("fetches grab stats", async () => {
    const grabs = {
      total_grabs: 50,
      successful: 45,
      failed: 5,
      success_rate: 0.9,
      top_indexers: [
        { indexer_id: "idx-1", indexer_name: "Prowlarr", grab_count: 30, success_rate: 0.95 },
        { indexer_id: "idx-2", indexer_name: "NZBgeek", grab_count: 20, success_rate: 0.85 },
      ],
    };
    server.use(
      http.get("/api/v1/stats/grabs", () => HttpResponse.json(grabs))
    );

    const { result } = renderHook(() => useGrabStats(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.total_grabs).toBe(50);
    expect(result.current.data?.top_indexers).toHaveLength(2);
  });
});
