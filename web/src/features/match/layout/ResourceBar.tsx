import { ResourceChip } from "../../../components/ui";
import type { MatchState } from "../../../types/wire";

interface ResourceBarProps {
  matchState: MatchState | null;
  ownSlot: string | undefined;
}

export function ResourceBar({ matchState, ownSlot }: ResourceBarProps) {
  const pool = ownSlot ? matchState?.resources?.[ownSlot] : undefined;
  const tick = matchState?.tick ?? 0;

  return (
    <header className="resource-bar">
      <ResourceChip label="Manpower" amount={pool?.manpower ?? 0} />
      <ResourceChip label="Food" amount={pool?.food ?? 0} warn={(pool?.food ?? 0) < 10} />
      <ResourceChip label="Iron" amount={pool?.iron ?? 0} />
      <span className="resource-bar__meta data-label">
        Tick {tick}
        {matchState?.game_time && ` · ${matchState.game_time}`}
      </span>
    </header>
  );
}
