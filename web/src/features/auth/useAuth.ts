import { useEffect, useState } from "react";
import { authApi } from "../../api";
import { useAppStore } from "../../app/store";

// useAuth bootstraps the session from a stored token (if any) and
// exposes login/register/logout helpers. The actual UI lives in Login.tsx.
export function useAuth() {
  const session = useAppStore((s) => s.session);
  const setSession = useAppStore((s) => s.setSession);
  const [bootstrapped, setBootstrapped] = useState(false);

  useEffect(() => {
    let alive = true;
    (async () => {
      const t = authApi.getToken();
      if (!t) {
        if (alive) setBootstrapped(true);
        return;
      }
      try {
        const me = await authApi.me();
        if (!alive) return;
        setSession({
          access_token: t,
          expires_at: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
          user: { id: me.id, email: me.email, display_name: me.display_name, color: "#5fa8ff" },
        });
      } catch {
        authApi.clearToken();
      } finally {
        if (alive) setBootstrapped(true);
      }
    })();
    return () => {
      alive = false;
    };
  }, [setSession]);

  function logout() {
    authApi.clearToken();
    setSession(null);
  }

  return { session, bootstrapped, logout };
}
