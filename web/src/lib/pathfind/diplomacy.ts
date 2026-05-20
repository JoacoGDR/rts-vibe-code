import type { PactState, TreatyState } from "../../types/wire";
import type { Leg, ProvinceOwners } from "./types";

function stance(
  mover: string,
  owner: string,
  diplomacy: TreatyState[] | undefined,
): "war" | "peace" | "alliance" {
  if (!owner || mover === owner) return "peace";
  const key = [mover, owner].sort().join("|");
  const t = diplomacy?.find(
    (d) => [d.slot_a, d.slot_b].sort().join("|") === key,
  );
  return (t?.stance as "war" | "peace" | "alliance") ?? "war";
}

function hasPact(
  from: string,
  to: string,
  kind: "right_of_way",
  pacts: PactState[] | undefined,
): boolean {
  return Boolean(pacts?.some((p) => p.from === from && p.to === to && p.kind === kind));
}

export function mayMoveThrough(
  mover: string,
  owner: string,
  diplomacy: TreatyState[] | undefined,
  pacts: PactState[] | undefined,
): boolean {
  if (!owner || mover === owner) return true;
  const s = stance(mover, owner, diplomacy);
  if (s === "alliance") return true;
  if (s === "peace") return hasPact(owner, mover, "right_of_way", pacts);
  return false;
}

export function hostileLegEnd(
  mover: string,
  owners: ProvinceOwners,
  leg: Leg,
  diplomacy: TreatyState[] | undefined,
): boolean {
  const owner = owners[leg.toProv] ?? "";
  if (!owner || owner === mover) return false;
  return stance(mover, owner, diplomacy) === "war";
}

export function routeBlocked(
  mover: string,
  owners: ProvinceOwners,
  legs: Leg[],
  diplomacy: TreatyState[] | undefined,
  pacts: PactState[] | undefined,
): boolean {
  for (const leg of legs) {
    const owner = owners[leg.toProv] ?? "";
    if (!owner || owner === mover) continue;
    if (mayMoveThrough(mover, owner, diplomacy, pacts)) continue;
    if (stance(mover, owner, diplomacy) !== "war") return true;
  }
  return false;
}

export function routeAttackTerminus(
  mover: string,
  owners: ProvinceOwners,
  legs: Leg[],
  diplomacy: TreatyState[] | undefined,
): boolean {
  return legs.some((l) => hostileLegEnd(mover, owners, l, diplomacy));
}

export function provinceOwners(
  provinces: { id: string; owner_id?: string }[],
): ProvinceOwners {
  const out: ProvinceOwners = {};
  for (const p of provinces) {
    out[p.id] = p.owner_id ?? "";
  }
  return out;
}
