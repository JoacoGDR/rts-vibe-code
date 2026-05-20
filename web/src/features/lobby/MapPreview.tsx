import { Panel } from "../../components/ui";
import { resolveMapSvgUrl, useMapAssets } from "../map/useMapAssets";
import type { MapDef } from "../../types/api";

interface MapPreviewProps {
  map: MapDef | undefined;
}

export function MapPreview({ map }: MapPreviewProps) {
  const svgUrl = map ? resolveMapSvgUrl(map.id, map.svg_url) : undefined;
  const { assets, loading, error } = useMapAssets(svgUrl);

  return (
    <Panel title="Theater preview" className="map-preview">
      {!map && <p className="muted">No map selected.</p>}
      {map && loading && <p className="muted">Loading map…</p>}
      {map && error && <p className="err">{error}</p>}
      {map && assets && (
        <div
          className="map-preview__svg"
          ref={(el) => {
            if (el && !el.querySelector("svg")) {
              el.replaceChildren(assets.svg.cloneNode(true));
            }
          }}
        />
      )}
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
