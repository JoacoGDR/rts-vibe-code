import { dist, neighbors } from "./graph";
import { goalProvince } from "./snap";
import type { Leg, PathGraph, Target } from "./types";

type ANode = { id: string; f: number; g: number; parent?: string; index: number; closed?: boolean };

function astar(g: PathGraph, start: string, goal: string): string[] | null {
  const open: ANode[] = [];
  const nodes = new Map<string, ANode>();
  const push = (n: ANode) => {
    n.index = open.length;
    open.push(n);
  };
  const pop = (): ANode | undefined => {
    open.sort((a, b) => a.f - b.f);
    return open.shift();
  };

  const startN: ANode = { id: start, g: 0, f: dist(g, start, goal), index: -1 };
  nodes.set(start, startN);
  push(startN);

  while (open.length > 0) {
    const cur = pop()!;
    if (cur.closed) continue;
    cur.closed = true;
    if (cur.id === goal) {
      const out: string[] = [goal];
      let c = goal;
      for (;;) {
        const n = nodes.get(c);
        if (!n?.parent) break;
        c = n.parent;
        out.push(c);
      }
      return out.reverse();
    }
    for (const nb of neighbors(g, cur.id)) {
      const tentG = cur.g + dist(g, cur.id, nb);
      let nn = nodes.get(nb);
      if (!nn) {
        nn = { id: nb, g: tentG, f: tentG + dist(g, nb, goal), index: -1 };
        nodes.set(nb, nn);
        push(nn);
        continue;
      }
      if (nn.closed) continue;
      if (tentG >= nn.g) continue;
      nn.parent = cur.id;
      nn.g = tentG;
      nn.f = tentG + dist(g, nb, goal);
    }
  }
  return null;
}

export function route(g: PathGraph, fromX: number, fromY: number, target: Target): Leg[] | null {
  const startProv = nearestProvince(fromX, fromY, g);
  const goalProv = goalProvince(target);
  if (!startProv || !goalProv) return null;

  let via: string[];
  if (startProv !== goalProv) {
    const path = astar(g, startProv, goalProv);
    if (!path) return null;
    via = path;
  } else {
    via = [startProv];
  }

  const legs: Leg[] = [];
  let curX = fromX;
  let curY = fromY;
  let curProv = startProv;

  for (let i = 1; i < via.length; i++) {
    const next = via[i];
    const c = g.coords[next];
    legs.push({
      fromProv: curProv,
      toProv: next,
      fromX: curX,
      fromY: curY,
      toX: c.x,
      toY: c.y,
    });
    curX = c.x;
    curY = c.y;
    curProv = next;
  }

  const tx = target.x;
  const ty = target.y;
  if (legs.length === 0 || legs[legs.length - 1].toX !== tx || legs[legs.length - 1].toY !== ty) {
    legs.push({
      fromProv: curProv,
      toProv: goalProv,
      fromX: curX,
      fromY: curY,
      toX: tx,
      toY: ty,
    });
  }
  if (legs.length === 0) {
    legs.push({
      fromProv: startProv,
      toProv: goalProv,
      fromX: fromX,
      fromY: fromY,
      toX: tx,
      toY: ty,
    });
  }
  return legs;
}

function nearestProvince(x: number, y: number, g: PathGraph): string {
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
