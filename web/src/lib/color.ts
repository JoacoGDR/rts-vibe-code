// Slot color helpers. Prefer the colour the server sends on
// PlayerState/ProvinceState; this is just a fallback for renders that
// don't have it (e.g. before the first state envelope arrives).

const FALLBACK_BY_SLOT: Record<string, number> = {
  red: 0xe8523a,
  blue: 0x3a8ce8,
  green: 0x3ae888,
  yellow: 0xe8c93a,
};

const DEFAULT_COLOR = 0x888888;

export function slotColorNumber(slot: string | undefined): number {
  if (!slot) return DEFAULT_COLOR;
  return FALLBACK_BY_SLOT[slot] ?? DEFAULT_COLOR;
}

// hexToNumber converts a #rrggbb string to a numeric Pixi colour.
// Returns null when the input is invalid so callers can fall back to a
// slot-name lookup.
export function hexToNumber(hex: string | undefined): number | null {
  if (!hex) return null;
  const trimmed = hex.startsWith("#") ? hex.slice(1) : hex;
  if (!/^[0-9a-fA-F]{6}$/.test(trimmed)) return null;
  return parseInt(trimmed, 16);
}

// resolveSlotColor prefers an explicit hex (server-provided) and falls
// back to the slot name table.
export function resolveSlotColor(slot: string | undefined, hex?: string): number {
  return hexToNumber(hex) ?? slotColorNumber(slot);
}
