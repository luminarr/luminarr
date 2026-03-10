import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useHistory } from "./history";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useHistory", () => {
  it("fetches history with default params", async () => {
    const history = [
      {
        id: "grab-1",
        movie_id: "movie-1",
        release_guid: "r1",
        release_title: "Movie.2024.1080p",
        protocol: "torrent",
        size: 1_000_000_000,
        download_status: "imported",
        grabbed_at: "2025-01-01T12:00:00Z",
      },
    ];
    server.use(
      http.get("/api/v1/history", () => HttpResponse.json(history))
    );

    const { result } = renderHook(() => useHistory({}), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].download_status).toBe("imported");
  });

  it("passes filter query params", async () => {
    let receivedUrl = "";
    server.use(
      http.get("/api/v1/history", ({ request }) => {
        receivedUrl = request.url;
        return HttpResponse.json([]);
      })
    );

    const { result } = renderHook(
      () => useHistory({ limit: 10, download_status: "failed", protocol: "torrent" }),
      { wrapper: createWrapper() }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(receivedUrl).toContain("limit=10");
    expect(receivedUrl).toContain("download_status=failed");
    expect(receivedUrl).toContain("protocol=torrent");
  });
});
