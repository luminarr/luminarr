import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";
import type { Movie, MovieListResponse, TMDBResult, Release, GrabHistory } from "@/types";

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
