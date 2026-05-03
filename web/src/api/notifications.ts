import { request } from "./client";

export interface NotificationItem {
  id: string;
  user_id: string;
  match_id: string;
  kind: string;
  payload: Record<string, unknown>;
  created_at: string;
  sent_at?: string;
  read_at?: string;
}

export interface ListNotificationsResponse {
  items: NotificationItem[];
  unread_count: number;
}

export const notificationsApi = {
  list(after?: string, limit?: number) {
    const params = new URLSearchParams();
    if (after) params.set("after", after);
    if (limit) params.set("limit", String(limit));
    const qs = params.toString();
    return request<ListNotificationsResponse>(
      "GET",
      `/api/v1/notifications${qs ? `?${qs}` : ""}`,
    );
  },

  unreadCount() {
    return request<{ count: number }>("GET", "/api/v1/notifications/unread-count");
  },

  markRead() {
    return request<{ marked: number }>("POST", "/api/v1/notifications/mark-read");
  },
};
