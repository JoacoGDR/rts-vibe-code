import type { UnitState } from "../../types/wire";

const UNIT_HIT_RADIUS = 14;
const DRAG_THRESHOLD = 4;

export function pickProvinceId(
  clientX: number,
  clientY: number,
  root: HTMLElement | null,
): string | null {
  if (!root) return null;
  for (const el of document.elementsFromPoint(clientX, clientY)) {
    if (!root.contains(el)) continue;
    const id = el.getAttribute("data-province-id");
    if (id) return id;
  }
  return null;
}

export function pickOwnedUnit(
  units: UnitState[],
  ownSlot: string | undefined,
  worldX: number,
  worldY: number,
): UnitState | undefined {
  if (!ownSlot) return undefined;
  for (const u of units) {
    if (u.owner_id !== ownSlot) continue;
    if (Math.hypot(u.x - worldX, u.y - worldY) <= UNIT_HIT_RADIUS) {
      return u;
    }
  }
  return undefined;
}

export function pointerMovedEnough(
  x0: number,
  y0: number,
  x1: number,
  y1: number,
): boolean {
  return Math.hypot(x1 - x0, y1 - y0) >= DRAG_THRESHOLD;
}

export { DRAG_THRESHOLD };
