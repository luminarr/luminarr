import { useMutation } from "@tanstack/react-query";
import { apiFetch } from "./client";

export interface AICommandResponse {
  action:
    | "navigate"
    | "search_movie"
    | "query_library"
    | "search_releases"
    | "explain"
    | "fallback"
    | "auto_search"
    | "run_task";
  params?: Record<string, unknown>;
  result?: Record<string, unknown>;
  explanation: string;
  requires_confirmation?: boolean;
  pending_action_id?: string;
  confirmation_message?: string;
}

export function useSendAICommand() {
  return useMutation({
    mutationFn: (text: string) =>
      apiFetch<AICommandResponse>("/ai/command", {
        method: "POST",
        body: JSON.stringify({ text }),
      }),
  });
}

export function useConfirmAIAction() {
  return useMutation({
    mutationFn: (actionId: string) =>
      apiFetch<AICommandResponse>("/ai/command/confirm", {
        method: "POST",
        body: JSON.stringify({ action_id: actionId }),
      }),
  });
}
