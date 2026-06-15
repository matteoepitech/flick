import { ApiError, whoami, type AuthSession } from "@/lib/api"

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
// server. A definite auth rejection (401 — unknown token or deleted user) clears
// the session and returns false, so a ghost session can't keep someone "logged
// in" after the account is gone. Transient/network errors keep the session to
// avoid logging users out when the API is briefly unreachable.
export async function verifySession(session: AuthSession, signal?: AbortSignal): Promise<boolean> {
  try {
    await whoami(session.token, signal)
    return true
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      clearSession()
      return false
    }
    return true
  }
}
