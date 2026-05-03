import { useEffect, useRef } from "react";
import { Application, Container, Graphics, Text, TextStyle } from "pixi.js";
import { useAppStore } from "../../app/store";
import { resolveSlotColor } from "../../lib/color";
import type { ProvinceState, UnitState } from "../../types/wire";

interface MapViewProps {
  ownSlot: string | undefined;
  onSelectUnit(unit: UnitState): void;
  onSelectProvince(province: ProvinceState): void;
  selectedUnitID: string | null;
}

export function MapView({ ownSlot, onSelectUnit, onSelectProvince, selectedUnitID }: MapViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const provincesLayer = useRef<Container | null>(null);
  const edgesLayer = useRef<Container | null>(null);
  const unitsLayer = useRef<Container | null>(null);
  const selectedRef = useRef<string | null>(null);
  const ownSlotRef = useRef<string | undefined>(ownSlot);
  const onSelectUnitRef = useRef(onSelectUnit);
  const onSelectProvinceRef = useRef(onSelectProvince);

  selectedRef.current = selectedUnitID;
  ownSlotRef.current = ownSlot;
  onSelectUnitRef.current = onSelectUnit;
  onSelectProvinceRef.current = onSelectProvince;

  useEffect(() => {
    let mounted = true;
    (async () => {
      if (!containerRef.current) return;
      const app = new Application();
      await app.init({
        background: "#0e1116",
        width: 800,
        height: 600,
        antialias: true,
      });
      if (!mounted) {
        app.destroy(true);
        return;
      }
      containerRef.current.appendChild(app.canvas);
      appRef.current = app;
      edgesLayer.current = new Container();
      provincesLayer.current = new Container();
      unitsLayer.current = new Container();
      app.stage.addChild(edgesLayer.current);
      app.stage.addChild(provincesLayer.current);
      app.stage.addChild(unitsLayer.current);

      const unsub = useAppStore.subscribe((state) => {
        if (state.matchState) draw(state.matchState);
      });

      const cur = useAppStore.getState().matchState;
      if (cur) draw(cur);

      function draw(state: {
        provinces: ProvinceState[];
        units: UnitState[];
        players: { id: string; color?: string }[];
      }) {
        if (!edgesLayer.current || !provincesLayer.current || !unitsLayer.current) return;
        edgesLayer.current.removeChildren();
        provincesLayer.current.removeChildren();
        unitsLayer.current.removeChildren();

        const colorBySlot: Record<string, string | undefined> = {};
        for (const p of state.players) colorBySlot[p.id] = p.color;

        for (const a of state.provinces) {
          for (const b of state.provinces) {
            if (a.id >= b.id) continue;
            const g = new Graphics();
            g.moveTo(a.x, a.y).lineTo(b.x, b.y).stroke({ width: 1, color: 0x223 });
            edgesLayer.current.addChild(g);
          }
        }

        for (const p of state.provinces) {
          const g = new Graphics();
          const fill = p.owner_id
            ? resolveSlotColor(p.owner_id, colorBySlot[p.owner_id])
            : 0x444444;
          g.circle(p.x, p.y, p.capital ? 28 : 20)
            .fill(fill)
            .stroke({ width: 2, color: 0xffffff });
          g.eventMode = "static";
          g.cursor = "pointer";
          g.on("pointertap", () => onSelectProvinceRef.current?.(p));
          provincesLayer.current.addChild(g);

          const label = new Text({
            text: p.id,
            style: new TextStyle({ fill: "#ffffff", fontSize: 14, fontWeight: "bold" }),
          });
          label.x = p.x - label.width / 2;
          label.y = p.y - label.height / 2;
          provincesLayer.current.addChild(label);
        }

        for (const u of state.units) {
          const owned = ownSlotRef.current && u.owner_id === ownSlotRef.current;
          const fill = resolveSlotColor(u.owner_id, colorBySlot[u.owner_id]);
          const g = new Graphics();
          g.rect(u.x - 10, u.y - 10, 20, 20)
            .fill(fill)
            .stroke({ width: 2, color: 0xffffff });
          if (selectedRef.current === u.id) {
            g.rect(u.x - 14, u.y - 14, 28, 28).stroke({ width: 2, color: 0xffff00 });
          }
          if (owned) {
            g.eventMode = "static";
            g.cursor = "pointer";
            g.on("pointertap", () => onSelectUnitRef.current?.(u));
          }
          unitsLayer.current.addChild(g);

          const hp = new Text({
            text: `${Math.round(u.hp)}`,
            style: new TextStyle({ fill: "#ffffff", fontSize: 11 }),
          });
          hp.x = u.x - hp.width / 2;
          hp.y = u.y + 14;
          unitsLayer.current.addChild(hp);
        }
      }

      return () => {
        unsub();
        app.destroy(true);
        appRef.current = null;
      };
    })();

    return () => {
      mounted = false;
      appRef.current?.destroy(true);
      appRef.current = null;
    };
  }, []);

  return <div ref={containerRef} className="map-view" />;
}
