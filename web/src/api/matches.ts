import { request } from "./client";
import type { MatchStatsView, MatchView } from "../types/api";

export const matchesApi = {
  list() {
    return request<MatchView[]>("GET", "/api/v1/matches");
  },

  get(id: string) {
    return request<MatchView>("GET", `/api/v1/matches/${id}`);
  },

  create(
    name: string,
    mapID: string,
    slot: string,
    options?: { autoStart?: boolean },
  ) {
    return request<MatchView>("POST", "/api/v1/matches", {
      name,
      map_id: mapID,
      slot,
      ...(options?.autoStart === false ? { auto_start: false } : {}),
    });
  },

  join(id: string, slot?: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/join`, { slot });
  },

  start(id: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/start`);
  },

  leave(id: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/leave`);
  },

  kick(id: string, userID: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/kick`, { user_id: userID });
  },

  handoff(id: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/handoff`);
  },

  stats(id: string) {
    return request<MatchStatsView>("GET", `/api/v1/matches/${id}/stats`);
  },
};
