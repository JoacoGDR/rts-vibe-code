import { EDGE_SNAP_MAX_DIST, NODE_SNAP_RADIUS, type PathGraph, type Target } from "./types";

function projectSegment(
  px: number,
  py: number,
  ax: number,
  ay: number,
  bx: number,
  by: number,
): { x: number; y: number; t: number; d: number } {
  const dx = bx - ax;
  const dy = by - ay;
  const len2 = dx * dx + dy * dy;
  if (len2 === 0) {
    const d = Math.hypot(px - ax, py - ay);
    return { x: ax, y: ay, t: 0, d };
  }
  let t = ((px - ax) * dx + (py - ay) * dy) / len2;
  t = Math.max(0, Math.min(1, t));
  const x = ax + t * dx;
  const y = ay + t * dy;
  return { x, y, t, d: Math.hypot(px - x, py - y) };
}

function nearestProvinceID(g: PathGraph, x: number, y: number): string {
  let best = "";
  let bestD = Infinity;
  for (const [id, c] of Object.entries(g.coords)) {
    const d = Math.hypot(x - c.x, y - c.y);
    if (d < bestD) {
      bestD = d;
      best = id;
    }
  }
  return best;
}

export function snapTarget(g: PathGraph, x: number, y: number): Target {
  let bestNode = "";
  let bestNodeDist = NODE_SNAP_RADIUS + 1;
  for (const [id, c] of Object.entries(g.coords)) {
    const d = Math.hypot(x - c.x, y - c.y);
    if (d <= NODE_SNAP_RADIUS && d < bestNodeDist) {
      bestNodeDist = d;
      bestNode = id;
    }
  }
  if (bestNode) {
    const c = g.coords[bestNode];
    return { kind: "node", province: bestNode, x: c.x, y: c.y };
  }

  let bestEdge: PathGraph["edges"][0] | null = null;
  let bestDist = EDGE_SNAP_MAX_DIST + 1;
  let bestT = 0;
  let bestX = x;
  let bestY = y;
  for (const e of g.edges) {
    const { x: px, y: py, t, d } = projectSegment(x, y, e.ax, e.ay, e.bx, e.by);
    if (d <= EDGE_SNAP_MAX_DIST && d < bestDist) {
      bestEdge = e;
      bestDist = d;
      bestT = t;
      bestX = px;
      bestY = py;
    }
  }
  if (bestEdge) {
    return {
      kind: "edge",
      edgeFrom: bestEdge.from,
      edgeTo: bestEdge.to,
      x: bestX,
      y: bestY,
      t: bestT,
    };
  }

  const near = nearestProvinceID(g, x, y);
  const c = g.coords[near];
  return { kind: "node", province: near, x: c.x, y: c.y };
}

export function goalProvince(target: Target): string {
  if (target.kind === "node" && target.province) return target.province;
  if ((target.t ?? 0) <= 0.5) return target.edgeFrom ?? "";
  return target.edgeTo ?? "";
}

