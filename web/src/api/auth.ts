import { request, tokenStore } from "./client";
import type { MeResponse, Session, WSTicketResponse } from "../types/api";

export const authApi = {
  setToken: tokenStore.set,
  clearToken: tokenStore.clear,
  getToken: tokenStore.get,

  register(email: string, password: string, displayName: string) {
    return request<Session>("POST", "/api/v1/auth/register", {
      email,
      password,
      display_name: displayName,
    });
  },

  login(email: string, password: string) {
    return request<Session>("POST", "/api/v1/auth/login", { email, password });
  },

  me() {
    return request<MeResponse>("GET", "/api/v1/auth/me");
  },

  wsTicket() {
    return request<WSTicketResponse>("POST", "/api/v1/auth/ws-ticket");
  },
};
