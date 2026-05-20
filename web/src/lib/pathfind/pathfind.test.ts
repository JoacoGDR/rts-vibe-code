import { describe, expect, it } from "vitest";
import { buildGraph, route, snapTarget } from "./index";

const tiny2p = {
  id: "tiny-2p",
  name: "Tiny",
  provinces: [
    { id: "A", x: 100, y: 100 },
    { id: "B", x: 700, y: 100 },
    { id: "C", x: 100, y: 500 },
    { id: "D", x: 700, y: 500 },
  ],
  edges: [
    { from: "A", to: "B" },
    { from: "A", to: "C" },
    { from: "B", to: "D" },
    { from: "C", to: "D" },
    { from: "A", to: "D" },
    { from: "B", to: "C" },
  ],
  slots: [],
  starting_units: [],
};

describe("pathfind", () => {
  it("snaps to node at province center", () => {
    const g = buildGraph(tiny2p);
    const t = snapTarget(g, 100, 100);
    expect(t.kind).toBe("node");
    expect(t.province).toBe("A");
  });

  it("routes A to D", () => {
    const g = buildGraph(tiny2p);
    const legs = route(g, 100, 100, { kind: "node", province: "D", x: 700, y: 500 });
    expect(legs).not.toBeNull();
    expect(legs!.length).toBeGreaterThan(0);
    const last = legs![legs!.length - 1];
    expect(last.toX).toBe(700);
    expect(last.toY).toBe(500);
  });
});
