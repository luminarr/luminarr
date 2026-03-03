import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";
import type { QueueItem } from "@/types";

export function useQueue() {
  return useQuery({
    queryKey: ["queue"],
    queryFn: () => apiFetch<QueueItem[]>("/queue"),
    // WebSocket events keep the queue fresh in real time; 60s polling is a
    // fallback for the case where the WebSocket connection has dropped.
    refetchInterval: 60_000,
  });
}

export function useRemoveFromQueue() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, deleteFiles }: { id: string; deleteFiles?: boolean }) =>
      apiFetch<void>(`/queue/${id}?delete_files=${deleteFiles ?? false}`, {
        method: "DELETE",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["queue"] });
      toast.success("Removed from queue");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}
