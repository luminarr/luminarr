import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { indexerFixture } from "@/test/fixtures";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useIndexers,
  useCreateIndexer,
  useUpdateIndexer,
  useDeleteIndexer,
  useTestIndexer,
} from "./indexers";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useIndexers", () => {
  it("fetches indexer list", async () => {
    server.use(
      http.get("/api/v1/indexers", () => HttpResponse.json([indexerFixture]))
    );

    const { result } = renderHook(() => useIndexers(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].name).toBe("Test Indexer");
    expect(result.current.data?.[0].kind).toBe("torznab");
  });
});

describe("useCreateIndexer", () => {
  it("creates an indexer via POST", async () => {
    server.use(
      http.post("/api/v1/indexers", () => HttpResponse.json(indexerFixture))
    );

    const { result } = renderHook(() => useCreateIndexer(), { wrapper: createWrapper() });
    result.current.mutate({
      name: "Test Indexer",
      kind: "torznab",
      settings: { url: "http://localhost:9696", api_key: "abc123" },
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useUpdateIndexer", () => {
  it("updates an indexer via PUT", async () => {
    server.use(
      http.put("/api/v1/indexers/idx-1", () => HttpResponse.json(indexerFixture))
    );

    const { result } = renderHook(() => useUpdateIndexer(), { wrapper: createWrapper() });
    result.current.mutate({
      id: "idx-1",
      name: "Test Indexer",
      kind: "torznab",
      settings: { url: "http://localhost:9696", api_key: "abc123" },
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useDeleteIndexer", () => {
  it("deletes an indexer", async () => {
    server.use(
      http.delete("/api/v1/indexers/idx-1", () =>
        new HttpResponse(null, { status: 204 })
      )
    );

    const { result } = renderHook(() => useDeleteIndexer(), { wrapper: createWrapper() });
    result.current.mutate("idx-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useTestIndexer", () => {
  it("tests an indexer and returns result", async () => {
    server.use(
      http.post("/api/v1/indexers/idx-1/test", () =>
        HttpResponse.json({ ok: true })
      )
    );

    const { result } = renderHook(() => useTestIndexer(), { wrapper: createWrapper() });
    result.current.mutate("idx-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.ok).toBe(true);
  });

  it("returns failure message", async () => {
    server.use(
      http.post("/api/v1/indexers/idx-1/test", () =>
        HttpResponse.json({ ok: false, message: "Connection refused" })
      )
    );

    const { result } = renderHook(() => useTestIndexer(), { wrapper: createWrapper() });
    result.current.mutate("idx-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.ok).toBe(false);
    expect(result.current.data?.message).toBe("Connection refused");
  });
});
