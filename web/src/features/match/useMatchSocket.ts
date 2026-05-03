import { useEffect, useRef, useState } from "react";
import { authApi, matchesApi } from "../../api";
import { useAppStore } from "../../app/store";
import type { MatchView } from "../../types/api";
import { GameSocket } from "./socket";

interface MatchSocketState {
  info: MatchView | null;
  err: string | null;
  socket: GameSocket | null;
}

// useMatchSocket loads the match info, exchanges a WS ticket and opens
// the GameSocket. It also tears the connection down on unmount.
export function useMatchSocket(matchID: string): MatchSocketState {
  const [info, setInfo] = useState<MatchView | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const socketRef = useRef<GameSocket | null>(null);
  const setMatchState = useAppStore((s) => s.setMatchState);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const fresh = await matchesApi.get(matchID);
        if (!alive) return;
        setInfo(fresh);
        const tk = await authApi.wsTicket();
        const socket = new GameSocket(tk.ticket, matchID);
        await socket.connect();
        if (!alive) {
          socket.close();
          return;
        }
        socketRef.current = socket;
      } catch (e) {
        if (alive) setErr(String(e));
      }
    })();
    return () => {
      alive = false;
      socketRef.current?.close();
      socketRef.current = null;
      setMatchState(null);
    };
  }, [matchID, setMatchState]);

  return { info, err, socket: socketRef.current };
}
