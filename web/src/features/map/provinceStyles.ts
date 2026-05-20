import type { PactState, ProvinceState, TreatyState } from "../../types/wire";

export type ProvinceVisual = "owned" | "ally" | "enemy" | "neutral" | "unknown";

export function resolveProvinceVisual(
  province: ProvinceState,
  ownSlot: string | undefined,
  diplomacy: TreatyState[] | undefined,
  pacts: PactState[] | undefined,
): ProvinceVisual {
  if (!province.owner_id) return "neutral";
  if (ownSlot && province.owner_id === ownSlot) return "owned";

  const treaty = diplomacy?.find(
    (t) =>
      ownSlot &&
      ((t.slot_a === ownSlot && t.slot_b === province.owner_id) ||
        (t.slot_b === ownSlot && t.slot_a === province.owner_id)),
  );
  if (treaty?.stance === "alliance") return "ally";
  if (treaty?.stance === "war") return "enemy";

  const sharesMap = pacts?.some(
    (p) =>
      ownSlot &&
      p.kind === "share_map" &&
      ((p.from === ownSlot && p.to === province.owner_id) ||
        (p.to === ownSlot && p.from === province.owner_id)),
  );
  if (sharesMap) return "ally";

  return "neutral";
}

export function visualToCssVar(visual: ProvinceVisual): string {
  switch (visual) {
    case "owned":
      return "var(--state-owned)";
    case "ally":
      return "var(--state-ally)";
    case "enemy":
      return "var(--state-enemy)";
    case "neutral":
      return "var(--state-neutral)";
    default:
      return "var(--state-unknown)";
  }
}

export function ownerFill(ownerId: string | undefined, colorBySlot: Record<string, string | undefined>): string {
  if (!ownerId) return "var(--state-neutral)";
  return colorBySlot[ownerId] ?? "var(--state-neutral)";
}
