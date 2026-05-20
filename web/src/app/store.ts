import { create } from "zustand";
import type { Session, User } from "../types/api";
import type { Leg, Target } from "../lib/pathfind";
import type { ChatInbound, MatchState, ProvinceState, ServerEvent, UnitState } from "../types/wire";

export interface DragState {
  unitId: string;
  pointerId: number;
  shiftHeld: boolean;
}

export interface DragPreview {
  target: Target;
  legs: Leg[];
  attackTerminus: boolean;
}

export type Selection =
  | { kind: "province"; province: ProvinceState }
  | { kind: "unit"; unit: UnitState }
  | null;

interface AppStore {
  session: Session | null;
  matchState: MatchState | null;
  events: ServerEvent[];
  chat: ChatInbound[];
  serverOffsetMs: number;
  selection: Selection;
  moveModeUnitId: string | null;
  drag: DragState | null;
  dragPreview: DragPreview | null;
  setSession(s: Session | null): void;
  setMatchState(s: MatchState | null): void;
  pushEvent(e: ServerEvent): void;
  pushChat(m: ChatInbound): void;
  resetChat(): void;
  setOffset(ms: number): void;
  setSelection(s: Selection): void;
  setMoveModeUnitId(id: string | null): void;
  setDrag(drag: DragState | null): void;
  setDragPreview(preview: DragPreview | null): void;
  clearMatchUI(): void;
}

export const useAppStore = create<AppStore>((set) => ({
  session: null,
  matchState: null,
  events: [],
  chat: [],
  serverOffsetMs: 0,
  selection: null,
  moveModeUnitId: null,
  drag: null,
  dragPreview: null,
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
  setSelection: (selection) => set({ selection }),
  setMoveModeUnitId: (moveModeUnitId) => set({ moveModeUnitId }),
  setDrag: (drag) => set({ drag }),
  setDragPreview: (dragPreview) => set({ dragPreview }),
  clearMatchUI: () =>
    set({
      selection: null,
      moveModeUnitId: null,
      drag: null,
      dragPreview: null,
      matchState: null,
      events: [],
    }),
}));

export function selectMe(): User | undefined {
  return useAppStore.getState().session?.user;
}
