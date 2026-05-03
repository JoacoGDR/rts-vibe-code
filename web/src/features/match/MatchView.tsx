import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { matchesApi } from "../../api";
import { useAppStore } from "../../app/store";
import type { ProvinceState, UnitState } from "../../types/wire";
import { ChatPanel } from "../chat/ChatPanel";
import { DiplomacyPanel } from "../diplomacy/DiplomacyPanel";
import { MapView } from "./MapView";
import { moveCommand } from "./commands";
import { useMatchSocket } from "./useMatchSocket";

export function MatchView() {
  const { matchID } = useParams<{ matchID: string }>();
  const navigate = useNavigate();
  const session = useAppStore((s) => s.session);
  const matchState = useAppStore((s) => s.matchState);
  const events = useAppStore((s) => s.events);
  const { info, err, socket } = useMatchSocket(matchID ?? "");
  const [selected, setSelected] = useState<UnitState | null>(null);
  const [startErr, setStartErr] = useState<string | null>(null);

  const ownSlot = useMemo(() => {
    if (!info || !session) return undefined;
    return info.players.find((p) => p.user_id === session.user.id)?.slot;
  }, [info, session]);

  async function start() {
    if (!matchID) return;
    try {
      await matchesApi.start(matchID);
    } catch (e) {
      setStartErr(String(e));
    }
  }

  function selectUnit(u: UnitState) {
    if (ownSlot && u.owner_id === ownSlot) {
      setSelected(u);
    }
  }

  function selectProvince(p: ProvinceState) {
    if (!selected || !socket) return;
    moveCommand(socket, selected.id, selected.origin ?? "", p.id);
    setSelected(null);
  }

  function leave() {
    navigate("/lobby");
  }

  const winner = matchState?.players.find((p) => p.id === matchState.players[0]?.id);
  const recentEnd = events.find((e) => e.kind === "match_ended");

  return (
    <div className="match-view">
      <header>
        <button onClick={leave}>Leave</button>
        <h2>{info?.name ?? "Match"}</h2>
        <span className="muted">
          {info?.status} &middot; you play <strong>{ownSlot ?? "—"}</strong>
        </span>
        {info?.status === "waiting" && info.players.length >= 2 && (
          <button onClick={start}>Start</button>
        )}
      </header>

      {info?.status === "waiting" && (
        <p className="muted">Waiting for players… ({info.players.length} joined)</p>
      )}

      <div className="board">
        <MapView
          ownSlot={ownSlot}
          onSelectUnit={selectUnit}
          onSelectProvince={selectProvince}
          selectedUnitID={selected?.id ?? null}
        />
        <aside>
          <h3>Players</h3>
          <ul>
            {info?.players.map((p) => (
              <li key={p.user_id} style={{ color: p.color }}>
                {p.slot} {p.user_id === session?.user.id ? "(you)" : ""}
              </li>
            ))}
          </ul>
          <h3>Recent events</h3>
          <ul className="events">
            {events.slice(-12).map((e, i) => (
              <li key={i}>
                <code>{e.kind}</code> {e.unit_id ? <span>unit {e.unit_id.slice(0, 6)}</span> : null}{" "}
                {e.province ? <span>@ {e.province}</span> : null}
              </li>
            ))}
          </ul>
          <DiplomacyPanel ownSlot={ownSlot} socket={socket} />
          <ChatPanel matchID={matchID ?? ""} socket={socket} ownSlot={ownSlot} />
        </aside>
      </div>

      {recentEnd && (
        <div className="banner">
          <h2>Match ended</h2>
          <p>Winner: {String(recentEnd.extra?.["slot"] ?? recentEnd.extra ?? winner?.id ?? "?")}</p>
          <button onClick={leave}>Back to lobby</button>
        </div>
      )}

      {(err || startErr) && <p className="err">{err ?? startErr}</p>}
    </div>
  );
}
