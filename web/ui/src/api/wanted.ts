import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";
import type { Movie, MovieListResponse, UpgradeRecommendations } from "@/types";

// Re-use the same paginated response shape as the movies list.
export function useWantedMissing(page: number, perPage: number) {
  return useQuery({
    queryKey: ["wanted", "missing", page, perPage],
    queryFn: () =>
      apiFetch<MovieListResponse>(`/wanted/missing?page=${page}&per_page=${perPage}`),
  });
}

export function useWantedCutoff() {
  return useQuery({
    queryKey: ["wanted", "cutoff"],
    queryFn: () => apiFetch<{ movies: Movie[]; total: number; page: number; per_page: number }>("/wanted/cutoff"),
  });
}

export function useUpgradeRecommendations() {
  return useQuery({
    queryKey: ["wanted", "upgrades"],
    queryFn: () => apiFetch<UpgradeRecommendations>("/wanted/upgrades"),
  });
}
