import { request } from "./client";
import type { MapDef } from "../types/api";

export const mapsApi = {
  list() {
    return request<MapDef[]>("GET", "/api/v1/maps");
  },
};
