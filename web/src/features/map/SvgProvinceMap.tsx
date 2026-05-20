import { useEffect, useRef } from "react";
import type { MapAssetIndex } from "./useMapAssets";
import { ownerFill, resolveProvinceVisual, visualToCssVar } from "./provinceStyles";
import type { ProvinceState, PactState, TreatyState } from "../../types/wire";
import "./map.css";

export interface SvgProvinceMapProps {
  assets: MapAssetIndex;
  provinces: ProvinceState[];
  players: { id: string; color?: string }[];
  diplomacy?: TreatyState[];
  pacts?: PactState[];
  ownSlot?: string;
  hoveredProvinceId: string | null;
  selectedProvinceId: string | null;
}

export function SvgProvinceMap({
  assets,
  provinces,
  players,
  diplomacy,
  pacts,
  ownSlot,
  hoveredProvinceId,
  selectedProvinceId,
}: SvgProvinceMapProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const mountedRef = useRef(false);

  const colorBySlot: Record<string, string | undefined> = {};
  for (const p of players) colorBySlot[p.id] = p.color;

  useEffect(() => {
    if (!hostRef.current || mountedRef.current) return;
    const clone = assets.svg.cloneNode(true) as SVGSVGElement;
    hostRef.current.replaceChildren(clone);
    mountedRef.current = true;
  }, [assets.svg]);

  useEffect(() => {
    if (!hostRef.current) return;
    const visible = new Set(provinces.map((p) => p.id));

    hostRef.current.querySelectorAll("[data-province-id]").forEach((el) => {
      const html = el as SVGElement;
      const id = html.getAttribute("data-province-id")!;
      if (!visible.has(id)) {
        html.style.display = "none";
        return;
      }
      html.style.display = "";
      const p = provinces.find((x) => x.id === id);
      if (!p) return;

      const visual = resolveProvinceVisual(p, ownSlot, diplomacy, pacts);
      const fill =
        visual === "owned" || visual === "ally" || visual === "enemy"
          ? ownerFill(p.owner_id, colorBySlot)
          : visualToCssVar(visual);

      html.style.setProperty("--province-fill", fill);
      html.style.setProperty("--province-stroke", visual === "owned" ? "var(--accent-command)" : "var(--border-brass)");
      html.classList.toggle("province--hover", id === hoveredProvinceId);
      html.classList.toggle("province--selected", id === selectedProvinceId);
    });
  }, [provinces, players, diplomacy, pacts, ownSlot, hoveredProvinceId, selectedProvinceId]);

  return <div ref={hostRef} className="map-svg-host" />;
}
