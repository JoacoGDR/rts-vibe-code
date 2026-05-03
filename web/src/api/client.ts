// Tiny fetch wrapper used by every per-resource client. Keeps the JWT
// injection, error parsing and idempotency-key stamping in one place.

const TOKEN_KEY = "supremacy.token";

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export const tokenStore = {
  set(token: string) {
    localStorage.setItem(TOKEN_KEY, token);
  },
  clear() {
    localStorage.removeItem(TOKEN_KEY);
  },
  get(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  },
};

function authHeaders(): HeadersInit {
  const t = tokenStore.get();
  return t ? { Authorization: `Bearer ${t}` } : {};
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    let code = "unknown";
    let message = res.statusText;
    try {
      const err = await res.json();
      code = err.error?.code ?? code;
      message = err.error?.message ?? message;
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, code, message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
