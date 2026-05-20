import type { MapDef } from "../../types/api";
import type { PathGraph } from "./types";

export function buildGraph(mapDef: MapDef): PathGraph {
  const coords: PathGraph["coords"] = {};
  for (const p of mapDef.provinces) {
    coords[p.id] = { x: p.x, y: p.y };
  }
  const edges: PathGraph["edges"] = [];
  for (const e of mapDef.edges) {
    const a = coords[e.from];
    const b = coords[e.to];
    if (!a || !b) continue;
    edges.push({ from: e.from, to: e.to, ax: a.x, ay: a.y, bx: b.x, by: b.y });
  }
  return { mapDef, coords, edges };
}

export function neighbors(g: PathGraph, id: string): string[] {
  const out = new Set<string>();
  for (const e of g.edges) {
    if (e.from === id) out.add(e.to);
    if (e.to === id) out.add(e.from);
  }
  return [...out].sort();
}

export function dist(g: PathGraph, a: string, b: string): number {
  const va = g.coords[a];
  const vb = g.coords[b];
  if (!va || !vb) return Infinity;
  return Math.hypot(va.x - vb.x, va.y - vb.y);
}
