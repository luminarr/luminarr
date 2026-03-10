import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useBlocklist, useDeleteBlocklistEntry, useClearBlocklist } from "./blocklist";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useBlocklist", () => {
  it("fetches paginated blocklist", async () => {
    const page = {
      items: [
        {
          id: "bl-1",
          movie_id: "movie-1",
          movie_title: "Fight Club",
          release_guid: "r1",
          release_title: "Fight.Club.CAM",
          protocol: "torrent",
          size: 700_000_000,
          added_at: "2025-01-01T00:00:00Z",
        },
      ],
      total: 1,
      page: 1,
      per_page: 50,
    };
    server.use(
      http.get("/api/v1/blocklist", () => HttpResponse.json(page))
    );

    const { result } = renderHook(() => useBlocklist(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.items).toHaveLength(1);
    expect(result.current.data?.items[0].release_title).toBe("Fight.Club.CAM");
  });
});

describe("useDeleteBlocklistEntry", () => {
  it("deletes a single entry", async () => {
    server.use(
      http.delete("/api/v1/blocklist/bl-1", () =>
        new HttpResponse(null, { status: 204 })
      )
    );

    const { result } = renderHook(() => useDeleteBlocklistEntry(), { wrapper: createWrapper() });
    result.current.mutate("bl-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useClearBlocklist", () => {
  it("clears the entire blocklist", async () => {
    server.use(
      http.delete("/api/v1/blocklist", () =>
        new HttpResponse(null, { status: 204 })
      )
    );

    const { result } = renderHook(() => useClearBlocklist(), { wrapper: createWrapper() });
    result.current.mutate();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
