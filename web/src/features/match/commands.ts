import { idem } from "../../lib/idem";
import type { Target } from "../../lib/pathfind";
import { nowISO } from "../../lib/time";
import type { ClientCommand } from "../../types/wire";
import type { GameSocket } from "./socket";

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

export function moveToTarget(
  socket: GameSocket,
  unitID: string,
  target: Target,
  options?: { queue?: boolean },
) {
  const args: Record<string, string> = {
    target_kind: target.kind,
    target_x: String(target.x),
    target_y: String(target.y),
  };
  if (target.kind === "node" && target.province) {
    args.target_province = target.province;
  }
  if (target.kind === "edge") {
    if (target.edgeFrom) args.edge_from = target.edgeFrom;
    if (target.edgeTo) args.edge_to = target.edgeTo;
    if (target.t != null) args.edge_t = String(target.t);
  }
  if (options?.queue) {
    args.queue = "true";
  }
  const cmd: ClientCommand = {
    kind: "move",
    unit_id: unitID,
    issued_at: nowISO(),
    idempotency_key: idem(),
    args,
  };
  socket.sendCommand(cmd);
}

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

export function recruitCommand(
  socket: GameSocket,
  provinceId: string,
  unitType: string,
) {
  const cmd: ClientCommand = {
    kind: "recruit",
    from: provinceId,
    issued_at: nowISO(),
    idempotency_key: idem(),
    args: { type: unitType },
  };
  socket.sendCommand(cmd);
}

export function constructCommand(
  socket: GameSocket,
  provinceId: string,
  buildingType: string,
) {
  const cmd: ClientCommand = {
    kind: "construct",
    from: provinceId,
    issued_at: nowISO(),
    idempotency_key: idem(),
    args: { type: buildingType },
  };
  socket.sendCommand(cmd);
}
