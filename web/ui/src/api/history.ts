import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "./client";
import type { GrabHistory } from "@/types";

export function useHistory(limit = 100) {
  return useQuery({
    queryKey: ["history", limit],
    queryFn: () => apiFetch<GrabHistory[]>(`/history?limit=${limit}`),
    staleTime: 30_000,
  });
}
