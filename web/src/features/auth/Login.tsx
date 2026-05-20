import { useState, type FormEvent } from "react";
import { ApiError, authApi } from "../../api";
import { Button, Panel } from "../../components/ui";
import { useAppStore } from "../../app/store";

export function Login() {
  const setSession = useAppStore((s) => s.setSession);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      const session =
        mode === "login"
          ? await authApi.login(email, password)
          : await authApi.register(email, password, displayName || email);
      authApi.setToken(session.access_token);
      setSession(session);
    } catch (e) {
      setErr(e instanceof ApiError ? `${e.code}: ${e.message}` : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Panel title={mode === "login" ? "Sign in" : "Create account"} className="auth-card">
      <form onSubmit={submit}>
        <label className="field">
          Email
          <input value={email} onChange={(e) => setEmail(e.target.value)} required />
        </label>
        <label className="field">
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        {mode === "register" && (
          <label className="field">
            Display name
            <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </label>
        )}
        <Button type="submit" variant="primary" disabled={busy}>
          {busy ? "Working…" : mode === "login" ? "Sign in" : "Register"}
        </Button>
      </form>
      {err && <p className="err">{err}</p>}
      <Button variant="ghost" onClick={() => setMode(mode === "login" ? "register" : "login")}>
        {mode === "login" ? "Need an account? Register" : "Already registered? Sign in"}
      </Button>
    </Panel>
  );
}
