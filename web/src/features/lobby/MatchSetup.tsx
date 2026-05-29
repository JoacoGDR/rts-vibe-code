import { Button, Panel } from "../../components/ui";
import type { MapDef } from "../../types/api";

interface MatchSetupProps {
  name: string;
  mapID: string;
  maps: MapDef[];
  selectedSlot: string;
  joinCode: string;
  busy: boolean;
  onNameChange: (v: string) => void;
  onMapChange: (id: string) => void;
  onJoinCodeChange: (v: string) => void;
  onCreate: () => void;
  onJoinByCode: () => void;
}

export function MatchSetup({
  name,
  mapID,
  maps,
  selectedSlot,
  joinCode,
  busy,
  onNameChange,
  onMapChange,
  onJoinCodeChange,
  onCreate,
  onJoinByCode,
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

      <hr className="match-setup__divider" />

      <label className="field">
        Join by match ID
        <input
          value={joinCode}
          onChange={(e) => onJoinCodeChange(e.target.value)}
          placeholder="uuid"
        />
      </label>
      <Button variant="secondary" onClick={onJoinByCode} disabled={busy || !joinCode.trim()}>
        Join operation
      </Button>
    </Panel>
  );
}
