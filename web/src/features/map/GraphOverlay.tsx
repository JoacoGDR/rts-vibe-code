import { useMemo } from "react";
import type { PathGraph } from "../../lib/pathfind/types";

interface GraphOverlayProps {
  width: number;
  height: number;
  graph: PathGraph | null;
}

export function GraphOverlay({ width, height, graph }: GraphOverlayProps) {
  const edges = useMemo(() => {
    if (!graph) return [];
    return graph.edges.map((edge) => (
      <line
        key={`${edge.from}-${edge.to}`}
        className="graph-edge"
        x1={edge.ax}
        y1={edge.ay}
        x2={edge.bx}
        y2={edge.by}
      />
    ));
  }, [graph]);

  const nodes = useMemo(() => {
    if (!graph) return [];
    return Object.entries(graph.coords).map(([provinceId, coord]) => (
      <circle
        key={provinceId}
        className="graph-node"
        cx={coord.x}
        cy={coord.y}
        r={4}
      />
    ));
  }, [graph]);

  if (!graph) return null;

  return (
    <svg
      className="graph-overlay"
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      aria-hidden
    >
      {edges}
      {nodes}
    </svg>
  );
}
