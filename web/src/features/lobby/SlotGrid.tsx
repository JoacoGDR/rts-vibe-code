import type { MapDef, MatchView } from "../../types/api";

interface SlotGridProps {
  map: MapDef | undefined;
  match: MatchView | null;
}

export function SlotGrid({ map, match }: SlotGridProps) {
  if (!map || !match) return null;
  if (match.status !== "waiting") return null;

  return (
    <div className="slot-grid">
      <h4 className="war-room-title">Command slots</h4>
      <ul>
        {map.slots.map((slot) => {
          const player = match.players.find((p) => p.slot === slot.id);
          return (
            <li key={slot.id} style={{ borderLeftColor: slot.color }}>
              <span className="slot-grid__id">{slot.id}</span>
              {player ? (
                <span>
                  {player.controlled_by_ai ? "AI" : "Human"} · {player.user_id.slice(0, 8)}…
                </span>
              ) : (
                <span className="muted">Open</span>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
