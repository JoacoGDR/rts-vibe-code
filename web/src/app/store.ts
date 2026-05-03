import { create } from "zustand";
import type { Session, User } from "../types/api";
import type { ChatInbound, MatchState, ServerEvent } from "../types/wire";

interface AppStore {
  session: Session | null;
  matchState: MatchState | null;
  events: ServerEvent[];
  chat: ChatInbound[];
  serverOffsetMs: number;
  setSession(s: Session | null): void;
  setMatchState(s: MatchState | null): void;
  pushEvent(e: ServerEvent): void;
  pushChat(m: ChatInbound): void;
  resetChat(): void;
  setOffset(ms: number): void;
}

export const useAppStore = create<AppStore>((set) => ({
  session: null,
  matchState: null,
  events: [],
  chat: [],
  serverOffsetMs: 0,
  setSession: (s) => set({ session: s }),
  setMatchState: (s) => set({ matchState: s }),
  pushEvent: (e) => set((prev) => ({ events: [...prev.events.slice(-49), e] })),
  pushChat: (m) =>
    set((prev) => {
      if (prev.chat.some((existing) => existing.id === m.id)) {
        return prev;
      }
      return { chat: [...prev.chat.slice(-199), m] };
    }),
  resetChat: () => set({ chat: [] }),
  setOffset: (ms) => set({ serverOffsetMs: ms }),
}));

export function selectMe(): User | undefined {
  return useAppStore.getState().session?.user;
}
