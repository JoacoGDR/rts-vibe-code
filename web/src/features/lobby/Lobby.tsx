import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, mapsApi, matchesApi } from "../../api";
import type { MapDef, MatchView } from "../../types/api";

export function Lobby() {
  const navigate = useNavigate();
  const [matches, setMatches] = useState<MatchView[]>([]);
  const [maps, setMaps] = useState<MapDef[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [name, setName] = useState("My Match");
  const [mapID, setMapID] = useState("tiny-2p");

  async function refresh() {
    try {
      const [ms, mp] = await Promise.all([matchesApi.list(), mapsApi.list()]);
      setMatches(ms);
      setMaps(mp);
      if (mp.length > 0 && !mp.find((m) => m.id === mapID)) {
        setMapID(mp[0].id);
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function create() {
    setBusy(true);
    setErr(null);
    try {
      const map = maps.find((m) => m.id === mapID) ?? maps[0];
      const slot = map?.slots[0]?.id ?? "red";
      const match = await matchesApi.create(name, mapID, slot);
      navigate(`/match/${match.id}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function join(id: string) {
    setBusy(true);
    setErr(null);
    try {
      await matchesApi.join(id);
      navigate(`/match/${id}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="lobby">
      <section>
        <h2>Create match</h2>
        <label>
          Name
          <input value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label>
          Map
          <select value={mapID} onChange={(e) => setMapID(e.target.value)}>
            {maps.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
        </label>
        <button onClick={create} disabled={busy}>
          Create
        </button>
      </section>

      <section>
        <h2>Open matches</h2>
        {matches.length === 0 && <p className="muted">No matches yet — create one above.</p>}
        <ul>
          {matches.map((m) => (
            <li key={m.id}>
              <strong>{m.name}</strong> &middot; <code>{m.map_id}</code> &middot;{" "}
              <span className={`status status-${m.status}`}>{m.status}</span> &middot;{" "}
              {m.players.length} player(s)
              <div className="actions">
                <button onClick={() => navigate(`/match/${m.id}`)}>Enter</button>
                {m.status === "waiting" && (
                  <button onClick={() => join(m.id)} disabled={busy}>
                    Join
                  </button>
                )}
              </div>
            </li>
          ))}
        </ul>
      </section>

      {err && <p className="err">{err}</p>}
    </div>
  );
}
