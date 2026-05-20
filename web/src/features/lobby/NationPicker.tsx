import type { CSSProperties } from "react";
import type { MapDef, MatchView } from "../../types/api";

interface NationPickerProps {
  map: MapDef | undefined;
  selectedSlot: string;
  onSelectSlot: (slot: string) => void;
  match?: MatchView | null;
}

export function NationPicker({ map, selectedSlot, onSelectSlot, match }: NationPickerProps) {
  if (!map) return <p className="muted">Select a theater map.</p>;

  return (
    <div className="nation-picker">
      <h3 className="war-room-title">Choose nation</h3>
      <div className="nation-picker__grid">
        {map.slots.map((slot) => {
          const taken = match?.players.some((p) => p.slot === slot.id);
          const capital = map.provinces.find((p) => p.id === slot.capital);
          const units = map.starting_units.filter((u) => u.slot === slot.id);
          return (
            <button
              key={slot.id}
              type="button"
              className={`nation-card ${selectedSlot === slot.id ? "nation-card--selected" : ""} ${taken ? "nation-card--taken" : ""}`}
              style={{ "--nation-color": slot.color } as CSSProperties}
              disabled={Boolean(taken)}
              onClick={() => onSelectSlot(slot.id)}
            >
              <span className="nation-card__swatch" />
              <span className="nation-card__name">{slot.id}</span>
              <span className="nation-card__capital">
                Capital: {capital?.name ?? slot.capital}
              </span>
              <span className="nation-card__units muted">
                {units.map((u) => u.type).join(", ") || "—"}
              </span>
              {taken && <span className="nation-card__badge">Occupied</span>}
            </button>
          );
        })}
      </div>
    </div>
  );
}
