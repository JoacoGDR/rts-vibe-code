import { idem } from "../../lib/idem";
import { nowISO } from "../../lib/time";
import type { ClientCommand } from "../../types/wire";
import type { GameSocket } from "./socket";

// moveCommand builds a `move` ClientCommand and dispatches it via the
// supplied socket. Keeping the wire shape here means components don't
// reach into the wire types directly.
export function moveCommand(socket: GameSocket, unitID: string, from: string, to: string) {
  const cmd: ClientCommand = {
    kind: "move",
    unit_id: unitID,
    from,
    to,
    issued_at: nowISO(),
    idempotency_key: idem(),
  };
  socket.sendCommand(cmd);
}

// diplomacyCommand wraps the seven diplomacy commands behind a single
// helper. The engine consumes the target slot through args.target_slot
// (see internal/domain/cmddom/diplomacy_helpers.go).
export type DiplomacyKind =
  | "declare_war"
  | "propose_peace"
  | "propose_alliance"
  | "accept_peace"
  | "accept_alliance"
  | "share_map"
  | "revoke_share_map"
  | "right_of_way_grant"
  | "right_of_way_revoke";

export function diplomacyCommand(socket: GameSocket, kind: DiplomacyKind, targetSlot: string) {
  const cmd: ClientCommand = {
    kind,
    issued_at: nowISO(),
    idempotency_key: idem(),
    args: { target_slot: targetSlot },
  };
  socket.sendCommand(cmd);
}
