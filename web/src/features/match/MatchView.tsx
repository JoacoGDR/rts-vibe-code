import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { mapsApi, matchesApi } from "../../api";
import { useAppStore } from "../../app/store";
import { useNationTheme } from "../../hooks/useNationTheme";
import type { MapDef, MatchStatsView } from "../../types/api";
import type { ProvinceState, UnitState } from "../../types/wire";
import { ChatPanel } from "../chat/ChatPanel";
import { DiplomacyPanel } from "../diplomacy/DiplomacyPanel";
import { Button } from "../../components/ui";
import { MapStage } from "../map/MapStage";
import { moveCommand, moveToTarget } from "./commands";
import type { Target } from "../../lib/pathfind";
import { MatchShell } from "./layout/MatchShell";
import { useMatchSocket } from "./useMatchSocket";

const STATE_WATCHDOG_MS = 8000;

export function MatchView() {
  const { matchID } = useParams<{ matchID: string }>();
  const navigate = useNavigate();
  const session = useAppStore((s) => s.session);
  const matchState = useAppStore((s) => s.matchState);
  const events = useAppStore((s) => s.events);
  const selection = useAppStore((s) => s.selection);
  const moveModeUnitId = useAppStore((s) => s.moveModeUnitId);
  const setSelection = useAppStore((s) => s.setSelection);
  const setMoveModeUnitId = useAppStore((s) => s.setMoveModeUnitId);
  const { info, err, socket } = useMatchSocket(matchID ?? "");
  const [mapDef, setMapDef] = useState<MapDef | undefined>();
  const [startErr, setStartErr] = useState<string | null>(null);
  const [stats, setStats] = useState<MatchStatsView | null>(null);
  const [stateStale, setStateStale] = useState(false);

  const awaitingState =
    info?.status === "starting" || info?.status === "active";

  const ownSlot = useMemo(() => {
    if (!info || !session) return undefined;
    return info.players.find((p) => p.user_id === session.user.id)?.slot;
  }, [info, session]);

  const ownColor = useMemo(() => {
    if (!info || !ownSlot) return undefined;
    return info.players.find((p) => p.slot === ownSlot)?.color;
  }, [info, ownSlot]);

  useNationTheme(ownColor);

  useEffect(() => {
    if (!info?.map_id) return;
    void mapsApi.list().then((maps) => setMapDef(maps.find((m) => m.id === info.map_id)));
  }, [info?.map_id]);

  const recentEnd = events.find((e) => e.kind === "match_ended");
  const showPostGame =
    info?.status === "ended" ||
    info?.status === "abandoned" ||
    Boolean(recentEnd);

  useEffect(() => {
    if (!matchID || !showPostGame) return;
    void matchesApi.stats(matchID).then(setStats).catch(() => setStats(null));
  }, [matchID, showPostGame]);

  useEffect(() => {
    setStateStale(false);
    if (!awaitingState || matchState) return;
    const timer = window.setTimeout(() => setStateStale(true), STATE_WATCHDOG_MS);
    return () => window.clearTimeout(timer);
  }, [awaitingState, matchState, matchID]);

  const starting = awaitingState && !matchState && !stateStale;
  const showStaleBanner = awaitingState && !matchState && stateStale;

  function requestStateResync() {
    socket?.requestResync();
    setStateStale(false);
    window.setTimeout(() => {
      if (!useAppStore.getState().matchState) setStateStale(true);
    }, STATE_WATCHDOG_MS);
  }

  const selectedUnit =
    selection?.kind === "unit"
      ? selection.unit
      : moveModeUnitId
        ? matchState?.units.find((u) => u.id === moveModeUnitId) ?? null
        : null;

  async function start() {
    if (!matchID) return;
    try {
      await matchesApi.start(matchID);
    } catch (e) {
      setStartErr(String(e));
    }
  }

  async function handoff() {
    if (!matchID) return;
    try {
      await matchesApi.handoff(matchID);
    } catch (e) {
      setStartErr(String(e));
    }
  }

  function selectUnit(u: UnitState) {
    if (ownSlot && u.owner_id === ownSlot) {
      setSelection({ kind: "unit", unit: u });
    }
  }

  function selectProvince(p: ProvinceState) {
    setSelection({ kind: "province", province: p });
    if (moveModeUnitId && socket) {
      const unit = matchState?.units.find((u) => u.id === moveModeUnitId);
      if (unit) {
        moveCommand(socket, unit.id, unit.origin ?? "", p.id);
        setMoveModeUnitId(null);
        setSelection(null);
      }
    }
  }

  function handleMoveToSelected() {
    if (selection?.kind !== "province" || !moveModeUnitId || !socket) return;
    const unit = matchState?.units.find((u) => u.id === moveModeUnitId);
    if (!unit) return;
    moveCommand(socket, unit.id, unit.origin ?? "", selection.province.id);
    setMoveModeUnitId(null);
    setSelection(null);
  }

  function handleCommitMove(unitId: string, target: Target, queue: boolean) {
    if (!socket) return;
    moveToTarget(socket, unitId, target, { queue });
    setMoveModeUnitId(null);
  }

  function leaveLobby() {
    navigate("/lobby");
  }

  const winnerLabel = recentEnd
    ? String(
        recentEnd.extra?.["coalition"] ??
          (recentEnd.extra?.["winners"] as string[] | undefined)?.join(", ") ??
          "?",
      )
    : info?.winner_user_id ?? "?";

  return (
    <>
      {starting && (
        <div className="overlay starting-overlay">
          <p>Connecting to battlefield… requesting state.</p>
        </div>
      )}

      {showStaleBanner && (
        <div className="stale-state-banner" role="alert">
          <p>
            Battlefield data not received. The simulation may have restarted — try requesting
            state again, or start a new match from the war room.
          </p>
          <div className="stale-state-banner__actions">
            <Button variant="primary" size="sm" onClick={requestStateResync} disabled={!socket}>
              Request resync
            </Button>
            <Button variant="secondary" size="sm" onClick={leaveLobby}>
              Back to war room
            </Button>
          </div>
        </div>
      )}

      <MatchShell
        info={info}
        matchState={matchState}
        events={events}
        mapDef={mapDef}
        ownSlot={ownSlot}
        onLeave={leaveLobby}
        onStart={info?.status === "waiting" ? start : undefined}
        onHandoff={ownSlot ? handoff : undefined}
        onMoveToSelected={handleMoveToSelected}
        socket={socket}
        mapStage={
          <MapStage
            mapDef={mapDef}
            ownSlot={ownSlot}
            selectedUnitID={selectedUnit?.id ?? null}
            onSelectUnit={selectUnit}
            onSelectProvince={selectProvince}
            onCommitMove={handleCommitMove}
          />
        }
        sidebar={
          <>
            <section className="match-players">
              <h3 className="war-room-title">Commanders</h3>
              <ul>
                {info?.players.map((p) => (
                  <li key={p.user_id} style={{ color: p.color }}>
                    {p.slot} {p.user_id === session?.user.id ? "(you)" : ""}
                    {p.controlled_by_ai ? " [AI]" : ""}
                  </li>
                ))}
              </ul>
            </section>
            <DiplomacyPanel ownSlot={ownSlot} socket={socket} />
            <ChatPanel matchID={matchID ?? ""} socket={socket} ownSlot={ownSlot} />
          </>
        }
        postGame={
          showPostGame ? (
            <div className="banner post-game">
              <h2>Match ended</h2>
              <p>Winner: {winnerLabel}</p>
              {stats && (
                <dl className="stats">
                  <dt>Duration</dt>
                  <dd>{stats.duration_sec}s</dd>
                  <dt>Units on field</dt>
                  <dd>{stats.total_units}</dd>
                  <dt>Capitals held</dt>
                  <dd>{stats.capitals_taken}</dd>
                </dl>
              )}
              {!stats && (
                <p className="muted">Stats will appear once the worker finalizes the match.</p>
              )}
              <button type="button" onClick={leaveLobby}>
                Back to war room
              </button>
            </div>
          ) : undefined
        }
      />

      {info?.status === "waiting" && (
        <p className="muted match-waiting">Waiting for commanders… ({info.players.length} joined)</p>
      )}

      {(err || startErr) && <p className="err match-err">{err ?? startErr}</p>}
    </>
  );
}
