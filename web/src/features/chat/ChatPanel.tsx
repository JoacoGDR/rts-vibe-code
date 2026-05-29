import { useEffect, useMemo, useRef, useState } from "react";
import { chatApi } from "../../api";
import { useAppStore } from "../../app/store";
import type { ChatInbound } from "../../types/wire";
import type { GameSocket } from "../match/socket";

const NEAR_BOTTOM_THRESHOLD_PX = 60;

interface Props {
  matchID: string;
  socket: GameSocket | null;
  ownSlot: string | undefined;
}

type Channel = "world" | "coalition";

export function ChatPanel({ matchID, socket, ownSlot }: Props) {
  const messages = useAppStore((s) => s.chat);
  const session = useAppStore((s) => s.session);
  const pushChat = useAppStore((s) => s.pushChat);
  const resetChat = useAppStore((s) => s.resetChat);
  const [channel, setChannel] = useState<Channel>("world");
  const [draft, setDraft] = useState("");
  const listRef = useRef<HTMLUListElement>(null);
  const isNearBottom = useRef(true);

  // Reset the local chat history when entering a different match.
  useEffect(() => {
    resetChat();
  }, [matchID, resetChat]);

  // Backfill recent world messages on mount so users see prior context.
  useEffect(() => {
    if (!matchID) return;
    chatApi
      .history(matchID, scopeKey(matchID, "world"))
      .then((rows) => rows.reverse().forEach(pushChat))
      .catch(() => undefined);
  }, [matchID, pushChat]);

  const visible = useMemo(
    () => messages.filter((m) => filterByChannel(m, channel, session?.user.id)),
    [messages, channel, session?.user.id],
  );

  // Auto-scroll to bottom when new messages arrive, but only if the user
  // is already near the bottom so manual scrolling up to read history is preserved.
  useEffect(() => {
    if (isNearBottom.current && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [visible.length]);

  function handleScroll() {
    if (!listRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = listRef.current;
    isNearBottom.current = scrollHeight - scrollTop - clientHeight < NEAR_BOTTOM_THRESHOLD_PX;
  }

  function send() {
    if (!socket || !draft.trim()) return;
    socket.sendChat({ scope: channel, body: draft.trim() });
    setDraft("");
  }

  return (
    <section className="chat">
      <header>
        <h3>Chat</h3>
        <div className="channels">
          <button
            className={channel === "world" ? "active" : ""}
            onClick={() => setChannel("world")}
          >
            World
          </button>
          <button
            className={channel === "coalition" ? "active" : ""}
            onClick={() => setChannel("coalition")}
            disabled={!ownSlot}
          >
            Coalition
          </button>
        </div>
      </header>
      <ul className="log" ref={listRef} onScroll={handleScroll}>
        {visible.length === 0 && <li className="muted">No messages yet</li>}
        {visible.map((m) => (
          <li key={m.id}>
            <span className="author">{m.author_slot || m.author_user_id.slice(0, 6)}</span>
            <span className="body">{m.body}</span>
          </li>
        ))}
      </ul>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          send();
        }}
      >
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder={`Message ${channel}…`}
          maxLength={1024}
        />
        <button type="submit" disabled={!draft.trim() || !socket}>
          Send
        </button>
      </form>
    </section>
  );
}

function scopeKey(matchID: string, channel: Channel): string {
  return channel === "world" ? `world:${matchID}` : `coal:${matchID}:`; // coalition history needs the leader, omit for now
}

function filterByChannel(msg: ChatInbound, channel: Channel, userID: string | undefined): boolean {
  if (channel === "world") return msg.scope.startsWith("world:");
  if (channel === "coalition") return msg.scope.startsWith("coal:");
  return userID ? msg.scope.includes(userID) : false;
}
