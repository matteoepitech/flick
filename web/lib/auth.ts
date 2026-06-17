import { ApiError, isAccountBlocked, whoami, type AuthSession } from "@/lib/api"

// Result of re-checking a stored session against the server.
// - "valid":   the account exists and is usable.
// - "invalid": the token/account is gone (session cleared); send to login.
// - "blocked": the account was blocked by an admin (session kept so we can tell
//              the user what happened); send to the blocked page.
export type SessionStatus = "valid" | "invalid" | "blocked"

// The signed-in session (user + token) is kept in localStorage. The API stores
// sessions server-side; this only remembers which token belongs to this browser.
const STORAGE_KEY = "flick.session"

export function saveSession(session: AuthSession): void {
  if (typeof window === "undefined") return
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
}

export function loadSession(): AuthSession | null {
  if (typeof window === "undefined") return null

  const raw = window.localStorage.getItem(STORAGE_KEY)
  if (!raw) return null

  try {
    const parsed = JSON.parse(raw) as AuthSession
    if (!parsed || typeof parsed.token !== "string" || !parsed.user) return null
    return parsed
  } catch {
    return null
  }
}

export function clearSession(): void {
  if (typeof window === "undefined") return
  window.localStorage.removeItem(STORAGE_KEY)
}

// verifySession: Confirm a stored session still maps to a real account on the
// server. A definite auth rejection (401, unknown token or deleted user) clears
// the session and returns false, so a ghost session can't keep someone "logged
// in" after the account is gone. Transient/network errors keep the session to
// avoid logging users out when the API is briefly unreachable.
export async function verifySession(session: AuthSession, signal?: AbortSignal): Promise<SessionStatus> {
  try {
    await whoami(session.token, signal)
    return "valid"
  } catch (err) {
    // Blocked by an admin: keep the session so the blocked page can show context.
    if (isAccountBlocked(err)) {
      return "blocked"
    }
    if (err instanceof ApiError && err.status === 401) {
      clearSession()
      return "invalid"
    }
    // Transient/network error: keep the user logged in to avoid false logouts.
    return "valid"
  }
}
