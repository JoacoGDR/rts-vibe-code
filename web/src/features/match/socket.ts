import { useAppStore } from "../../app/store";
import type { ChatOutbound, ClientCommand, ClientEnvelope, ServerEnvelope } from "../../types/wire";

// GameSocket owns the websocket lifecycle for a single match. The match
// component drives it via the useMatchSocket hook below; everything else
// reads state from the zustand store.
export class GameSocket {
  private ws: WebSocket | null = null;
  private matchID: string;
  private ticket: string;
  private pingTimer: number | null = null;

  constructor(ticket: string, matchID: string) {
    this.ticket = ticket;
    this.matchID = matchID;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = new URL(`/ws`, location.origin);
      url.protocol = location.protocol === "https:" ? "wss:" : "ws:";
      url.searchParams.set("ticket", this.ticket);
      url.searchParams.set("match_id", this.matchID);
      const ws = new WebSocket(url.toString());
      this.ws = ws;
      ws.onopen = () => {
        this.send({ type: "hello", match_id: this.matchID });
        this.pingTimer = window.setInterval(() => {
          this.send({ type: "ping" });
        }, 20000);
        resolve();
      };
      ws.onerror = (e) => reject(e);
      ws.onclose = () => {
        if (this.pingTimer) window.clearInterval(this.pingTimer);
      };
      ws.onmessage = (msg) => this.handleMessage(msg);
    });
  }

  private handleMessage(msg: MessageEvent) {
    let env: ServerEnvelope;
    try {
      env = JSON.parse(msg.data);
    } catch {
      return;
    }
    if (env.sent_at) {
      const offset = new Date(env.sent_at).getTime() - Date.now();
      useAppStore.getState().setOffset(offset);
    }
    switch (env.type) {
      case "state":
        if (env.state) useAppStore.getState().setMatchState(env.state);
        break;
      case "event":
        if (env.event) useAppStore.getState().pushEvent(env.event);
        break;
      case "chat":
        if (env.chat) useAppStore.getState().pushChat(env.chat);
        break;
      case "error":
        console.warn("server error", env.error);
        break;
      case "hello_ack":
      case "pong":
        break;
    }
  }

  sendCommand(command: ClientCommand) {
    this.send({ type: "command", match_id: this.matchID, command });
  }

  sendChat(chat: ChatOutbound) {
    this.send({ type: "chat", match_id: this.matchID, chat });
  }

  private send(env: ClientEnvelope) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.ws.send(JSON.stringify(env));
  }

  close() {
    if (this.pingTimer) window.clearInterval(this.pingTimer);
    this.ws?.close();
    this.ws = null;
  }
}
