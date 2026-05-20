import { Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { Button } from "../components/ui";
import { Login } from "../features/auth/Login";
import { useAuth } from "../features/auth/useAuth";
import { Lobby } from "../features/lobby/Lobby";
import { MatchView } from "../features/match/MatchView";
import { NotificationBell } from "../features/notifications/NotificationBell";

export function App() {
  const { session, bootstrapped, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  if (!bootstrapped) {
    return <div className="app-shell app-loading">Loading war room…</div>;
  }

  function handleLogout() {
    logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="app-shell">
      <header className="topbar war-room-topbar">
        <h1 className="war-room-title topbar__brand">War room</h1>
        {session && (
          <>
            <span className="topbar__user muted">{session.user.display_name}</span>
            <NotificationBell />
            <Button variant="ghost" size="sm" onClick={handleLogout}>
              Logout
            </Button>
          </>
        )}
      </header>

      <Routes>
        <Route path="/login" element={session ? <Navigate to="/lobby" replace /> : <Login />} />
        <Route
          path="/lobby"
          element={
            session ? <Lobby /> : <Navigate to="/login" replace state={{ from: location }} />
          }
        />
        <Route
          path="/match/:matchID"
          element={session ? <MatchView /> : <Navigate to="/login" replace />}
        />
        <Route path="*" element={<Navigate to={session ? "/lobby" : "/login"} replace />} />
      </Routes>
    </div>
  );
}
