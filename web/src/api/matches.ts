import { request } from "./client";
import type { MatchView } from "../types/api";

export const matchesApi = {
  list() {
    return request<MatchView[]>("GET", "/api/v1/matches");
  },

  get(id: string) {
    return request<MatchView>("GET", `/api/v1/matches/${id}`);
  },

  create(name: string, mapID: string, slot: string) {
    return request<MatchView>("POST", "/api/v1/matches", {
      name,
      map_id: mapID,
      slot,
    });
  },

  join(id: string, slot?: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/join`, { slot });
  },

  start(id: string) {
    return request<MatchView>("POST", `/api/v1/matches/${id}/start`);
  },
};
