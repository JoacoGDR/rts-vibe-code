import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { TransformComponent, TransformWrapper } from "react-zoom-pan-pinch";
import { useAppStore } from "../../app/store";
import {
  buildGraph,
  clientToWorld,
  provinceOwners,
  route,
  routeAttackTerminus,
  routeBlocked,
  snapTarget,
  type Target,
} from "../../lib/pathfind";
import type { MapDef } from "../../types/api";
import type { ProvinceState, UnitState } from "../../types/wire";
import { PathOverlay } from "./PathOverlay";
import { SvgProvinceMap } from "./SvgProvinceMap";
import { UnitOverlay } from "./UnitOverlay";
import { pickOwnedUnit, pickProvinceId, pointerMovedEnough } from "./mapPick";
import { resolveMapSvgUrl, useMapAssets } from "./useMapAssets";
import "./map.css";

export interface MapStageProps {
  mapDef: MapDef | undefined;
  ownSlot: string | undefined;
  selectedUnitID: string | null;
  onSelectUnit: (unit: UnitState) => void;
  onSelectProvince: (province: ProvinceState) => void;
  onCommitMove?: (unitId: string, target: Target, queue: boolean) => void;
}

interface PendingPointer {
  unit: UnitState;
  clientX: number;
  clientY: number;
  pointerId: number;
  shiftKey: boolean;
}

export function MapStage({
  mapDef,
  ownSlot,
  selectedUnitID,
  onSelectUnit,
  onSelectProvince,
  onCommitMove,
}: MapStageProps) {
  const matchState = useAppStore((s) => s.matchState);
  const selection = useAppStore((s) => s.selection);
  const drag = useAppStore((s) => s.drag);
  const dragPreview = useAppStore((s) => s.dragPreview);
  const setDrag = useAppStore((s) => s.setDrag);
  const setDragPreview = useAppStore((s) => s.setDragPreview);
  const setSelection = useAppStore((s) => s.setSelection);
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const layerRef = useRef<HTMLDivElement>(null);
  const pendingRef = useRef<PendingPointer | null>(null);
  const graph = useMemo(() => (mapDef ? buildGraph(mapDef) : null), [mapDef]);

  const svgUrl = mapDef ? resolveMapSvgUrl(mapDef.id, mapDef.svg_url) : undefined;
  const { assets, loading, error } = useMapAssets(svgUrl);
  const mapW = assets?.viewBox.width ?? 800;
  const mapH = assets?.viewBox.height ?? 600;

  const selectedProvinceId =
    selection?.kind === "province" ? selection.province.id : null;

  const provinces = useMemo(
    () => matchState?.provinces ?? [],
    [matchState?.provinces],
  );
  const players = matchState?.players ?? [];

  const selectedUnit = useMemo(
    () => matchState?.units.find((u) => u.id === selectedUnitID) ?? null,
    [matchState?.units, selectedUnitID],
  );

  const updateDragPreview = useCallback(
    (clientX: number, clientY: number) => {
      if (!drag || !graph || !matchState || !ownSlot || !layerRef.current) return;
      const unit = matchState.units.find((u) => u.id === drag.unitId);
      if (!unit) return;
      const rect = layerRef.current.getBoundingClientRect();
      const { x, y } = clientToWorld(clientX, clientY, rect, mapW, mapH);
      const target = snapTarget(graph, x, y);
      const legs = route(graph, unit.x, unit.y, target);
      if (!legs) {
        setDragPreview(null);
        return;
      }
      const owners = provinceOwners(provinces);
      if (routeBlocked(ownSlot, owners, legs, matchState.diplomacy, matchState.pacts)) {
        setDragPreview(null);
        return;
      }
      const attackTerminus = routeAttackTerminus(ownSlot, owners, legs, matchState.diplomacy);
      setDragPreview({ target, legs, attackTerminus });
    },
    [drag, graph, mapW, mapH, matchState, ownSlot, provinces, setDragPreview],
  );

  useEffect(() => {
    if (!drag) return;

    const onMove = (e: PointerEvent) => {
      if (e.pointerId !== drag.pointerId) return;
      updateDragPreview(e.clientX, e.clientY);
    };

    const onUp = (e: PointerEvent) => {
      if (e.pointerId !== drag.pointerId) return;
      const preview = useAppStore.getState().dragPreview;
      if (preview && onCommitMove) {
        onCommitMove(drag.unitId, preview.target, drag.shiftHeld);
      }
      setDrag(null);
      setDragPreview(null);
    };

    const onCancel = (e: PointerEvent) => {
      if (e.pointerId !== drag.pointerId) return;
      setDrag(null);
      setDragPreview(null);
    };

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setDrag(null);
        setDragPreview(null);
      }
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onCancel);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onCancel);
      window.removeEventListener("keydown", onKey);
    };
  }, [drag, onCommitMove, setDrag, setDragPreview, updateDragPreview]);

  const selectProvinceById = useCallback(
    (id: string) => {
      const p = provinces.find((x) => x.id === id);
      if (p) onSelectProvince(p);
    },
    [provinces, onSelectProvince],
  );

  const handleDragStart = useCallback(
    (unit: UnitState, pointerId: number, shiftKey: boolean) => {
      setDrag({ unitId: unit.id, pointerId, shiftHeld: shiftKey });
      onSelectUnit(unit);
    },
    [onSelectUnit, setDrag],
  );

  const handleLayerPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (!layerRef.current || !matchState || e.button !== 0) return;
      const rect = layerRef.current.getBoundingClientRect();
      const { x, y } = clientToWorld(e.clientX, e.clientY, rect, mapW, mapH);
      const unit = pickOwnedUnit(matchState.units, ownSlot, x, y);
      if (unit) {
        pendingRef.current = {
          unit,
          clientX: e.clientX,
          clientY: e.clientY,
          pointerId: e.pointerId,
          shiftKey: e.shiftKey,
        };
        return;
      }
      pendingRef.current = null;
      const provId = pickProvinceId(e.clientX, e.clientY, layerRef.current);
      if (provId) {
        selectProvinceById(provId);
        return;
      }
      setSelection(null);
    },
    [matchState, ownSlot, mapW, mapH, selectProvinceById, setSelection],
  );

  const handleLayerPointerMove = useCallback(
    (e: React.PointerEvent) => {
      const pending = pendingRef.current;
      if (pending && pending.pointerId === e.pointerId) {
        if (pointerMovedEnough(pending.clientX, pending.clientY, e.clientX, e.clientY)) {
          pendingRef.current = null;
          handleDragStart(pending.unit, pending.pointerId, pending.shiftKey);
        }
        return;
      }
      if (!layerRef.current) return;
      const provId = pickProvinceId(e.clientX, e.clientY, layerRef.current);
      setHoveredId(provId);
    },
    [handleDragStart],
  );

  const handleLayerPointerUp = useCallback(
    (e: React.PointerEvent) => {
      const pending = pendingRef.current;
      if (!pending || pending.pointerId !== e.pointerId) return;
      pendingRef.current = null;
      if (!pointerMovedEnough(pending.clientX, pending.clientY, e.clientX, e.clientY)) {
        onSelectUnit(pending.unit);
      }
    },
    [onSelectUnit],
  );

  const handleLayerPointerLeave = useCallback(() => {
    setHoveredId(null);
    pendingRef.current = null;
  }, []);

  const hoveredProvince = useMemo(
    () => provinces.find((p) => p.id === hoveredId),
    [provinces, hoveredId],
  );

  if (!matchState) {
    return <div className="map-loading">Awaiting battlefield data…</div>;
  }

  if (loading) return <div className="map-loading">Loading theater map…</div>;
  if (error || !assets) {
    return <div className="map-error">{error ?? "Map unavailable"}</div>;
  }

  return (
    <div className="map-stage">
      <TransformWrapper minScale={0.35} maxScale={8} wheel={{ step: 0.08 }}>
        <TransformComponent wrapperClass="map-transform-wrapper" contentClass="map-transform-content">
          <div
            ref={layerRef}
            className="map-motion-layer"
            style={{ width: mapW, height: mapH }}
            onPointerDown={handleLayerPointerDown}
            onPointerMove={handleLayerPointerMove}
            onPointerUp={handleLayerPointerUp}
            onPointerLeave={handleLayerPointerLeave}
          >
            <SvgProvinceMap
              assets={assets}
              provinces={provinces}
              players={players}
              diplomacy={matchState.diplomacy}
              pacts={matchState.pacts}
              ownSlot={ownSlot}
              hoveredProvinceId={hoveredId}
              selectedProvinceId={selectedProvinceId}
            />
            <PathOverlay
              width={mapW}
              height={mapH}
              dragPreview={dragPreview}
              selectedUnit={selectedUnit}
            />
            <UnitOverlay
              selectedUnitID={selectedUnitID}
              draggingUnitId={drag?.unitId ?? null}
              width={mapW}
              height={mapH}
            />
          </div>
        </TransformComponent>
      </TransformWrapper>
      {hoveredProvince && (
        <div
          className="map-tooltip-floating"
          style={{ left: "50%", bottom: "12%", transform: "translateX(-50%)" }}
        >
          <strong>{hoveredProvince.id}</strong>
          {hoveredProvince.capital && " · Capital"}
          {hoveredProvince.owner_id && (
            <span className="muted"> · {hoveredProvince.owner_id}</span>
          )}
        </div>
      )}
    </div>
  );
}
