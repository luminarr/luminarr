import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "./client";
import type { RadarrPreviewResult, RadarrImportOptions, RadarrImportResult } from "@/types";

// Go nil slices serialize to JSON null, not []. Normalize here so the UI
// never has to guard against null arrays.
function normalizePreview(d: RadarrPreviewResult): RadarrPreviewResult {
  return {
    ...d,
    quality_profiles: d.quality_profiles ?? [],
    root_folders: d.root_folders ?? [],
    indexers: d.indexers ?? [],
    download_clients: d.download_clients ?? [],
  };
}

function normalizeImportResult(d: RadarrImportResult): RadarrImportResult {
  return { ...d, errors: d.errors ?? [] };
}

export function useRadarrPreview() {
  return useMutation({
    mutationFn: (req: { url: string; api_key: string }) =>
      apiFetch<RadarrPreviewResult>("/import/radarr/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }).then(normalizePreview),
  });
}

export function useRadarrImport() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: { url: string; api_key: string; options: RadarrImportOptions }) =>
      apiFetch<RadarrImportResult>("/import/radarr/execute", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      }).then(normalizeImportResult),
    onSuccess: () => {
      // Invalidate all lists that import may have populated so navigating
      // to those pages shows fresh data without a manual browser refresh.
      qc.invalidateQueries({ queryKey: ["quality-profiles"] });
      qc.invalidateQueries({ queryKey: ["libraries"] });
      qc.invalidateQueries({ queryKey: ["indexers"] });
      qc.invalidateQueries({ queryKey: ["download-clients"] });
    },
  });
}
