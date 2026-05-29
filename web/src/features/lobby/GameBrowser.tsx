import { Badge, Button } from "../../components/ui";
import type { MatchView } from "../../types/api";

export type LobbyFilter = "all" | "open" | "mine" | "active" | "ended";

interface GameBrowserProps {
  matches: MatchView[];
  filter: LobbyFilter;
  onFilterChange: (f: LobbyFilter) => void;
  selectedMatchId: string | null;
  onSelectMatch: (m: MatchView) => void;
  onEnter: (id: string) => void;
  onJoin: (id: string) => void;
  onLeave: (id: string) => void;
  busy: boolean;
  userId?: string;
}

const FILTERS: LobbyFilter[] = ["all", "open", "mine", "active", "ended"];

function statusVariant(status: MatchView["status"]) {
  if (status === "active" || status === "starting") return "active" as const;
  if (status === "ended" || status === "abandoned") return "ended" as const;
  return "waiting" as const;
}

export function GameBrowser({
  matches,
  filter,
  onFilterChange,
  selectedMatchId,
  onSelectMatch,
  onEnter,
  onJoin,
  onLeave,
  busy,
  userId,
}: GameBrowserProps) {
  return (
    <section className="game-browser">
      <h2 className="war-room-title">Theater roster</h2>
      <div className="game-browser__filters">
        {FILTERS.map((f) => (
          <button
            key={f}
            type="button"
            className={`ui-tab ${filter === f ? "ui-tab--active" : ""}`}
            onClick={() => onFilterChange(f)}
          >
            {f}
          </button>
        ))}
      </div>
      {matches.length === 0 ? (
        <p className="muted">No matches in this filter.</p>
      ) : (
        <ul className="game-browser__list">
          {matches.map((m) => {
            const isMember = userId ? m.players.some((p) => p.user_id === userId) : false;
            return (
              <li
                key={m.id}
                className={selectedMatchId === m.id ? "selected" : ""}
                onClick={() => onSelectMatch(m)}
              >
                <div className="game-browser__row">
                  <strong>{m.name}</strong>
                  <Badge variant={statusVariant(m.status)}>{m.status}</Badge>
                </div>
                <span className="muted game-browser__meta">
                  {m.map_id} · {m.players.length} commander(s)
                </span>
                <div className="game-browser__actions">
                  <Button variant="ghost" size="sm" onClick={() => onEnter(m.id)}>
                    Enter
                  </Button>
                  {m.status === "waiting" && !isMember && (
                    <Button variant="primary" size="sm" onClick={() => onJoin(m.id)} disabled={busy}>
                      Join
                    </Button>
                  )}
                  {m.status === "waiting" && isMember && (
                    <Button variant="secondary" size="sm" onClick={() => onLeave(m.id)} disabled={busy}>
                      Leave
                    </Button>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
