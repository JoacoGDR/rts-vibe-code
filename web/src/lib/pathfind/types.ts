import type { MapDef } from "../../types/api";

export const NODE_SNAP_RADIUS = 24;
export const EDGE_SNAP_MAX_DIST = 20;

export type TargetKind = "node" | "edge";

export interface Target {
  kind: TargetKind;
  province?: string;
  edgeFrom?: string;
  edgeTo?: string;
  x: number;
  y: number;
  t?: number;
}

export interface Leg {
  fromProv: string;
  toProv: string;
  fromX: number;
  fromY: number;
  toX: number;
  toY: number;
}

export type ProvinceOwners = Record<string, string>;

export interface PathGraph {
  mapDef: MapDef;
  coords: Record<string, { x: number; y: number }>;
  edges: { from: string; to: string; ax: number; ay: number; bx: number; by: number }[];
}
