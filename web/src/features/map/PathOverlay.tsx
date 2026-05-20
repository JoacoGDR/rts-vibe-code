import { useMemo } from "react";
import type { DragPreview } from "../../app/store";
import type { Leg } from "../../lib/pathfind";
import type { UnitState } from "../../types/wire";

interface PathOverlayProps {
  width: number;
  height: number;
  dragPreview: DragPreview | null;
  selectedUnit: UnitState | null;
}

function legPoints(legs: Leg[], startX: number, startY: number): string {
  const pts: string[] = [`${startX},${startY}`];
  for (const l of legs) {
    pts.push(`${l.toX},${l.toY}`);
  }
  return pts.join(" ");
}

function arrowHead(x: number, y: number, fromX: number, fromY: number): string {
  const dx = x - fromX;
  const dy = y - fromY;
  const len = Math.hypot(dx, dy) || 1;
  const ux = dx / len;
  const uy = dy / len;
  const size = 12;
  const px = -uy;
  const py = ux;
  const tipX = x;
  const tipY = y;
  const b1x = x - ux * size + px * (size * 0.4);
  const b1y = y - uy * size + py * (size * 0.4);
  const b2x = x - ux * size - px * (size * 0.4);
  const b2y = y - uy * size - py * (size * 0.4);
  return `${tipX},${tipY} ${b1x},${b1y} ${b2x},${b2y}`;
}

function unitPathLegs(unit: UnitState): Leg[] {
  if (!unit.path?.length) return [];
  return unit.path.map((l) => ({
    fromProv: l.from_prov ?? "",
    toProv: l.to_prov ?? "",
    fromX: l.from_x ?? 0,
    fromY: l.from_y ?? 0,
    toX: l.to_x ?? 0,
    toY: l.to_y ?? 0,
  }));
}

export function PathOverlay({ width, height, dragPreview, selectedUnit }: PathOverlayProps) {
  const staticPath = useMemo(() => {
    if (!selectedUnit?.path?.length) return null;
    const idx = selectedUnit.path_index ?? 0;
    const legs = unitPathLegs(selectedUnit);
    const remaining = legs.slice(idx);
    if (remaining.length === 0) return null;
    const active = [remaining[0]];
    const queued = remaining.slice(1);
    const startX = selectedUnit.x;
    const startY = selectedUnit.y;
    return { active, queued, startX, startY };
  }, [selectedUnit]);

  return (
    <svg
      className="path-overlay"
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      aria-hidden
    >
      {staticPath && staticPath.queued.length > 0 && (
        <polyline
          className="path-line path-line--queued"
          points={legPoints(
            staticPath.queued,
            staticPath.active[0]?.toX ?? staticPath.startX,
            staticPath.active[0]?.toY ?? staticPath.startY,
          )}
          fill="none"
        />
      )}
      {staticPath && staticPath.active.length > 0 && (
        <>
          <polyline
            className="path-line path-line--active"
            points={legPoints(staticPath.active, staticPath.startX, staticPath.startY)}
            fill="none"
          />
          {(() => {
            const last = staticPath.active[staticPath.active.length - 1];
            const prev =
              staticPath.active.length > 1
                ? staticPath.active[staticPath.active.length - 2]
                : { toX: staticPath.startX, toY: staticPath.startY };
            return (
              <polygon
                className="path-arrow"
                points={arrowHead(last.toX, last.toY, prev.toX, prev.toY)}
              />
            );
          })()}
        </>
      )}

      {dragPreview && dragPreview.legs.length > 0 && (
        <>
          <polyline
            className={`path-line path-line--preview${dragPreview.attackTerminus ? " path-line--attack" : ""}`}
            points={legPoints(
              dragPreview.legs,
              dragPreview.legs[0].fromX,
              dragPreview.legs[0].fromY,
            )}
            fill="none"
          />
          {(() => {
            const last = dragPreview.legs[dragPreview.legs.length - 1];
            const prev =
              dragPreview.legs.length > 1
                ? dragPreview.legs[dragPreview.legs.length - 2]
                : dragPreview.legs[0];
            return (
              <polygon
                className={`path-arrow${dragPreview.attackTerminus ? " path-arrow--attack" : ""}`}
                points={arrowHead(last.toX, last.toY, prev.toX, prev.toY)}
              />
            );
          })()}
        </>
      )}
    </svg>
  );
}
