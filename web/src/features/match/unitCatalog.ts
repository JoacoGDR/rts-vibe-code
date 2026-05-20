import type { ResourcePool } from "../../types/wire";

export interface UnitCatalogEntry {
  id: string;
  label: string;
  cost: ResourcePool;
  needsBuilding?: string;
}

export interface BuildingCatalogEntry {
  id: string;
  label: string;
  cost: ResourcePool;
}

export const RECRUITABLE_UNITS: UnitCatalogEntry[] = [
  { id: "infantry", label: "Infantry", cost: { manpower: 5, food: 3, iron: 0 } },
  { id: "cavalry", label: "Cavalry", cost: { manpower: 8, food: 5, iron: 3 } },
  {
    id: "armor",
    label: "Armor",
    cost: { manpower: 12, food: 5, iron: 8 },
    needsBuilding: "factory",
  },
];

export const BUILDINGS: BuildingCatalogEntry[] = [
  { id: "factory", label: "Factory", cost: { manpower: 10, food: 0, iron: 20 } },
];

export function canAfford(pool: ResourcePool | undefined, cost: ResourcePool): boolean {
  if (!pool) return false;
  return (
    pool.manpower >= cost.manpower &&
    pool.food >= cost.food &&
    (pool.iron ?? 0) >= (cost.iron ?? 0)
  );
}

export function formatCost(cost: ResourcePool): string {
  const parts: string[] = [];
  if (cost.manpower > 0) parts.push(`${cost.manpower} MP`);
  if (cost.food > 0) parts.push(`${cost.food} food`);
  if ((cost.iron ?? 0) > 0) parts.push(`${cost.iron} iron`);
  return parts.join(", ") || "free";
}
