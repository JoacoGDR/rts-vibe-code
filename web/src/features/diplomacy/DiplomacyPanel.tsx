import { useMemo } from "react";
import { useAppStore } from "../../app/store";
import type { DiplomacyKind } from "../match/commands";
import { diplomacyCommand } from "../match/commands";
import type { GameSocket } from "../match/socket";
import type { PactState, Stance, TreatyState } from "../../types/wire";

interface Props {
  ownSlot: string | undefined;
  socket: GameSocket | null;
}

// DiplomacyPanel renders the per-slot stance grid plus the per-slot
// action buttons. We keep the styling deliberately spartan so the
// component fits next to the existing aside without crowding the map.
export function DiplomacyPanel({ ownSlot, socket }: Props) {
  const matchState = useAppStore((s) => s.matchState);
  const others = useMemo(() => {
    if (!matchState || !ownSlot) return [];
    return matchState.players.filter((p) => p.id !== ownSlot);
  }, [matchState, ownSlot]);

  if (!matchState || !ownSlot || !socket) {
    return null;
  }

  return (
    <section className="diplomacy">
      <h3>Diplomacy</h3>
      <ul>
        {others.map((p) => (
          <li key={p.id} style={{ borderColor: p.color }}>
            <strong>{p.id}</strong>
            <span className="stance">{stanceFor(ownSlot, p.id, matchState.diplomacy ?? [])}</span>
            <PendingHint ownSlot={ownSlot} other={p.id} treaties={matchState.diplomacy ?? []} />
            <DiplomacyActions
              ownSlot={ownSlot}
              other={p.id}
              treaties={matchState.diplomacy ?? []}
              pacts={matchState.pacts ?? []}
              onAct={(kind) => diplomacyCommand(socket, kind, p.id)}
            />
          </li>
        ))}
      </ul>
    </section>
  );
}

function PendingHint({
  ownSlot,
  other,
  treaties,
}: {
  ownSlot: string;
  other: string;
  treaties: TreatyState[];
}) {
  const t = treatyBetween(ownSlot, other, treaties);
  if (!t || !t.pending) return null;
  const label = t.pending_from === ownSlot ? `awaiting ${other}` : `${other} offers ${t.pending}`;
  return <span className="pending"> ({label})</span>;
}

function DiplomacyActions({
  ownSlot,
  other,
  treaties,
  pacts,
  onAct,
}: {
  ownSlot: string;
  other: string;
  treaties: TreatyState[];
  pacts: PactState[];
  onAct(kind: DiplomacyKind): void;
}) {
  const stance = stanceFor(ownSlot, other, treaties);
  const treaty = treatyBetween(ownSlot, other, treaties);
  const pendingFromOther = treaty?.pending && treaty.pending_from === other;
  const sharingMap = pacts.some(
    (p) => p.from === ownSlot && p.to === other && p.kind === "share_map",
  );
  const grantingROW = pacts.some(
    (p) => p.from === ownSlot && p.to === other && p.kind === "right_of_way",
  );
  return (
    <div className="actions">
      {stance !== "war" && <button onClick={() => onAct("declare_war")}>Declare war</button>}
      {stance === "war" && !pendingFromOther && (
        <>
          <button onClick={() => onAct("propose_peace")}>Propose peace</button>
        </>
      )}
      {stance === "peace" && (
        <button onClick={() => onAct("propose_alliance")}>Propose alliance</button>
      )}
      {pendingFromOther && treaty?.pending === "peace" && (
        <button onClick={() => onAct("accept_peace")}>Accept peace</button>
      )}
      {pendingFromOther && treaty?.pending === "alliance" && (
        <button onClick={() => onAct("accept_alliance")}>Accept alliance</button>
      )}
      {stance !== "war" && (
        <button onClick={() => onAct(sharingMap ? "revoke_share_map" : "share_map")}>
          {sharingMap ? "Revoke share-map" : "Share map"}
        </button>
      )}
      {stance !== "war" && (
        <button onClick={() => onAct(grantingROW ? "right_of_way_revoke" : "right_of_way_grant")}>
          {grantingROW ? "Revoke right-of-way" : "Grant right-of-way"}
        </button>
      )}
    </div>
  );
}

function treatyBetween(a: string, b: string, treaties: TreatyState[]): TreatyState | undefined {
  return treaties.find(
    (t) => (t.slot_a === a && t.slot_b === b) || (t.slot_a === b && t.slot_b === a),
  );
}

function stanceFor(a: string, b: string, treaties: TreatyState[]): Stance {
  return treatyBetween(a, b, treaties)?.stance ?? "war";
}
