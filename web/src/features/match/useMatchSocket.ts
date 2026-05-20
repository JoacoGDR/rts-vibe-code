import { useEffect, useState } from "react";
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
  const [socket, setSocket] = useState<GameSocket | null>(null);
  const clearMatchUI = useAppStore((s) => s.clearMatchUI);

  useEffect(() => {
    let alive = true;
    setSocket(null);
    setErr(null);

    (async () => {
      try {
        const fresh = await matchesApi.get(matchID);
        if (!alive) return;
        setInfo(fresh);
        const tk = await authApi.wsTicket();
        const gameSocket = new GameSocket(tk.ticket, matchID);
        await gameSocket.connect();
        if (!alive) {
          gameSocket.close();
          return;
        }
        setSocket(gameSocket);
      } catch (e) {
        if (alive) setErr(String(e));
      }
    })();

    return () => {
      alive = false;
      setSocket((prev) => {
        prev?.close();
        return null;
      });
      clearMatchUI();
    };
  }, [matchID, clearMatchUI]);

  return { info, err, socket };
}
