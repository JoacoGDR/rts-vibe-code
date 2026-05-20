import { useEffect, useState } from "react";

export interface MapAssetIndex {
  svg: SVGSVGElement;
  provinces: Map<string, SVGElement>;
  viewBox: { x: number; y: number; width: number; height: number };
}

function parseViewBox(svg: SVGSVGElement) {
  const vb = svg.viewBox.baseVal;
  if (vb.width > 0) return { x: vb.x, y: vb.y, width: vb.width, height: vb.height };
  const w = parseFloat(svg.getAttribute("width") ?? "800");
  const h = parseFloat(svg.getAttribute("height") ?? "600");
  return { x: 0, y: 0, width: w, height: h };
}

export function useMapAssets(svgUrl: string | undefined) {
  const [assets, setAssets] = useState<MapAssetIndex | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!svgUrl) {
      setAssets(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);

    (async () => {
      try {
        const res = await fetch(svgUrl, { cache: "force-cache" });
        if (!res.ok) throw new Error(`Failed to load map: ${res.status}`);
        const text = await res.text();
        const doc = new DOMParser().parseFromString(text, "image/svg+xml");
        const svg = doc.documentElement;
        if (svg.tagName !== "svg") throw new Error("Invalid map SVG");

        svg.querySelectorAll("script").forEach((n) => n.remove());
        svg.setAttribute("class", "province-map-svg");

        const provinces = new Map<string, SVGElement>();
        svg.querySelectorAll("[data-province-id]").forEach((el) => {
          const id = el.getAttribute("data-province-id");
          if (id) provinces.set(id, el as SVGElement);
        });

        if (!cancelled) {
          setAssets({
            svg: svg as unknown as SVGSVGElement,
            provinces,
            viewBox: parseViewBox(svg as unknown as SVGSVGElement),
          });
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [svgUrl]);

  return { assets, loading, error };
}

/** Resolves svg_url from map def or falls back to bundled public asset. */
export function resolveMapSvgUrl(mapId: string, svgUrl?: string): string {
  if (svgUrl) return svgUrl;
  return `/maps/${mapId}.svg`;
}
