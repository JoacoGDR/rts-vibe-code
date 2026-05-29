import { Button, Panel } from "../../components/ui";
import type { MapDef } from "../../types/api";

interface MatchSetupProps {
  name: string;
  mapID: string;
  maps: MapDef[];
  selectedSlot: string;
  busy: boolean;
  onNameChange: (v: string) => void;
  onMapChange: (id: string) => void;
  onCreate: () => void;
}

export function MatchSetup({
  name,
  mapID,
  maps,
  selectedSlot,
  busy,
  onNameChange,
  onMapChange,
  onCreate,
}: MatchSetupProps) {
  return (
    <Panel title="Match setup" className="match-setup">
      <label className="field">
        Operation name
        <input value={name} onChange={(e) => onNameChange(e.target.value)} />
      </label>
      <label className="field">
        Theater
        <select value={mapID} onChange={(e) => onMapChange(e.target.value)}>
          {maps.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </label>
      <p className="muted match-setup__slot">
        Nation: <strong>{selectedSlot}</strong>
      </p>
      <Button variant="primary" onClick={onCreate} disabled={busy}>
        Create operation
      </Button>
    </Panel>
  );
}
