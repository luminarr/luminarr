import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiFetch } from "./client";
import type { Collection, PersonSearchResult, EntitySearchResult } from "@/types";

export function useCollections() {
  return useQuery({
    queryKey: ["collections"],
    queryFn: () => apiFetch<Collection[]>("/collections"),
  });
}

export function useCollection(id: string) {
  return useQuery({
    queryKey: ["collections", id],
    queryFn: () => apiFetch<Collection>(`/collections/${id}`),
    enabled: !!id,
    staleTime: 30_000,
  });
}

export function useCreateCollection() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { person_id: number; person_type: string }) =>
      apiFetch<Collection>("/collections", { method: "POST", body: JSON.stringify(body) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["collections"] }),
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useDeleteCollection() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/collections/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["collections"] });
      toast.success("Collection deleted");
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useAddMissing(collectionId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { library_id: string; quality_profile_id: string; minimum_availability: string }) =>
      apiFetch<{ added: number; skipped_duplicates: number }>(
        `/collections/${collectionId}/add-missing`,
        { method: "POST", body: JSON.stringify(body) }
      ),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["collections", collectionId] });
      toast.success(`Added ${data?.added ?? 0} film${(data?.added ?? 0) === 1 ? "" : "s"} to library`);
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useAddSelected(collectionId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      tmdb_ids: number[];
      library_id: string;
      quality_profile_id: string;
      minimum_availability: string;
    }) =>
      apiFetch<{ added: number; skipped_duplicates: number }>(
        `/collections/${collectionId}/add-selected`,
        { method: "POST", body: JSON.stringify(body) }
      ),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["collections", collectionId] });
      toast.success(`Added ${data?.added ?? 0} film${(data?.added ?? 0) === 1 ? "" : "s"} to library`);
    },
    onError: (err) => toast.error((err as Error).message),
  });
}

export function useSearchPeople(query: string) {
  return useQuery({
    queryKey: ["tmdb-people", query],
    queryFn: () =>
      apiFetch<PersonSearchResult[]>(`/tmdb/people/search?q=${encodeURIComponent(query)}`),
    enabled: query.trim().length >= 2,
    staleTime: 60_000,
  });
}

export function useSearchAll(query: string) {
  return useQuery({
    queryKey: ["tmdb-search", query],
    queryFn: () =>
      apiFetch<EntitySearchResult[]>(`/tmdb/search?q=${encodeURIComponent(query)}`),
    enabled: query.trim().length >= 2,
    staleTime: 60_000,
  });
}
