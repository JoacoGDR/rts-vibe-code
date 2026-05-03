import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, notificationsApi } from "../../api";
import type { NotificationItem } from "../../api";

const POLL_INTERVAL_MS = 30_000;

// NotificationBell is the topbar dropdown that surfaces in-app
// notifications written by the worker. It polls /unread-count on the
// configured interval and lazily fetches the full list when the user
// opens the panel.
export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<NotificationItem[]>([]);
  const [unread, setUnread] = useState(0);
  const [err, setErr] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  const refreshCount = useCallback(async () => {
    try {
      const { count } = await notificationsApi.unreadCount();
      setUnread(count);
    } catch (e) {
      // 401 means we're not logged in yet — ignore quietly.
      if (e instanceof ApiError && e.status === 401) return;
      setErr(String(e));
    }
  }, []);

  const refreshList = useCallback(async () => {
    try {
      const { items, unread_count } = await notificationsApi.list(undefined, 25);
      setItems(items);
      setUnread(unread_count);
    } catch (e) {
      setErr(String(e));
    }
  }, []);

  useEffect(() => {
    refreshCount();
    const id = window.setInterval(refreshCount, POLL_INTERVAL_MS);
    return () => window.clearInterval(id);
  }, [refreshCount]);

  useEffect(() => {
    if (!open) return;
    refreshList();
    notificationsApi.markRead().then(() => setUnread(0)).catch(() => {});
  }, [open, refreshList]);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (!open || !ref.current) return;
      if (!ref.current.contains(e.target as Node)) setOpen(false);
    }
    window.addEventListener("mousedown", onClick);
    return () => window.removeEventListener("mousedown", onClick);
  }, [open]);

  return (
    <div className="bell" ref={ref}>
      <button
        className="bell-toggle"
        onClick={() => setOpen((v) => !v)}
        aria-label="Notifications"
        title="Notifications"
      >
        🔔
        {unread > 0 && <span className="badge">{unread > 99 ? "99+" : unread}</span>}
      </button>
      {open && (
        <div className="panel">
          <header>
            <h4>Notifications</h4>
            <button className="link" onClick={() => setOpen(false)}>
              Close
            </button>
          </header>
          {err && <div className="empty">{err}</div>}
          {!err && items.length === 0 && <div className="empty">Nothing yet.</div>}
          {!err && items.length > 0 && (
            <ul>
              {items.map((n) => (
                <li key={n.id} className={n.read_at ? "" : "unread"}>
                  <strong>{labelFor(n.kind)}</strong>
                  <div>{describe(n)}</div>
                  <time>{new Date(n.created_at).toLocaleString()}</time>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

function labelFor(kind: string): string {
  switch (kind) {
    case "province_captured":
      return "Province lost";
    case "under_attack":
      return "Under attack";
    case "treaty_proposed":
      return "Treaty proposed";
    case "match_ended":
      return "Match ended";
    case "bot_takeover":
      return "AI took over your slot";
    default:
      return kind;
  }
}

function describe(n: NotificationItem): string {
  switch (n.kind) {
    case "province_captured": {
      const prov = String(n.payload?.["province"] ?? "?");
      const newOwner = String(n.payload?.["new_owner"] ?? "?");
      return `${prov} fell to ${newOwner}.`;
    }
    case "under_attack": {
      const prov = String(n.payload?.["province"] ?? "?");
      const attacker = String(n.payload?.["attacker"] ?? "?");
      return `${attacker} is attacking ${prov}.`;
    }
    case "treaty_proposed": {
      const from = String(n.payload?.["from"] ?? "?");
      const stance = String(n.payload?.["stance"] ?? "?");
      return `${from} proposed ${stance}.`;
    }
    case "match_ended": {
      const winners = Array.isArray(n.payload?.["winners"])
        ? (n.payload?.["winners"] as string[]).join(", ")
        : "";
      return winners ? `Winners: ${winners}` : "Match concluded.";
    }
    case "bot_takeover":
      return "An AI bot is now playing your slot.";
    default:
      return "";
  }
}
