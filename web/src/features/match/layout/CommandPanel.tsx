import { Button, Panel, ResourceChip } from "../../../components/ui";
import { useAppStore } from "../../../app/store";
import { buildGraph, snapTarget } from "../../../lib/pathfind";
import type { MapDef } from "../../../types/api";
import type { GameSocket } from "../socket";
import {
  BUILDINGS,
  RECRUITABLE_UNITS,
  canAfford,
  formatCost,
} from "../unitCatalog";
import { constructCommand, recruitCommand } from "../commands";

interface CommandPanelProps {
  mapDef: MapDef | undefined;
  ownSlot: string | undefined;
  socket: GameSocket | null;
  onMoveToSelected: () => void;
}

function unitsInProvince(
  mapDef: MapDef | undefined,
  provinceId: string,
  units: { id: string; type: string; owner_id: string; x: number; y: number; hp: number }[],
) {
  if (!mapDef) return [];
  const graph = buildGraph(mapDef);
  return units.filter((u) => {
    const t = snapTarget(graph, u.x, u.y);
    return t.kind === "node" && t.province === provinceId;
  });
}

export function CommandPanel({
  mapDef,
  ownSlot,
  socket,
  onMoveToSelected,
}: CommandPanelProps) {
  const selection = useAppStore((s) => s.selection);
  const moveModeUnitId = useAppStore((s) => s.moveModeUnitId);
  const matchState = useAppStore((s) => s.matchState);
  const setMoveModeUnitId = useAppStore((s) => s.setMoveModeUnitId);

  if (!selection) {
    return (
      <Panel title="Command" className="command-panel command-panel--empty">
        <p className="muted">Select a province or unit on the map.</p>
      </Panel>
    );
  }

  if (selection.kind === "province") {
    const p = selection.province;
    const provinceMeta = mapDef?.provinces.find((x) => x.id === p.id);
    const buildings =
      matchState?.buildings?.filter((b) => b.province === p.id) ?? [];
    const garrison = unitsInProvince(mapDef, p.id, matchState?.units ?? []);
    const pool = ownSlot ? matchState?.resources?.[ownSlot] : undefined;
    const isOwned = ownSlot != null && p.owner_id === ownSlot;
    const provinceRecruit = matchState?.queues?.recruits?.find(
      (r) => r.province === p.id,
    );
    const provinceBuild = matchState?.queues?.constructions?.find(
      (c) => c.province === p.id,
    );
    const hasFactory = buildings.some((b) => b.type === "factory");
    const provinceBusy = Boolean(provinceRecruit || provinceBuild);
    const recruitDisabledReason = !isOwned
      ? "Enemy territory"
      : provinceBusy
        ? "Province busy"
        : null;

    return (
      <Panel title={provinceMeta?.name ?? p.id} className="command-panel">
        <dl className="command-dl">
          <dt>Province</dt>
          <dd>{p.id}</dd>
          <dt>Owner</dt>
          <dd>{p.owner_id ?? "—"}</dd>
          {p.capital && (
            <>
              <dt>Type</dt>
              <dd>Capital</dd>
            </>
          )}
        </dl>

        {garrison.length > 0 && (
          <>
            <h4 className="command-subhead">Garrison</h4>
            <ul className="command-list">
              {garrison.map((u) => (
                <li key={u.id}>
                  {u.type} · {Math.round(u.hp)} HP
                </li>
              ))}
            </ul>
          </>
        )}

        {isOwned && pool && (
          <>
            <h4 className="command-subhead">Your resources</h4>
            <div className="command-resources">
              <ResourceChip label="Manpower" amount={pool.manpower} />
              <ResourceChip label="Food" amount={pool.food} warn={pool.food < 10} />
              <ResourceChip label="Iron" amount={pool.iron} />
            </div>
          </>
        )}

        {(provinceRecruit || provinceBuild) && (
          <>
            <h4 className="command-subhead">Production</h4>
            <ul className="command-list">
              {provinceRecruit && (
                <li>
                  Recruiting {provinceRecruit.unit_type} (completes{" "}
                  {new Date(provinceRecruit.completes_at).toLocaleTimeString()})
                </li>
              )}
              {provinceBuild && (
                <li>
                  Building {provinceBuild.building_type} (completes{" "}
                  {new Date(provinceBuild.completes_at).toLocaleTimeString()})
                </li>
              )}
            </ul>
          </>
        )}

        {buildings.length > 0 && (
          <>
            <h4 className="command-subhead">Buildings</h4>
            <ul className="command-list">
              {buildings.map((b, i) => (
                <li key={i}>
                  {b.type} ({b.owner})
                </li>
              ))}
            </ul>
          </>
        )}

        {isOwned && socket && (
          <>
            <h4 className="command-subhead">Recruit</h4>
            {recruitDisabledReason && (
              <p className="command-hint muted">{recruitDisabledReason}</p>
            )}
            <div className="command-actions">
              {RECRUITABLE_UNITS.map((unit) => {
                const needsBlock = Boolean(unit.needsBuilding && !hasFactory);
                const afford = canAfford(pool, unit.cost);
                const disabled =
                  Boolean(recruitDisabledReason) || needsBlock || !afford;
                let hint = formatCost(unit.cost);
                if (needsBlock) hint += " · needs factory";
                else if (!afford) hint += " · insufficient resources";
                return (
                  <Button
                    key={unit.id}
                    variant="secondary"
                    size="sm"
                    disabled={disabled}
                    onClick={() => recruitCommand(socket, p.id, unit.id)}
                    title={hint}
                  >
                    {unit.label}
                  </Button>
                );
              })}
            </div>

            {!hasFactory && !provinceBusy && (
              <>
                <h4 className="command-subhead">Construct</h4>
                <div className="command-actions">
                  {BUILDINGS.map((b) => {
                    const afford = canAfford(pool, b.cost);
                    const disabled = provinceBusy || !afford;
                    return (
                      <Button
                        key={b.id}
                        variant="secondary"
                        size="sm"
                        disabled={disabled}
                        onClick={() => constructCommand(socket, p.id, b.id)}
                        title={
                          afford
                            ? formatCost(b.cost)
                            : `${formatCost(b.cost)} · insufficient resources`
                        }
                      >
                        {b.label}
                      </Button>
                    );
                  })}
                </div>
              </>
            )}
          </>
        )}

        {moveModeUnitId && (
          <Button variant="primary" size="sm" onClick={onMoveToSelected}>
            Move unit here
          </Button>
        )}
      </Panel>
    );
  }

  const u = selection.unit;
  return (
    <Panel title={`Unit ${u.type}`} className="command-panel">
      <dl className="command-dl">
        <dt>HP</dt>
        <dd>{Math.round(u.hp)}</dd>
        <dt>Owner</dt>
        <dd>{u.owner_id}</dd>
        <dt>Position</dt>
        <dd>
          ({Math.round(u.x)}, {Math.round(u.y)})
        </dd>
      </dl>
      {ownSlot && u.owner_id === ownSlot && (
        <Button
          variant={moveModeUnitId === u.id ? "secondary" : "primary"}
          size="sm"
          onClick={() => setMoveModeUnitId(moveModeUnitId === u.id ? null : u.id)}
        >
          {moveModeUnitId === u.id ? "Cancel move" : "Move unit"}
        </Button>
      )}
    </Panel>
  );
}
