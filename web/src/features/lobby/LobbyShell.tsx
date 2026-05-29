import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, mapsApi, matchesApi } from "../../api";
import { useAppStore } from "../../app/store";
import type { MapDef, MatchView } from "../../types/api";
import { GameBrowser, type LobbyFilter } from "./GameBrowser";
import { MapPreview } from "./MapPreview";
import { MatchSetup } from "./MatchSetup";
import { NationPicker } from "./NationPicker";
import { SlotGrid } from "./SlotGrid";

export function LobbyShell() {
  const navigate = useNavigate();
  const session = useAppStore((s) => s.session);
  const [matches, setMatches] = useState<MatchView[]>([]);
  const [maps, setMaps] = useState<MapDef[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [name, setName] = useState("My Match");
  const [mapID, setMapID] = useState("classic-4p");
  const [selectedSlot, setSelectedSlot] = useState("red");
  const [filter, setFilter] = useState<LobbyFilter>("all");
  const [joinCode, setJoinCode] = useState("");
  const [selectedMatch, setSelectedMatch] = useState<MatchView | null>(null);

  // When a match is selected in the browser, the center panel shows that match's map.
  // When no match is selected, it falls back to the create-form map.
  const displayedMapID = selectedMatch ? selectedMatch.map_id : mapID;
  const displayedMap = useMemo(
    () => maps.find((m) => m.id === displayedMapID),
    [maps, displayedMapID],
  );

  async function refresh() {
    try {
      const [ms, mp] = await Promise.all([matchesApi.list(), mapsApi.list()]);
      setMatches(ms);
      setMaps(mp);
      if (mp.length > 0 && !mp.find((m) => m.id === mapID)) {
        setMapID(mp[0].id);
        setSelectedSlot(mp[0].slots[0]?.id ?? "red");
      }
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filtered = useMemo(() => {
    const uid = session?.user.id;
    return matches.filter((m) => {
      switch (filter) {
        case "open":
          return m.status === "waiting";
        case "mine":
          return uid ? m.players.some((p) => p.user_id === uid) : false;
        case "active":
          return m.status === "active" || m.status === "starting";
        case "ended":
          return m.status === "ended" || m.status === "abandoned";
        default:
          return true;
      }
    });
  }, [matches, filter, session?.user.id]);

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

  async function join(id: string) {
    setBusy(true);
    setErr(null);
    try {
      const matchToJoin = matches.find((m) => m.id === id);
      const matchMap = maps.find((m) => m.id === matchToJoin?.map_id);
      const slotExists = matchMap?.slots.some((s) => s.id === selectedSlot);
      if (!slotExists) {
        setErr("Selected slot does not exist on this map — pick a valid nation first.");
        setBusy(false);
        return;
      }
      await matchesApi.join(id, selectedSlot);
      navigate(`/match/${id}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function joinByCode() {
    const id = joinCode.trim();
    if (!id) return;
    await join(id);
  }

  async function leave(id: string) {
    setBusy(true);
    setErr(null);
    try {
      await matchesApi.leave(id);
      await refresh();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="lobby-shell">
      <header className="lobby-shell__header">
        <h1 className="war-room-title">War room · Match command</h1>
      </header>
      <div className="lobby-shell__grid">
        <GameBrowser
          matches={filtered}
          filter={filter}
          onFilterChange={setFilter}
          selectedMatchId={selectedMatch?.id ?? null}
          onSelectMatch={setSelectedMatch}
          onEnter={(id) => navigate(`/match/${id}`)}
          onJoin={join}
          onLeave={leave}
          busy={busy}
          userId={session?.user.id}
        />
        <div className="lobby-shell__center">
          <MapPreview map={displayedMap} />
          <NationPicker
            map={displayedMap}
            selectedSlot={selectedSlot}
            onSelectSlot={setSelectedSlot}
            match={selectedMatch}
          />
          <SlotGrid map={displayedMap} match={selectedMatch} />
        </div>
        <MatchSetup
          name={name}
          mapID={mapID}
          maps={maps}
          selectedSlot={selectedSlot}
          joinCode={joinCode}
          busy={busy}
          onNameChange={setName}
          onMapChange={(id) => {
            setMapID(id);
            const m = maps.find((x) => x.id === id);
            if (m?.slots[0]) setSelectedSlot(m.slots[0].id);
          }}
          onJoinCodeChange={setJoinCode}
          onCreate={create}
          onJoinByCode={joinByCode}
        />
      </div>
      {err && <p className="err lobby-shell__err">{err}</p>}
    </div>
  );
}
