import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";
import type { Movie, MovieListResponse, TMDBResult, Release, GrabHistory, MovieFile, RenameMovieResult, AutoSearchResult, BulkSearchAccepted, ExplainResult } from "@/types";

export function useEditions() {
  return useQuery({
    queryKey: ["editions"],
    queryFn: () => apiFetch<string[]>("/editions"),
    staleTime: Infinity,
  });
}

interface MovieFilters {
  library_id?: string;
  page?: number;
  per_page?: number;
}

export function useMovies(filters?: MovieFilters) {
  const params = new URLSearchParams();
  if (filters?.library_id) params.set("library_id", filters.library_id);
  if (filters?.page) params.set("page", String(filters.page));
  if (filters?.per_page) params.set("per_page", String(filters.per_page));
  const qs = params.toString();

  return useQuery({
    queryKey: ["movies", filters],
    queryFn: () => apiFetch<MovieListResponse>(`/movies${qs ? `?${qs}` : ""}`),
  });
}

export function useMovie(id: string) {
  return useQuery({
    queryKey: ["movies", id],
    queryFn: () => apiFetch<Movie>(`/movies/${id}`),
    enabled: !!id,
  });
}

export function useLookupMovies() {
  return useMutation({
    mutationFn: (body: { query?: string; tmdb_id?: number; year?: number }) =>
      apiFetch<TMDBResult[]>("/movies/lookup", { method: "POST", body: JSON.stringify(body) }),
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useAddMovie() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      tmdb_id: number;
      library_id: string;
      quality_profile_id: string;
      monitored?: boolean;
      minimum_availability?: string;
      preferred_edition?: string;
    }) => apiFetch<Movie>("/movies", { method: "POST", body: JSON.stringify(body) }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["movies"] });
      toast.success("Movie added to library");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export interface MovieUpdateRequest {
  title: string;
  monitored: boolean;
  library_id: string;
  quality_profile_id: string;
  minimum_availability?: string;
  preferred_edition?: string;
}

export function useUpdateMovie() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: MovieUpdateRequest & { id: string }) =>
      apiFetch<Movie>(`/movies/${id}`, { method: "PUT", body: JSON.stringify(body) }),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ["movies"] });
      qc.invalidateQueries({ queryKey: ["movies", id] });
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useDeleteMovie() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/movies/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["movies"] });
      toast.success("Movie removed");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useMovieReleases(movieId: string) {
  return useQuery({
    queryKey: ["movies", movieId, "releases"],
    queryFn: () => apiFetch<Release[]>(`/movies/${movieId}/releases`),
    enabled: !!movieId,
  });
}

export interface GrabReleaseRequest {
  guid: string;
  title: string;
  indexer_id?: string;
  protocol: string;
  download_url: string;
  size: number;
}

export function useGrabRelease() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ movieId, guid, ...body }: GrabReleaseRequest & { movieId: string }) =>
      apiFetch<GrabHistory>(
        `/movies/${movieId}/releases/grab`,
        { method: "POST", body: JSON.stringify({ guid, ...body }) }
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["queue"] }),
  });
}

export function useRefreshMovie() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/movies/${id}/refresh`, { method: "POST" }),
    onSuccess: (_, id) => qc.invalidateQueries({ queryKey: ["movies", id] }),
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useMovieFiles(movieId: string) {
  return useQuery({
    queryKey: ["movies", movieId, "files"],
    queryFn: () => apiFetch<MovieFile[]>(`/movies/${movieId}/files`),
    enabled: !!movieId,
  });
}

export function useDeleteMovieFile(movieId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ fileId, deleteFromDisk }: { fileId: string; deleteFromDisk: boolean }) =>
      apiFetch<void>(
        `/movies/${movieId}/files/${fileId}?delete_from_disk=${deleteFromDisk}`,
        { method: "DELETE" }
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["movies", movieId, "files"] });
      qc.invalidateQueries({ queryKey: ["movies", movieId] });
    },
  });
}

export function useMatchMovie() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, tmdb_id }: { id: string; tmdb_id: number }) =>
      apiFetch<Movie>(`/movies/${id}/match`, { method: "POST", body: JSON.stringify({ tmdb_id }) }),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ["movies"] });
      qc.invalidateQueries({ queryKey: ["movies", id] });
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export interface MovieSuggestions {
  parsed_title: string;
  parsed_year: number;
  results: TMDBResult[];
}

export function useMovieSuggestions(movieId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["movies", movieId, "suggestions"],
    queryFn: () => apiFetch<MovieSuggestions>(`/movies/${movieId}/suggestions`),
    enabled: enabled && !!movieId,
  });
}

export function useRenameMovie() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, dryRun }: { id: string; dryRun: boolean }) =>
      apiFetch<RenameMovieResult>(
        `/movies/${id}/rename?dry_run=${dryRun}`,
        { method: "POST" }
      ),
    onSuccess: (data, { id }) => {
      if (!data.dry_run) {
        qc.invalidateQueries({ queryKey: ["movies", id, "files"] });
        qc.invalidateQueries({ queryKey: ["movies", id] });
      }
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useMovieHistory(movieId: string) {
  return useQuery({
    queryKey: ["movies", movieId, "history"],
    queryFn: () => apiFetch<GrabHistory[]>(`/movies/${movieId}/history`),
    enabled: !!movieId,
  });
}

export function useAutoSearch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (movieId: string) =>
      apiFetch<AutoSearchResult>(`/movies/${movieId}/search`, { method: "POST" }),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["queue"] });
      if (data.movie_id) {
        qc.invalidateQueries({ queryKey: ["movies", data.movie_id] });
        qc.invalidateQueries({ queryKey: ["movies", data.movie_id, "history"] });
      }
    },
  });
}

export function useBulkAutoSearch() {
  return useMutation({
    mutationFn: (movieIds: string[]) =>
      apiFetch<BulkSearchAccepted>("/movies/search", {
        method: "POST",
        body: JSON.stringify({ movie_ids: movieIds }),
      }),
  });
}

export function useExplainReleases(movieId: string, enabled: boolean) {
  return useQuery({
    queryKey: ["movies", movieId, "releases", "explain"],
    queryFn: () => apiFetch<ExplainResult>(`/movies/${movieId}/releases/explain`),
    enabled: enabled && !!movieId,
  });
}

export interface MovieCredits {
  cast: {
    id: number;
    name: string;
    character: string;
    profile_path: string;
    order: number;
  }[];
  crew: {
    id: number;
    name: string;
    job: string;
    department: string;
    profile_path: string;
  }[];
  recommendations: {
    tmdb_id: number;
    title: string;
    year: number;
    poster_path: string;
    in_library: boolean;
    movie_id?: string;
  }[];
}

export function useMovieCredits(movieId: string) {
  return useQuery({
    queryKey: ["movies", movieId, "credits"],
    queryFn: () => apiFetch<MovieCredits>(`/movies/${movieId}/credits`),
    enabled: !!movieId,
    staleTime: 5 * 60_000, // cache 5 min — TMDB data doesn't change often
  });
}
