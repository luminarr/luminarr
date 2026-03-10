import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { systemStatusFixture, healthyReport, degradedReport } from "@/test/fixtures";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useSystemStatus,
  useSystemHealth,
  useTasks,
  useRunTask,
  usePlugins,
  useSystemConfig,
  useSaveConfig,
} from "./system";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useSystemStatus", () => {
  it("fetches system status", async () => {
    server.use(
      http.get("/api/v1/system/status", () => HttpResponse.json(systemStatusFixture))
    );

    const { result } = renderHook(() => useSystemStatus(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.app_name).toBe("Luminarr");
    expect(result.current.data?.version).toBe("0.0.0-test");
    expect(result.current.data?.go_version).toBe("go1.23.0");
  });
});

describe("useSystemHealth", () => {
  it("fetches healthy report", async () => {
    server.use(
      http.get("/api/v1/system/health", () => HttpResponse.json(healthyReport))
    );

    const { result } = renderHook(() => useSystemHealth(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.status).toBe("healthy");
    expect(result.current.data?.checks).toHaveLength(0);
  });

  it("fetches degraded report with checks", async () => {
    server.use(
      http.get("/api/v1/system/health", () => HttpResponse.json(degradedReport))
    );

    const { result } = renderHook(() => useSystemHealth(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.status).toBe("degraded");
    expect(result.current.data?.checks).toHaveLength(1);
    expect(result.current.data?.checks[0].name).toBe("indexer");
  });
});

describe("useTasks", () => {
  it("fetches task list", async () => {
    const tasks = [
      { name: "refresh-metadata", interval: "24h" },
      { name: "check-downloads", interval: "5m" },
    ];
    server.use(
      http.get("/api/v1/tasks", () => HttpResponse.json(tasks))
    );

    const { result } = renderHook(() => useTasks(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(2);
    expect(result.current.data?.[0].name).toBe("refresh-metadata");
  });
});

describe("useRunTask", () => {
  it("posts to run a task (202)", async () => {
    let calledWith = "";
    server.use(
      http.post("/api/v1/tasks/:name/run", ({ params }) => {
        calledWith = params.name as string;
        return new HttpResponse(null, { status: 202 });
      })
    );

    const { result } = renderHook(() => useRunTask(), { wrapper: createWrapper() });
    result.current.mutate("refresh-metadata");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(calledWith).toBe("refresh-metadata");
  });
});

describe("usePlugins", () => {
  it("fetches plugin list", async () => {
    const plugins = {
      indexers: ["torznab", "newznab"],
      downloaders: ["qbittorrent", "deluge"],
      notifications: ["discord", "webhook", "email"],
    };
    server.use(
      http.get("/api/v1/system/plugins", () => HttpResponse.json(plugins))
    );

    const { result } = renderHook(() => usePlugins(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.indexers).toContain("torznab");
    expect(result.current.data?.downloaders).toContain("qbittorrent");
    expect(result.current.data?.notifications).toHaveLength(3);
  });
});

describe("useSystemConfig", () => {
  it("fetches system config", async () => {
    server.use(
      http.get("/api/v1/system/config", () =>
        HttpResponse.json({
          tmdb_key_configured: true,
          tmdb_key_source: "default",
          api_key: "abc123",
        })
      )
    );

    const { result } = renderHook(() => useSystemConfig(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.tmdb_key_configured).toBe(true);
    expect(result.current.data?.api_key).toBe("abc123");
  });
});

describe("useSaveConfig", () => {
  it("saves config via PUT", async () => {
    let receivedBody: unknown = null;
    server.use(
      http.put("/api/v1/system/config", async ({ request }) => {
        receivedBody = await request.json();
        return HttpResponse.json({ saved: true, config_file: "/config/config.yaml" });
      })
    );

    const { result } = renderHook(() => useSaveConfig(), { wrapper: createWrapper() });
    result.current.mutate({ tmdb_api_key: "new-key" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedBody).toEqual({ tmdb_api_key: "new-key" });
  });
});
