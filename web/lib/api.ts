import JSZip from "jszip"
import { hashBlob, equal } from "@/lib/checksum"

const API_PREFIX = "/api/v1"

// In the browser the API lives on the same origin: Caddy routes /api/v1/* to
// the Go API. Server-side (SSR) we talk to the API directly on the Docker
// network, overridable through FLICK_API_URL.
function getApiBase(): string {
  if (typeof window !== "undefined") {
    return window.location.origin
  }
  return process.env.FLICK_API_URL ?? "http://flick-api:15702"
}

function apiUrl(path: string): URL {
  return new URL(`${API_PREFIX}${path}`, getApiBase())
}

export function getConfigureUrl(): string {
  return apiUrl("/configure").toString()
}

export class CodeNotFoundError extends Error {
  constructor(public code: string) {
    super(`Code not found: ${code}`)
    this.name = "CodeNotFoundError"
  }
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message)
    this.name = "ApiError"
  }
}

// Thrown when a downloaded archive's BLAKE3 digest does not match the one the
// server announced: the bytes were corrupted in transit and must not be saved.
export class ChecksumMismatchError extends Error {
  constructor(
    public expected: string,
    public got: string
  ) {
    super("Downloaded file is corrupted (checksum mismatch)")
    this.name = "ChecksumMismatchError"
  }
}

// Error message the API returns (HTTP 403, body { "error": "Account blocked" })
// on any authenticated endpoint when the account has been blocked by an admin.
export const ACCOUNT_BLOCKED_CODE = "Account blocked"

// isAccountBlocked: True when an error is the API's "account blocked" rejection,
// as opposed to other 403s (e.g. "admin privileges required").
export function isAccountBlocked(err: unknown): boolean {
  return err instanceof ApiError && err.status === 403 && err.message === ACCOUNT_BLOCKED_CODE
}

// parseErrorMessage: Extracts the human-readable message from a server error
// body. The API returns `{"error": "..."}`; falls back to the raw text.
function parseErrorMessage(body: string, fallback: string): string {
  if (!body) return fallback
  try {
    const parsed = JSON.parse(body) as { error?: unknown }
    if (typeof parsed.error === "string" && parsed.error) return parsed.error
  } catch {
    // Not JSON, use the raw body below.
  }
  return body || fallback
}

export interface UploadProgress {
  loaded: number
  total: number
}

// Anonymous uploader id, the web counterpart of the CLI's credentials file: it
// lets the server attribute uploads from visitors who are not signed in.
const UPLOADER_ID_KEY = "flick.uploaderId"
// Mirrors the key written by lib/auth.ts; read directly to avoid an import cycle.
const SESSION_KEY = "flick.session"

// identify: Ask the server to create an anonymous user and return its UUID. The
// API replies 201 with { "user_id": "<uuid>" } (see the CLI's /identify call).
export async function identify(signal?: AbortSignal): Promise<string> {
  const url = apiUrl("/identify")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as { user_id?: string }
  if (!data.user_id) throw new ApiError(res.status, "Invalid identify response")
  return data.user_id
}

// ensureUploaderId: Resolve the X-Flick-User-ID the server requires on upload. A
// signed-in visitor uploads under their account id; everyone else reuses (or
// creates once) an anonymous id kept in localStorage, just like the CLI keeps
// its credentials file.
export async function ensureUploaderId(signal?: AbortSignal): Promise<string> {
  if (typeof window !== "undefined") {
    const rawSession = window.localStorage.getItem(SESSION_KEY)
    if (rawSession) {
      try {
        const parsed = JSON.parse(rawSession) as { user?: { id?: string } }
        if (parsed.user?.id) return parsed.user.id
      } catch {
        // Malformed session: fall through to the anonymous id below.
      }
    }

    const existing = window.localStorage.getItem(UPLOADER_ID_KEY)
    if (existing) return existing
  }

  const id = await identify(signal)
  if (typeof window !== "undefined") window.localStorage.setItem(UPLOADER_ID_KEY, id)
  return id
}

// UploadEntry: one file to store in the archive, keyed by its path relative to
// the upload root. A loose file uses just its name ("photo.png"); a folder
// keeps its structure ("myfolder/sub/a.txt"), exactly like the CLI.
export interface UploadEntry {
  path: string
  file: File
}

// Upload: a single item the user staged (a loose file or a folder). `name` is its
// display name; `isFolder` tells us whether its entries keep a directory prefix.
export interface Upload {
  name: string
  isFolder: boolean
  entries: UploadEntry[]
}

// randomArchiveName: the name under which the whole upload is stored and later
// handed back on download. A random uuid keeps unrelated uploads from colliding
// on disk and is exactly what the receiver saves (e.g. "<uuid>.zip").
function randomArchiveName(): string {
  const id =
    typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
      ? crypto.randomUUID()
      : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
  return `${id}.zip`
}

export async function uploadFile(
  items: Upload[],
  expiration: string,
  maxDownloadCount: number,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal
): Promise<string> {
  const url = apiUrl("/upload")
  url.searchParams.set("expiration", expiration)
  url.searchParams.set("maxDownloadCount", String(maxDownloadCount))

  // The server requires a known uploader (X-Flick-User-ID), exactly like the CLI.
  const uploaderId = await ensureUploaderId(signal)

  // Everything the user staged goes into ONE archive, stored and served as-is:
  // the download just hands this single zip back (no client unzip/rezip) and the
  // info endpoint reads inside it to list each item. A loose file sits at the
  // archive root; a folder keeps its full structure. The archive name is a random
  // uuid, so two unrelated uploads can never collide on disk.
  const zip = new JSZip()
  const usedTop = new Set<string>()

  // Keep top-level names unique so two staged items never overwrite each other:
  // files keep their extension when suffixed ("report (2).pdf"), folders don't.
  const dedupTop = (name: string, keepExt: boolean): string => {
    if (!usedTop.has(name)) {
      usedTop.add(name)
      return name
    }
    const dot = keepExt ? name.lastIndexOf(".") : -1
    const base = dot > 0 ? name.slice(0, dot) : name
    const ext = dot > 0 ? name.slice(dot) : ""
    for (let n = 2; ; n++) {
      const candidate = `${base} (${n})${ext}`
      if (!usedTop.has(candidate)) {
        usedTop.add(candidate)
        return candidate
      }
    }
  }

  for (const item of items) {
    if (item.isFolder) {
      const top = dedupTop(item.name, false)
      for (const entry of item.entries) {
        // entry.path is "<folder>/<rest>"; re-root it under the deduped name.
        const slash = entry.path.indexOf("/")
        const rest = slash === -1 ? entry.path : entry.path.slice(slash + 1)
        zip.file(`${top}/${rest}`, entry.file)
      }
    } else {
      zip.file(dedupTop(item.name, true), item.entries[0].file)
    }
  }

  const archive = await zip.generateAsync({ type: "blob", compression: "DEFLATE" })

  // Checksum the exact archive bytes we are about to upload; the server stores
  // this digest and hands it back on download so the downloader can confirm the
  // transfer is intact. Identical to the CLI and Go server (BLAKE3 hex).
  const archiveChecksum = await hashBlob(archive)

  const form = new FormData()
  form.append("file", archive, randomArchiveName())

  return new Promise<string>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open("POST", url.toString())
    xhr.setRequestHeader("X-Flick-User-ID", uploaderId)
    xhr.setRequestHeader("X-Flick-Checksum", archiveChecksum)

    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress({ loaded: event.loaded, total: event.total })
      }
    })

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(xhr.responseText.trim())
      } else {
        reject(new ApiError(xhr.status, parseErrorMessage(xhr.responseText, xhr.statusText)))
      }
    })
    xhr.addEventListener("error", () => reject(new ApiError(0, "Network error")))
    xhr.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")))

    signal?.addEventListener("abort", () => xhr.abort())
    xhr.send(form)
  })
}

// DownloadedArchive: one stored item pulled back from the server (the combined
// <uuid>.zip), which the browser saves as-is.
export interface DownloadedArchive {
  name: string
  blob: Blob
}

// downloadByCode: Pull a code's stored archive(s) in ONE request (this consumes
// the single-use download). The server replies with a multipart/form-data body
// (the same shape the CLI reads): each "file" part is a stored zip, returned
// untouched so the caller saves it.
export async function downloadByCode(code: string, signal?: AbortSignal): Promise<DownloadedArchive[]> {
  const url = apiUrl("/download")
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), { method: "GET", signal })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (!res.ok) throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))

  // The server announces the stored archive's BLAKE3 digest. Older uploads
  // without a stored checksum omit it, in which case we skip verification.
  const expectedChecksum = res.headers.get("X-Flick-Checksum")

  const form = await res.formData()
  const archives: DownloadedArchive[] = []
  for (const value of form.getAll("file")) {
    if (!(value instanceof File)) continue

    // Recompute over the bytes we actually received and refuse a corrupted
    // archive before handing it to the caller to save.
    if (expectedChecksum) {
      const got = await hashBlob(value)
      if (!equal(got, expectedChecksum)) {
        throw new ChecksumMismatchError(expectedChecksum, got)
      }
    }

    archives.push({ name: value.name, blob: value })
  }
  return archives
}

// triggerBlobDownload: Save a blob under filename via a temporary <a download>.
export function triggerBlobDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// DownloadInfoItem: one item behind a code (a loose file or a folder), listed
// without transmitting any content.
export interface DownloadInfoItem {
  name: string
  isFolder: boolean
  fileCount: number
  size: number
}

export interface DownloadInfo {
  items: DownloadInfoItem[]
  // True when the upload is end-to-end encrypted. The browser has no key and no
  // way to decrypt, so the receive page blocks the download and points to the CLI.
  encrypted: boolean
}

// fetchDownloadInfo: List the items behind a code WITHOUT consuming a download.
// The receive page uses this on load so merely opening the page never burns the
// single-use code; the real (consuming) transfer happens later via downloadByCode.
export async function fetchDownloadInfo(code: string, signal?: AbortSignal): Promise<DownloadInfo> {
  const url = apiUrl("/download/info")
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), { method: "GET", signal })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (!res.ok) throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))

  const data = (await res.json()) as { items?: DownloadInfoItem[]; encrypted?: boolean }
  return { items: data.items ?? [], encrypted: data.encrypted === true }
}

export async function loadUserConfiguration(
  signal?: AbortSignal
): Promise<Record<string, string | number | boolean>> {
  const url = apiUrl("/user-configure")

  const res = await fetch(url.toString(), { method: "GET", signal })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as unknown
  if (data === null || typeof data !== "object") {
    throw new ApiError(res.status, "Invalid configuration response")
  }
  return data as Record<string, string | number | boolean>
}

export async function loadConfiguration(
  signal?: AbortSignal
): Promise<Record<string, string | number | boolean>> {
  const url = apiUrl("/configure")

  const res = await fetch(url.toString(), { method: "GET", signal })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as unknown
  if (data === null || typeof data !== "object") {
    throw new ApiError(res.status, "Invalid configuration response")
  }
  return data as Record<string, string | number | boolean>
}

export async function saveConfiguration(
  values: Record<string, string | number | boolean>,
  signal?: AbortSignal
): Promise<void> {
  const url = apiUrl("/configure")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(values),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

export interface ServerLimits {
  default: number
  max: number
  allowMultiple: boolean
  maxFileSizeMb: number
}

export async function fetchServerLimits(signal?: AbortSignal): Promise<ServerLimits> {
  const conf = await loadUserConfiguration(signal)
  const def = typeof conf.default_download_count === "number" ? conf.default_download_count : 1
  const max = typeof conf.max_download_count === "number" ? conf.max_download_count : def
  const allowMultiple = conf.allow_multiple_downloads === true
  const maxFileSizeMb = typeof conf.max_file_size_mb === "number" ? conf.max_file_size_mb : 1000
  return { default: def, max: Math.max(max, def), allowMultiple, maxFileSizeMb }
}

export interface StatsSnapshot {
  timestamp: string
  activeCodes: number
  totalUploads: number
  totalDownloads: number
  userCount: number
  storageBytes: number
}

export async function fetchStats(signal?: AbortSignal): Promise<StatsSnapshot> {
  const url = apiUrl("/stats")

  const res = await fetch(url.toString(), { method: "GET", signal })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as unknown
  if (data === null || typeof data !== "object") {
    throw new ApiError(res.status, "Invalid stats response")
  }

  const obj = data as Record<string, unknown>
  const activeCodes = typeof obj.activeCodes === "number" ? obj.activeCodes : 0
  const totalUploads = typeof obj.totalUploads === "number" ? obj.totalUploads : 0
  const totalDownloads = typeof obj.totalDownloads === "number" ? obj.totalDownloads : 0
  const userCount = typeof obj.userCount === "number" ? obj.userCount : 0
  const storageBytes = typeof obj.storageBytes === "number" ? obj.storageBytes : 0
  const timestamp =
    typeof obj.timestamp === "string" ? obj.timestamp : new Date().toISOString()

  return { timestamp, activeCodes, totalUploads, totalDownloads, userCount, storageBytes }
}

// Global account role, mirrors the API's user_role enum.
export type UserRole = "admin" | "user"

// Role a user holds inside a group, mirrors the API's group_role enum. Carried
// on the session so the dashboard can show maintainers their group view. The API
// does not expose group memberships yet, so this stays undefined for now.
export type GroupRole = "member" | "maintainer" | "owner"

export interface AuthUser {
  id: string
  username: string
  email: string
  role: UserRole
  groupRole?: GroupRole
  blocked: boolean
  createdAt?: string
}

export interface AuthSession {
  user: AuthUser
  token: string
}

// coerceRole: Defensive coercion of an unknown role value coming from the API.
export function coerceRole(value: unknown): UserRole {
  return value === "admin" ? "admin" : "user"
}

export async function registerUser(
  username: string,
  email: string,
  password: string,
  signal?: AbortSignal
): Promise<AuthUser> {
  const url = apiUrl("/register")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, email, password }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as {
    id?: string
    username?: string
    email?: string
    role?: unknown
    blocked?: unknown
    created_at?: string
  }
  return {
    id: data.id ?? "",
    username: data.username ?? "",
    email: data.email ?? "",
    role: coerceRole(data.role),
    blocked: data.blocked === true,
    createdAt: data.created_at,
  }
}

export async function loginUser(email: string, password: string, signal?: AbortSignal): Promise<AuthSession> {
  const url = apiUrl("/login")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  // The server replies with { token, expires_at, user: { id, username, email, role, blocked, created_at } }.
  const data = (await res.json()) as {
    token?: string
    user?: {
      id?: string
      username?: string
      email?: string
      role?: unknown
      blocked?: unknown
      created_at?: string
    }
  }
  if (!data.token || !data.user) {
    throw new ApiError(res.status, "Invalid login response")
  }

  return {
    token: data.token,
    user: {
      id: data.user.id ?? "",
      username: data.user.username ?? "",
      email: data.user.email ?? "",
      role: coerceRole(data.user.role),
      blocked: data.user.blocked === true,
      createdAt: data.user.created_at,
    },
  }
}

// approveDevice: Approve a pending CLI device authorization. `userCode` is the
// short code the CLI displayed; `token` is the current web session token, which
// proves who is approving. On success the server stores a fresh session token on
// the device authorization, which the CLI then fetches on its next poll.
export async function approveDevice(userCode: string, token: string, signal?: AbortSignal): Promise<void> {
  const url = apiUrl("/device/approve")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user_code: userCode, token }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// whoami: Resolve the account a session token belongs to. Throws an ApiError
// with status 401 when the token is unknown or its user no longer exists, which
// callers use to purge a stale session. Uses POST because fetch() cannot send a
// body on a GET request.
export async function whoami(token: string, signal?: AbortSignal): Promise<AuthUser> {
  const url = apiUrl("/whoami")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as {
    user?: {
      id?: string
      username?: string
      email?: string
      role?: unknown
      blocked?: unknown
      created_at?: string
    }
  }
  if (!data.user) {
    throw new ApiError(res.status, "Invalid whoami response")
  }

  return {
    id: data.user.id ?? "",
    username: data.user.username ?? "",
    email: data.user.email ?? "",
    role: coerceRole(data.user.role),
    blocked: data.user.blocked === true,
    createdAt: data.user.created_at,
  }
}

export interface AdminUser {
  id: string
  username: string
  email: string
  role: UserRole
  blocked: boolean
  createdAt?: string
}

// toAdminUser: Maps a raw API user object (snake_case) to AdminUser.
function toAdminUser(raw: unknown): AdminUser {
  const obj = (raw ?? {}) as Record<string, unknown>
  return {
    id: typeof obj.id === "string" ? obj.id : "",
    username: typeof obj.username === "string" ? obj.username : "",
    email: typeof obj.email === "string" ? obj.email : "",
    role: coerceRole(obj.role),
    blocked: obj.blocked === true,
    createdAt: typeof obj.created_at === "string" ? obj.created_at : undefined,
  }
}

// listUsers: Admin-only fetch of every user. Requires an admin session token.
export async function listUsers(token: string, signal?: AbortSignal): Promise<AdminUser[]> {
  const url = apiUrl("/admin/users")

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as unknown
  if (!Array.isArray(data)) {
    throw new ApiError(res.status, "Invalid users response")
  }
  return data.map(toAdminUser)
}

// UserUpdate: The partial PATCH payload. Only provided fields are changed.
export interface UserUpdate {
  username?: string
  email?: string
  password?: string
  role?: UserRole
  blocked?: boolean
}

// updateUser: Admin-only partial update (PATCH) of a single user.
export async function updateUser(
  token: string,
  id: string,
  changes: UserUpdate,
  signal?: AbortSignal
): Promise<AdminUser> {
  const url = apiUrl(`/admin/users/${id}`)

  const res = await fetch(url.toString(), {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(changes),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  return toAdminUser(await res.json())
}
