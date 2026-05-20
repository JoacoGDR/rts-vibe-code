import { Button } from "../../../components/ui";
import { Badge } from "../../../components/ui";
import type { MatchView } from "../../../types/api";

interface MatchHeaderProps {
  info: MatchView | null;
  ownSlot: string | undefined;
  onLeave: () => void;
  onStart?: () => void;
  onHandoff?: () => void;
}

export function MatchHeader({ info, ownSlot, onLeave, onStart, onHandoff }: MatchHeaderProps) {
  const statusVariant =
    info?.status === "active" || info?.status === "starting"
      ? "active"
      : info?.status === "ended" || info?.status === "abandoned"
        ? "ended"
        : "waiting";

  return (
    <header className="match-header">
      <Button variant="ghost" size="sm" onClick={onLeave}>
        ← War room
      </Button>
      <h2 className="match-header__title">{info?.name ?? "Match"}</h2>
      {info && <Badge variant={statusVariant}>{info.status}</Badge>}
      <span className="muted match-header__slot">
        Command: <strong>{ownSlot ?? "—"}</strong>
      </span>
      <div className="match-header__actions">
        {info?.status === "waiting" && info.players.length >= 2 && onStart && (
          <Button variant="primary" size="sm" onClick={onStart}>
            Commence
          </Button>
        )}
        {(info?.status === "active" || info?.status === "starting") && ownSlot && onHandoff && (
          <Button variant="secondary" size="sm" onClick={onHandoff}>
            Hand to AI
          </Button>
        )}
      </div>
    </header>
  );
}
