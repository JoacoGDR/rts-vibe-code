import { request } from "./client";

export interface ChatMessageView {
  id: string;
  match_id: string;
  scope: string;
  author_user_id: string;
  author_slot: string;
  body: string;
  sent_at: string;
}

export interface SendChatBody {
  scope: string;
  body: string;
  author_slot?: string;
}

export const chatApi = {
  history(matchID: string, scope: string, limit = 50, before?: string) {
    const params = new URLSearchParams({ scope, limit: String(limit) });
    if (before) params.set("before", before);
    return request<ChatMessageView[]>(
      "GET",
      `/api/v1/matches/${matchID}/chat?${params.toString()}`,
    );
  },
  send(matchID: string, body: SendChatBody) {
    return request<ChatMessageView>("POST", `/api/v1/matches/${matchID}/chat`, body);
  },
};
