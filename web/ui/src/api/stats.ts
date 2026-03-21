import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";

export interface CollectionStats {
  total_movies: number;
  monitored: number;
  with_file: number;
  missing: number;
  needs_upgrade: number;
  edition_mismatches: number;
  recently_added: number;
}

export interface QualityBucket {
  resolution: string;
  source: string;
  codec: string;
  hdr: string;
  count: number;
}

export interface StoragePoint {
  captured_at: string;
  total_bytes: number;
  file_count: number;
}

export interface StorageStats {
  total_bytes: number;
  file_count: number;
  trend: StoragePoint[];
}

export interface IndexerStat {
  indexer_id: string;
  indexer_name: string;
  grab_count: number;
  success_rate: number;
}

export interface GrabStats {
  total_grabs: number;
  successful: number;
  failed: number;
  success_rate: number;
  top_indexers: IndexerStat[];
}

export function useCollectionStats() {
  return useQuery({
    queryKey: ["stats", "collection"],
    queryFn: () => apiFetch<CollectionStats>("/stats/collection"),
  });
}

export function useQualityStats() {
  return useQuery({
    queryKey: ["stats", "quality"],
    queryFn: () => apiFetch<QualityBucket[]>("/stats/quality"),
  });
}

export function useStorageStats() {
  return useQuery({
    queryKey: ["stats", "storage"],
    queryFn: () => apiFetch<StorageStats>("/stats/storage"),
  });
}

export function useGrabStats() {
  return useQuery({
    queryKey: ["stats", "grabs"],
    queryFn: () => apiFetch<GrabStats>("/stats/grabs"),
  });
}

export interface DecadeBucket {
  decade: string;
  count: number;
}

export interface GrowthPoint {
  month: string;
  added: number;
  cumulative: number;
}

export interface GenreBucket {
  genre: string;
  count: number;
}

export function useDecadeStats() {
  return useQuery({
    queryKey: ["stats", "decades"],
    queryFn: () => apiFetch<DecadeBucket[]>("/stats/decades"),
  });
}

export function useGrowthStats() {
  return useQuery({
    queryKey: ["stats", "growth"],
    queryFn: () => apiFetch<GrowthPoint[]>("/stats/growth"),
  });
}

export function useGenreStats() {
  return useQuery({
    queryKey: ["stats", "genres"],
    queryFn: () => apiFetch<GenreBucket[]>("/stats/genres"),
  });
}

export interface QualityTier {
  resolution: string;
  source: string;
  count: number;
}

export function useQualityTiers() {
  return useQuery({
    queryKey: ["stats", "quality", "tiers"],
    queryFn: () => apiFetch<QualityTier[]>("/stats/quality/tiers"),
  });
}

export function useQualityMovies(resolution: string, source: string, enabled: boolean) {
  const params = new URLSearchParams();
  if (resolution) params.set("resolution", resolution);
  if (source) params.set("source", source);
  return useQuery({
    queryKey: ["stats", "quality", "movies", resolution, source],
    queryFn: () => apiFetch<string[]>(`/stats/quality/movies?${params.toString()}`),
    enabled: enabled && (!!resolution || !!source),
  });
}
