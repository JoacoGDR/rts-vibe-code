import { useEffect, useRef } from "react";
import { Application, Container, Graphics, Text, TextStyle } from "pixi.js";
import { useAppStore } from "../../app/store";
import { resolveSlotColor } from "../../lib/color";
import type { UnitState } from "../../types/wire";

interface UnitOverlayProps {
  selectedUnitID: string | null;
  draggingUnitId: string | null;
  width: number;
  height: number;
}

/** Renders unit markers; interaction is handled by MapStage hit-testing. */
export function UnitOverlay({ selectedUnitID, draggingUnitId, width, height }: UnitOverlayProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const unitsLayer = useRef<Container | null>(null);
  const selectedRef = useRef(selectedUnitID);
  const draggingRef = useRef(draggingUnitId);

  selectedRef.current = selectedUnitID;
  draggingRef.current = draggingUnitId;

  useEffect(() => {
    let mounted = true;
    (async () => {
      if (!containerRef.current) return;
      const app = new Application();
      await app.init({
        backgroundAlpha: 0,
        width,
        height,
        antialias: true,
      });
      if (!mounted) {
        app.destroy(true);
        return;
      }
      containerRef.current.appendChild(app.canvas);
      appRef.current = app;
      unitsLayer.current = new Container();
      app.stage.addChild(unitsLayer.current);

      const unsub = useAppStore.subscribe((state) => {
        if (state.matchState) draw(state.matchState);
      });
      const cur = useAppStore.getState().matchState;
      if (cur) draw(cur);

      function draw(state: {
        units: UnitState[];
        players: { id: string; color?: string }[];
      }) {
        if (!unitsLayer.current) return;
        unitsLayer.current.removeChildren();
        const colorBySlot: Record<string, string | undefined> = {};
        for (const p of state.players) colorBySlot[p.id] = p.color;

        for (const u of state.units) {
          const fill = resolveSlotColor(u.owner_id, colorBySlot[u.owner_id]);
          const g = new Graphics();
          g.rect(u.x - 10, u.y - 10, 20, 20).fill(fill).stroke({ width: 2, color: 0xffffff });
          if (selectedRef.current === u.id || draggingRef.current === u.id) {
            g.rect(u.x - 14, u.y - 14, 28, 28).stroke({ width: 2, color: 0xc9a227 });
          }
          unitsLayer.current.addChild(g);

          const hp = new Text({
            text: `${Math.round(u.hp)}`,
            style: new TextStyle({ fill: "#e8eaed", fontSize: 10, fontFamily: "IBM Plex Mono" }),
          });
          hp.x = u.x - hp.width / 2;
          hp.y = u.y + 12;
          unitsLayer.current.addChild(hp);
        }
      }

      return () => unsub();
    })();

    return () => {
      mounted = false;
      appRef.current?.destroy(true);
      appRef.current = null;
    };
  }, [width, height]);

  return <div ref={containerRef} className="unit-overlay" aria-hidden />;
}

