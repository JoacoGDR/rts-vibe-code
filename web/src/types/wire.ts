// Mirrors pkg/shared/wire/messages.go. Keep field names in sync.

export const WIRE_VERSION = 3;

export type ClientMessageType = "hello" | "command" | "ping" | "resync" | "chat" | "goodbye";
export type ServerMessageType = "hello_ack" | "state" | "event" | "chat" | "error" | "pong";

export interface ClientCommand {
  kind: string;
  unit_id?: string;
  from?: string;
  to?: string;
  issued_at: string;
  idempotency_key: string;
  args?: Record<string, string>;
}

export interface ClientEnvelope {
  type: ClientMessageType;
  wire_version?: number;
  id?: string;
  match_id?: string;
  command?: ClientCommand;
  chat?: ChatOutbound;
  resync_seq?: number;
}

export interface ServerEnvelope {
  type: ServerMessageType;
  id?: string;
  match_id?: string;
  seq?: number;
  sent_at: string;
  state?: MatchState;
  event?: ServerEvent;
  chat?: ChatInbound;
  error?: { code: string; message: string };
}

export interface ChatOutbound {
  scope: "world" | "coalition" | "dm";
  target_user_id?: string;
  body: string;
}

export interface ChatInbound {
  id: string;
  match_id: string;
  scope: string;
  author_user_id: string;
  author_display_name?: string;
  author_slot?: string;
  body: string;
  sent_at: string;
}

export interface MatchState {
  match_id: string;
  tick: number;
  game_time: string;
  provinces: ProvinceState[];
  units: UnitState[];
  players: PlayerState[];
  resources?: Record<string, ResourcePool>;
  buildings?: BuildingState[];
  queues?: QueueState;
  diplomacy?: TreatyState[];
  pacts?: PactState[];
}

export type Stance = "war" | "peace" | "alliance";
export type PendingStance = "" | "peace" | "alliance";

export interface TreatyState {
  slot_a: string;
  slot_b: string;
  stance: Stance;
  pending?: PendingStance;
  pending_from?: string;
  changed_at: string;
}

export interface PactState {
  from: string;
  to: string;
  kind: "share_map" | "right_of_way";
  granted_at: string;
}

export interface ResourcePool {
  manpower: number;
  food: number;
  iron: number;
}

export interface BuildingState {
  type: string;
  province: string;
  owner: string;
}

export interface QueueState {
  recruits?: QueuedRecruit[];
  constructions?: QueuedConstruction[];
}

export interface QueuedRecruit {
  province: string;
  unit_type: string;
  owner: string;
  completes_at: string;
}

export interface QueuedConstruction {
  province: string;
  building_type: string;
  owner: string;
  completes_at: string;
}

export interface ProvinceState {
  id: string;
  owner_id?: string;
  x: number;
  y: number;
  capital?: boolean;
}

export interface PathLegState {
  from_prov?: string;
  to_prov?: string;
  from_x?: number;
  from_y?: number;
  to_x?: number;
  to_y?: number;
  arrives_at?: string;
}

export interface UnitState {
  id: string;
  owner_id: string;
  type: string;
  x: number;
  y: number;
  hp: number;
  origin?: string;
  dest?: string;
  started_at?: string;
  arrives_at?: string;
  path?: PathLegState[];
  path_index?: number;
}

export interface PlayerState {
  id: string;
  name: string;
  color?: string;
  alive: boolean;
}

export interface ServerEvent {
  kind: string;
  occur_at: string;
  unit_id?: string;
  province?: string;
  extra?: Record<string, unknown>;
}
