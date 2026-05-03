// idem produces a fresh idempotency key for a client command. The server
// uses these to deduplicate retries. Falls back to a timestamp-based key
// if crypto.randomUUID is unavailable (very old browsers).
export function idem(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}
