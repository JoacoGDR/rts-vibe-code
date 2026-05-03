// Mirrors pkg/api JSON tags (e.g. TokenResponse uses snake_case).
export interface Session {
  access_token: string;
  expires_at: string;
  user: User;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  color: string;
}

export interface MeResponse {
  id: string;
  email: string;
  display_name: string;
}

export interface WSTicketResponse {
  ticket: string;
}

export interface MatchView {
  id: string;
  name: string;
  map_id: string;
  status: "waiting" | "active" | "ended";
  created_at: string;
  started_at?: string;
  ended_at?: string;
  winner_user_id?: string;
  players: PlayerView[];
}

export interface PlayerView {
  user_id: string;
  slot: string;
  color: string;
  alive: boolean;
}

export interface MapDef {
  id: string;
  name: string;
  provinces: { id: string; x: number; y: number; name?: string }[];
  edges: { from: string; to: string }[];
  slots: { id: string; color: string; capital: string }[];
  starting_units: { slot: string; type: string; province: string; hp?: number }[];
}
