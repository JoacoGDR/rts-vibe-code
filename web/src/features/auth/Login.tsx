import { useState, type FormEvent } from "react";
import { ApiError, authApi } from "../../api";
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
    <div className="auth-card">
      <h2>{mode === "login" ? "Sign in" : "Create account"}</h2>
      <form onSubmit={submit}>
        <label>
          Email
          <input value={email} onChange={(e) => setEmail(e.target.value)} required />
        </label>
        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        {mode === "register" && (
          <label>
            Display name
            <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          </label>
        )}
        <button type="submit" disabled={busy}>
          {busy ? "Working…" : mode === "login" ? "Sign in" : "Register"}
        </button>
      </form>
      {err && <p className="err">{err}</p>}
      <button className="link" onClick={() => setMode(mode === "login" ? "register" : "login")}>
        {mode === "login" ? "Need an account? Register" : "Already registered? Sign in"}
      </button>
    </div>
  );
}
