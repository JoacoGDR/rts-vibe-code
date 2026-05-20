import type { ReactNode } from "react";
import { MatchHeader } from "./MatchHeader";
import { ResourceBar } from "./ResourceBar";
import { CommandPanel } from "./CommandPanel";
import { NewsTicker } from "./NewsTicker";
import type { MatchView, MapDef } from "../../../types/api";
import type { MatchState, ServerEvent } from "../../../types/wire";
import type { GameSocket } from "../socket";

interface MatchShellProps {
  info: MatchView | null;
  matchState: MatchState | null;
  events: ServerEvent[];
  mapDef: MapDef | undefined;
  ownSlot: string | undefined;
  mapStage: ReactNode;
  sidebar: ReactNode;
  onLeave: () => void;
  onStart?: () => void;
  onHandoff?: () => void;
  onMoveToSelected: () => void;
  socket: GameSocket | null;
  postGame?: ReactNode;
}

export function MatchShell({
  info,
  matchState,
  events,
  mapDef,
  ownSlot,
  mapStage,
  sidebar,
  onLeave,
  onStart,
  onHandoff,
  onMoveToSelected,
  socket,
  postGame,
}: MatchShellProps) {
  return (
    <div className="match-shell">
      <MatchHeader
        info={info}
        ownSlot={ownSlot}
        onLeave={onLeave}
        onStart={onStart}
        onHandoff={onHandoff}
      />
      <ResourceBar matchState={matchState} ownSlot={ownSlot} />
      <NewsTicker events={events} />
      <div className="match-shell__body">
        <div className="match-shell__map">{mapStage}</div>
        <aside className="match-shell__aside">
          <CommandPanel
            mapDef={mapDef}
            ownSlot={ownSlot}
            socket={socket}
            onMoveToSelected={onMoveToSelected}
          />
          {sidebar}
        </aside>
      </div>
      {postGame}
    </div>
  );
}
