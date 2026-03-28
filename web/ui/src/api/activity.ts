import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";

export interface Activity {
  id: string;
  type: string;
  category: "grab" | "import" | "task" | "health" | "movie";
  movie_id?: string;
  title: string;
  detail?: Record<string, unknown>;
  created_at: string;
}

export interface ActivityListResult {
  activities: Activity[];
  total: number;
}

export interface ActivityFilters {
  category?: string;
  since?: string;
  limit?: number;
}

export function useActivity(filters?: ActivityFilters) {
  const params = new URLSearchParams();
  if (filters?.category) params.set("category", filters.category);
  if (filters?.since) params.set("since", filters.since);
  if (filters?.limit) params.set("limit", String(filters.limit));
  const qs = params.toString();

  return useQuery({
    queryKey: ["activity", filters],
    queryFn: () =>
      apiFetch<ActivityListResult>(`/activity${qs ? `?${qs}` : ""}`),
    refetchInterval: 15_000,
  });
}
