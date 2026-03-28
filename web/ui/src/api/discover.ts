import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";

export interface DiscoverResult {
  tmdb_id: number;
  title: string;
  year: number;
  overview: string;
  poster_path: string;
  rating: number;
  in_library: boolean;
  excluded: boolean;
  library_movie_id?: string;
}

export interface DiscoverListResponse {
  results: DiscoverResult[];
  page: number;
  total_pages: number;
}

export interface TMDBGenre {
  id: number;
  name: string;
}

type DiscoverCategory = "trending" | "popular" | "top-rated" | "upcoming";

export function useDiscover(category: DiscoverCategory, page: number) {
  return useQuery({
    queryKey: ["discover", category, page],
    queryFn: () =>
      apiFetch<DiscoverListResponse>(
        `/discover/${category}?page=${page}`
      ),
    staleTime: 5 * 60_000,
  });
}

export function useDiscoverByGenre(genreId: number, page: number) {
  return useQuery({
    queryKey: ["discover", "genre", genreId, page],
    queryFn: () =>
      apiFetch<DiscoverListResponse>(
        `/discover/genre/${genreId}?page=${page}`
      ),
    enabled: genreId > 0,
    staleTime: 5 * 60_000,
  });
}

export function useGenreList() {
  return useQuery({
    queryKey: ["discover", "genres"],
    queryFn: () => apiFetch<TMDBGenre[]>("/discover/genres"),
    staleTime: Infinity,
  });
}
