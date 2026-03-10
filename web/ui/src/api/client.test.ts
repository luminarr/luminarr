import { describe, it, expect } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { apiFetch, APIError } from "./client";

describe("apiFetch", () => {
  it("returns JSON for successful responses", async () => {
    server.use(
      http.get("/api/v1/test", () =>
        HttpResponse.json({ message: "ok" })
      )
    );

    const result = await apiFetch<{ message: string }>("/test");
    expect(result).toEqual({ message: "ok" });
  });

  it("sends Content-Type application/json", async () => {
    let receivedContentType = "";
    server.use(
      http.get("/api/v1/test", ({ request }) => {
        receivedContentType = request.headers.get("content-type") ?? "";
        return HttpResponse.json({});
      })
    );

    await apiFetch("/test");
    expect(receivedContentType).toBe("application/json");
  });

  it("throws APIError on non-ok response", async () => {
    server.use(
      http.get("/api/v1/test", () =>
        HttpResponse.json(
          { title: "Not Found", detail: "Movie not found" },
          { status: 404 }
        )
      )
    );

    await expect(apiFetch("/test")).rejects.toThrow(APIError);
    try {
      await apiFetch("/test");
    } catch (e) {
      const err = e as APIError;
      expect(err.status).toBe(404);
      expect(err.detail).toBe("Movie not found");
    }
  });

  it("extracts huma errors[0].message", async () => {
    server.use(
      http.post("/api/v1/test", () =>
        HttpResponse.json(
          {
            title: "Unprocessable Entity",
            errors: [{ message: "name is required" }],
          },
          { status: 422 }
        )
      )
    );

    try {
      await apiFetch("/test", { method: "POST" });
    } catch (e) {
      const err = e as APIError;
      expect(err.status).toBe(422);
      expect(err.message).toBe("name is required");
      expect(err.detail).toBe("name is required");
    }
  });

  it("returns undefined for 202 Accepted", async () => {
    server.use(
      http.post("/api/v1/test", () => new HttpResponse(null, { status: 202 }))
    );

    const result = await apiFetch("/test", { method: "POST" });
    expect(result).toBeUndefined();
  });

  it("returns undefined for 204 No Content", async () => {
    server.use(
      http.delete("/api/v1/test", () => new HttpResponse(null, { status: 204 }))
    );

    const result = await apiFetch("/test", { method: "DELETE" });
    expect(result).toBeUndefined();
  });

  it("falls back to statusText when body is not JSON", async () => {
    server.use(
      http.get("/api/v1/test", () =>
        new HttpResponse("Internal Server Error", { status: 500 })
      )
    );

    try {
      await apiFetch("/test");
    } catch (e) {
      const err = e as APIError;
      expect(err.status).toBe(500);
      expect(err.name).toBe("APIError");
    }
  });

  it("passes custom headers and method", async () => {
    let receivedMethod = "";
    server.use(
      http.put("/api/v1/test", ({ request }) => {
        receivedMethod = request.method;
        return HttpResponse.json({ updated: true });
      })
    );

    const result = await apiFetch<{ updated: boolean }>("/test", {
      method: "PUT",
      body: JSON.stringify({ name: "new" }),
    });
    expect(result).toEqual({ updated: true });
    expect(receivedMethod).toBe("PUT");
  });
});
