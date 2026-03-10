import { describe, it, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { movieFixture, releaseFixture } from "@/test/fixtures";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useMovies,
  useMovie,
  useLookupMovies,
  useAddMovie,
  useUpdateMovie,
  useDeleteMovie,
  useMovieReleases,
  useGrabRelease,
  useRefreshMovie,
  useMovieFiles,
  useDeleteMovieFile,
  useMovieHistory,
  useRenameMovie,
} from "./movies";

function createWrapper() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children);
}

describe("useMovies", () => {
  it("fetches movie list", async () => {
    server.use(
      http.get("/api/v1/movies", () =>
        HttpResponse.json({ movies: [movieFixture], total: 1, page: 1, per_page: 50 })
      )
    );

    const { result } = renderHook(() => useMovies(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.movies).toHaveLength(1);
    expect(result.current.data?.movies[0].title).toBe("Fight Club");
    expect(result.current.data?.total).toBe(1);
  });

  it("passes filter query params", async () => {
    let receivedUrl = "";
    server.use(
      http.get("/api/v1/movies", ({ request }) => {
        receivedUrl = request.url;
        return HttpResponse.json({ movies: [], total: 0, page: 2, per_page: 10 });
      })
    );

    const { result } = renderHook(
      () => useMovies({ library_id: "lib-1", page: 2, per_page: 10 }),
      { wrapper: createWrapper() }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedUrl).toContain("library_id=lib-1");
    expect(receivedUrl).toContain("page=2");
    expect(receivedUrl).toContain("per_page=10");
  });
});

describe("useMovie", () => {
  it("fetches a single movie", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1", () => HttpResponse.json(movieFixture))
    );

    const { result } = renderHook(() => useMovie("movie-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.title).toBe("Fight Club");
    expect(result.current.data?.tmdb_id).toBe(550);
  });

  it("does not fetch when id is empty", async () => {
    const { result } = renderHook(() => useMovie(""), { wrapper: createWrapper() });
    // Should never start fetching
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useLookupMovies", () => {
  it("looks up movies by query", async () => {
    const tmdbResults = [
      { tmdb_id: 550, title: "Fight Club", original_title: "Fight Club", overview: "", release_date: "1999-10-15", year: 1999, popularity: 60 },
    ];
    server.use(
      http.post("/api/v1/movies/lookup", () => HttpResponse.json(tmdbResults))
    );

    const { result } = renderHook(() => useLookupMovies(), { wrapper: createWrapper() });
    result.current.mutate({ query: "fight club" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].tmdb_id).toBe(550);
  });
});

describe("useAddMovie", () => {
  it("adds a movie via POST", async () => {
    let receivedBody: unknown = null;
    server.use(
      http.post("/api/v1/movies", async ({ request }) => {
        receivedBody = await request.json();
        return HttpResponse.json(movieFixture);
      })
    );

    const { result } = renderHook(() => useAddMovie(), { wrapper: createWrapper() });
    result.current.mutate({
      tmdb_id: 550,
      library_id: "lib-1",
      quality_profile_id: "qp-1",
      monitored: true,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedBody).toEqual({
      tmdb_id: 550,
      library_id: "lib-1",
      quality_profile_id: "qp-1",
      monitored: true,
    });
  });
});

describe("useUpdateMovie", () => {
  it("updates a movie via PUT", async () => {
    let receivedBody: unknown = null;
    server.use(
      http.put("/api/v1/movies/movie-1", async ({ request }) => {
        receivedBody = await request.json();
        return HttpResponse.json(movieFixture);
      })
    );

    const { result } = renderHook(() => useUpdateMovie(), { wrapper: createWrapper() });
    result.current.mutate({
      id: "movie-1",
      title: "Fight Club",
      monitored: false,
      library_id: "lib-1",
      quality_profile_id: "qp-1",
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedBody).toEqual({
      title: "Fight Club",
      monitored: false,
      library_id: "lib-1",
      quality_profile_id: "qp-1",
    });
  });
});

describe("useDeleteMovie", () => {
  it("deletes a movie", async () => {
    let deletedId = "";
    server.use(
      http.delete("/api/v1/movies/:id", ({ params }) => {
        deletedId = params.id as string;
        return new HttpResponse(null, { status: 204 });
      })
    );

    const { result } = renderHook(() => useDeleteMovie(), { wrapper: createWrapper() });
    result.current.mutate("movie-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(deletedId).toBe("movie-1");
  });
});

describe("useMovieReleases", () => {
  it("fetches releases for a movie", async () => {
    server.use(
      http.get("/api/v1/movies/movie-1/releases", () =>
        HttpResponse.json([releaseFixture])
      )
    );

    const { result } = renderHook(() => useMovieReleases("movie-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].title).toBe(releaseFixture.title);
    expect(result.current.data?.[0].seeds).toBe(42);
  });

  it("does not fetch when movieId is empty", () => {
    const { result } = renderHook(() => useMovieReleases(""), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("useGrabRelease", () => {
  it("grabs a release via POST", async () => {
    let receivedBody: unknown = null;
    server.use(
      http.post("/api/v1/movies/movie-1/releases/grab", async ({ request }) => {
        receivedBody = await request.json();
        return HttpResponse.json({
          id: "grab-1",
          movie_id: "movie-1",
          release_guid: "release-1",
          release_title: releaseFixture.title,
          protocol: "torrent",
          size: releaseFixture.size,
          download_status: "grabbed",
          grabbed_at: "2025-01-01T12:00:00Z",
        });
      })
    );

    const { result } = renderHook(() => useGrabRelease(), { wrapper: createWrapper() });
    result.current.mutate({
      movieId: "movie-1",
      guid: "release-1",
      title: releaseFixture.title,
      protocol: "torrent",
      download_url: releaseFixture.download_url,
      size: releaseFixture.size,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedBody).toHaveProperty("guid", "release-1");
  });
});

describe("useRefreshMovie", () => {
  it("refreshes a movie (202)", async () => {
    server.use(
      http.post("/api/v1/movies/movie-1/refresh", () =>
        new HttpResponse(null, { status: 202 })
      )
    );

    const { result } = renderHook(() => useRefreshMovie(), { wrapper: createWrapper() });
    result.current.mutate("movie-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useMovieFiles", () => {
  it("fetches movie files", async () => {
    const files = [{
      id: "file-1",
      movie_id: "movie-1",
      path: "/movies/Fight Club (1999)/Fight.Club.1999.1080p.BluRay.mkv",
      size_bytes: 8_589_934_592,
      quality: { resolution: "1080p", source: "bluray", codec: "x264", hdr: "", name: "Bluray-1080p" },
      imported_at: "2025-01-01T00:00:00Z",
    }];
    server.use(
      http.get("/api/v1/movies/movie-1/files", () => HttpResponse.json(files))
    );

    const { result } = renderHook(() => useMovieFiles("movie-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].path).toContain("Fight.Club");
  });
});

describe("useDeleteMovieFile", () => {
  it("deletes a movie file with disk deletion", async () => {
    let receivedUrl = "";
    server.use(
      http.delete("/api/v1/movies/movie-1/files/:fileId", ({ request }) => {
        receivedUrl = request.url;
        return new HttpResponse(null, { status: 204 });
      })
    );

    const { result } = renderHook(() => useDeleteMovieFile("movie-1"), { wrapper: createWrapper() });
    result.current.mutate({ fileId: "file-1", deleteFromDisk: true });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(receivedUrl).toContain("delete_from_disk=true");
  });
});

describe("useMovieHistory", () => {
  it("fetches movie grab history", async () => {
    const history = [{
      id: "grab-1",
      movie_id: "movie-1",
      release_guid: "release-1",
      release_title: "Fight.Club.1999.1080p",
      protocol: "torrent",
      size: 8_589_934_592,
      download_status: "imported",
      grabbed_at: "2025-01-01T12:00:00Z",
    }];
    server.use(
      http.get("/api/v1/movies/movie-1/history", () => HttpResponse.json(history))
    );

    const { result } = renderHook(() => useMovieHistory("movie-1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].download_status).toBe("imported");
  });
});

describe("useRenameMovie", () => {
  it("performs dry run rename", async () => {
    const renameResult = {
      dry_run: true,
      renamed: [
        { file_id: "f1", old_path: "/movies/old.mkv", new_path: "/movies/new.mkv" },
      ],
    };
    server.use(
      http.post("/api/v1/movies/movie-1/rename", ({ request }) => {
        expect(request.url).toContain("dry_run=true");
        return HttpResponse.json(renameResult);
      })
    );

    const { result } = renderHook(() => useRenameMovie(), { wrapper: createWrapper() });
    result.current.mutate({ id: "movie-1", dryRun: true });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.dry_run).toBe(true);
    expect(result.current.data?.renamed).toHaveLength(1);
  });
});
