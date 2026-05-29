import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, mapsApi, matchesApi } from "../../api";
import type { MapDef } from "../../types/api";
import { MapPreview } from "./MapPreview";
import { MatchSetup } from "./MatchSetup";
import { NationPicker } from "./NationPicker";

export function CreateMatchView() {
  const navigate = useNavigate();
  const [maps, setMaps] = useState<MapDef[]>([]);
  const [name, setName] = useState("My Match");
  const [mapID, setMapID] = useState("classic-4p");
  const [selectedSlot, setSelectedSlot] = useState("red");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const selectedMap = useMemo(() => maps.find((m) => m.id === mapID), [maps, mapID]);

  useEffect(() => {
    mapsApi.list().then((loaded) => {
      setMaps(loaded);
      if (loaded.length > 0) {
        setMapID(loaded[0].id);
        setSelectedSlot(loaded[0].slots[0]?.id ?? "red");
      }
    }).catch((e: unknown) => {
      setErr(e instanceof ApiError ? e.message : String(e));
    });
  }, []);

  async function create() {
    setBusy(true);
    setErr(null);
    try {
      const match = await matchesApi.create(name, mapID, selectedSlot);
      navigate(`/match/${match.id}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function handleMapChange(id: string) {
    setMapID(id);
    const m = maps.find((x) => x.id === id);
    if (m?.slots[0]) setSelectedSlot(m.slots[0].id);
  }

  return (
    <div className="create-match-view">
      <header className="lobby-shell__header">
        <button
          type="button"
          className="btn btn--ghost"
          onClick={() => navigate("/lobby")}
        >
          ← War room
        </button>
        <h1 className="war-room-title">New operation</h1>
      </header>
      <div className="create-match-view__grid">
        <div className="lobby-shell__center">
          <MapPreview map={selectedMap} selectedSlot={selectedSlot} />
          <NationPicker
            map={selectedMap}
            selectedSlot={selectedSlot}
            onSelectSlot={setSelectedSlot}
          />
        </div>
        <MatchSetup
          name={name}
          mapID={mapID}
          maps={maps}
          selectedSlot={selectedSlot}
          busy={busy}
          onNameChange={setName}
          onMapChange={handleMapChange}
          onCreate={create}
        />
      </div>
      {err && <p className="err">{err}</p>}
    </div>
  );
}
