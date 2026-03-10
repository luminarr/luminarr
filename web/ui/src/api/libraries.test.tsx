import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { libraryFixture } from "@/test/fixtures";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useLibraries,
  useCreateLibrary,
  useUpdateLibrary,
  useDeleteLibrary,
  useScanLibrary,
  useLibraryStats,
} from "./libraries";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useLibraries", () => {
  it("fetches library list", async () => {
    server.use(
      http.get("/api/v1/libraries", () => HttpResponse.json([libraryFixture]))
    );

    const { result } = renderHook(() => useLibraries(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].name).toBe("Movies");
    expect(result.current.data?.[0].root_path).toBe("/movies");
  });
});

describe("useCreateLibrary", () => {
  it("creates a library via POST", async () => {
    let receivedBody: unknown = null;
    server.use(
      http.post("/api/v1/libraries", async ({ request }) => {
        receivedBody = await request.json();
        return HttpResponse.json(libraryFixture);
      })
    );

    const { result } = renderHook(() => useCreateLibrary(), { wrapper: createWrapper() });
    result.current.mutate({ name: "Movies", root_path: "/movies" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedBody).toEqual({ name: "Movies", root_path: "/movies" });
  });
});

describe("useUpdateLibrary", () => {
  it("updates a library via PUT", async () => {
    server.use(
      http.put("/api/v1/libraries/lib-1", () => HttpResponse.json(libraryFixture))
    );

    const { result } = renderHook(() => useUpdateLibrary(), { wrapper: createWrapper() });
    result.current.mutate({ id: "lib-1", name: "Movies", root_path: "/movies" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useDeleteLibrary", () => {
  it("deletes a library", async () => {
    server.use(
      http.delete("/api/v1/libraries/lib-1", () =>
        new HttpResponse(null, { status: 204 })
      )
    );

    const { result } = renderHook(() => useDeleteLibrary(), { wrapper: createWrapper() });
    result.current.mutate("lib-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useScanLibrary", () => {
  it("triggers a library scan (202)", async () => {
    server.use(
      http.post("/api/v1/libraries/lib-1/scan", () =>
        new HttpResponse(null, { status: 202 })
      )
    );

    const { result } = renderHook(() => useScanLibrary(), { wrapper: createWrapper() });
    result.current.mutate("lib-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useLibraryStats", () => {
  it("fetches library stats", async () => {
    const stats = {
      movie_count: 42,
      total_size_bytes: 500_000_000_000,
      free_space_bytes: 100_000_000_000,
      health_ok: true,
      health_message: "",
    };
    server.use(
      http.get("/api/v1/libraries/lib-1/stats", () => HttpResponse.json(stats))
    );

    const { result } = renderHook(() => useLibraryStats("lib-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.movie_count).toBe(42);
    expect(result.current.data?.health_ok).toBe(true);
  });

  it("does not fetch when id is empty", () => {
    const { result } = renderHook(() => useLibraryStats(""), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });
});
