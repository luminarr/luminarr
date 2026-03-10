import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { queueItemFixture } from "@/test/fixtures";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useQueue, useRemoveFromQueue, useBlocklistQueueItem } from "./queue";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useQueue", () => {
  it("fetches queue items", async () => {
    server.use(
      http.get("/api/v1/queue", () => HttpResponse.json([queueItemFixture]))
    );

    const { result } = renderHook(() => useQueue(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].status).toBe("downloading");
    expect(result.current.data?.[0].downloaded_bytes).toBe(4_294_967_296);
  });
});

describe("useRemoveFromQueue", () => {
  it("removes item with delete_files option", async () => {
    let receivedUrl = "";
    server.use(
      http.delete("/api/v1/queue/:id", ({ request }) => {
        receivedUrl = request.url;
        return new HttpResponse(null, { status: 204 });
      })
    );

    const { result } = renderHook(() => useRemoveFromQueue(), { wrapper: createWrapper() });
    result.current.mutate({ id: "q-1", deleteFiles: true });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedUrl).toContain("delete_files=true");
  });
});

describe("useBlocklistQueueItem", () => {
  it("blocklists a queue item", async () => {
    let calledId = "";
    server.use(
      http.post("/api/v1/queue/:id/blocklist", ({ params }) => {
        calledId = params.id as string;
        return new HttpResponse(null, { status: 204 });
      })
    );

    const { result } = renderHook(() => useBlocklistQueueItem(), { wrapper: createWrapper() });
    result.current.mutate("q-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(calledId).toBe("q-1");
  });
});
