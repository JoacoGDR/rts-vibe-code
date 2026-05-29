import { useEffect, useRef } from "react";
import { Panel } from "../../components/ui";
import { resolveMapSvgUrl, useMapAssets } from "../map/useMapAssets";
import type { MapDef } from "../../types/api";

interface MapPreviewProps {
  map: MapDef | undefined;
  selectedSlot?: string;
  players?: { slot: string; color: string }[];
}

export function MapPreview({ map, selectedSlot, players }: MapPreviewProps) {
  const svgUrl = map ? resolveMapSvgUrl(map.id, map.svg_url) : undefined;
  const { assets, loading, error } = useMapAssets(svgUrl);
  const containerRef = useRef<HTMLDivElement>(null);

  // Inserts a fresh SVG clone and immediately colors provinces in the live DOM.
  // Using a single effect ensures colors are applied to the cloned nodes that
  // are actually in the document — assets.provinces references the original
  // parsed nodes which are never inserted, so querying from containerRef is
  // the only way to reach the live elements.
  useEffect(() => {
    if (!containerRef.current || !assets || !map) return;

    containerRef.current.replaceChildren(assets.svg.cloneNode(true));

    const slotColors = new Map<string, string>();
    if (players && players.length > 0) {
      for (const p of players) slotColors.set(p.slot, p.color);
    } else if (selectedSlot) {
      const slotDef = map.slots.find((s) => s.id === selectedSlot);
      if (slotDef) slotColors.set(selectedSlot, slotDef.color);
    }

    for (const province of map.provinces) {
      const el = containerRef.current.querySelector(`[data-province-id="${province.id}"]`);
      if (!el) continue;
      const color = province.home_slot ? slotColors.get(province.home_slot) : undefined;
      (el as SVGElement).style.fill = color ?? "";
    }
  }, [assets, map, selectedSlot, players]);

  return (
    <Panel title="Theater preview" className="map-preview">
      {!map && <p className="muted">No map selected.</p>}
      {map && loading && <p className="muted">Loading map…</p>}
      {map && error && <p className="err">{error}</p>}
      {map && <div className="map-preview__svg" ref={containerRef} />}
      {map && (
        <dl className="map-preview__stats">
          <dt>Provinces</dt>
          <dd>{map.provinces.length}</dd>
          <dt>Nations</dt>
          <dd>{map.slots.length}</dd>
          {map.rules_summary && (
            <>
              <dt>Rules</dt>
              <dd>{map.rules_summary}</dd>
            </>
          )}
        </dl>
      )}
    </Panel>
  );
}
