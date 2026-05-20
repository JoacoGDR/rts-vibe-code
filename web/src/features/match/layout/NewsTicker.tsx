import type { ServerEvent } from "../../../types/wire";

interface NewsTickerProps {
  events: ServerEvent[];
}

function formatEvent(e: ServerEvent): string {
  const parts = [e.kind.replace(/_/g, " ")];
  if (e.province) parts.push(`@ ${e.province}`);
  if (e.unit_id) parts.push(`unit ${e.unit_id.slice(0, 8)}`);
  return parts.join(" · ");
}

export function NewsTicker({ events }: NewsTickerProps) {
  const recent = events.slice(-8).reverse();
  if (recent.length === 0) return null;

  return (
    <div className="news-ticker" aria-live="polite">
      <span className="news-ticker__label">Intel</span>
      <div className="news-ticker__track">
        {recent.map((e, i) => (
          <span key={`${e.occur_at}-${i}`} className="ticker-item">
            {formatEvent(e)}
          </span>
        ))}
      </div>
    </div>
  );
}
