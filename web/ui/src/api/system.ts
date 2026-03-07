import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";
import type { SystemStatus, HealthReport, Task, LogEntry, PluginList, UpdateCheck } from "@/types";

export function useSystemStatus() {
  return useQuery({
    queryKey: ["system", "status"],
    queryFn: () => apiFetch<SystemStatus>("/system/status"),
    refetchInterval: 30_000,
  });
}

export function useSystemHealth() {
  return useQuery({
    queryKey: ["system", "health"],
    queryFn: () => apiFetch<HealthReport>("/system/health"),
    refetchInterval: 60_000,
  });
}

export function useTasks() {
  return useQuery({
    queryKey: ["tasks"],
    queryFn: () => apiFetch<Task[]>("/tasks"),
    refetchInterval: 30_000,
  });
}

export function useRunTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiFetch<void>(`/tasks/${name}/run`, { method: "POST" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tasks"] }),
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useSystemLogs(level?: string, lines?: number) {
  const params = new URLSearchParams();
  if (level) params.set("level", level);
  if (lines) params.set("lines", String(lines));
  const qs = params.toString();
  return useQuery({
    queryKey: ["system", "logs", level, lines],
    queryFn: () => apiFetch<LogEntry[]>(`/system/logs${qs ? `?${qs}` : ""}`),
    refetchInterval: 10_000,
  });
}

export function usePlugins() {
  return useQuery({
    queryKey: ["system", "plugins"],
    queryFn: () => apiFetch<PluginList>("/system/plugins"),
    staleTime: Infinity,
  });
}

export function useCheckForUpdates() {
  return useMutation({
    mutationFn: () => apiFetch<UpdateCheck>("/system/updates"),
  });
}

export interface SystemConfig {
  tmdb_key_configured: boolean;
  tmdb_key_source: "default" | "custom" | "none";
  config_file?: string;
}

export function useSystemConfig() {
  return useQuery({
    queryKey: ["system", "config"],
    queryFn: () => apiFetch<SystemConfig>("/system/config"),
  });
}

export function useSaveConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { tmdb_api_key?: string; ai_api_key?: string }) =>
      apiFetch<{ saved: boolean; config_file: string }>("/system/config", {
        method: "PUT",
        body: JSON.stringify(body),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["system", "status"] });
      qc.invalidateQueries({ queryKey: ["system", "config"] });
      toast.success("Settings saved");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}
