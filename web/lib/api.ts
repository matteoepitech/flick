import JSZip from "jszip"

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

export async function uploadFile(
  file: File,
  expiration: string,
  maxDownloadCount: number,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal
): Promise<string> {
  const url = apiUrl("/upload")
  url.searchParams.set("expiration", expiration)
  url.searchParams.set("maxDownloadCount", String(maxDownloadCount))

  // Like the CLI, the web always uploads a zip archive: the server stores it
  // compressed and the client (CLI or web) extracts it on download.
  const zip = new JSZip()
  zip.file(file.name, file)
  const archive = await zip.generateAsync({ type: "blob", compression: "DEFLATE" })

  const form = new FormData()
  form.append("file", archive, `${file.name}.zip`)

  return new Promise<string>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open("POST", url.toString())

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

export interface DownloadedFile {
  name: string
  blob: Blob
  size: number
}

export async function downloadByCode(code: string, signal?: AbortSignal): Promise<DownloadedFile[]> {
  const url = apiUrl("/download")
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), { method: "GET", signal })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (!res.ok) throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))

  const form = await res.formData()
  const files: DownloadedFile[] = []

  // Every part is a zip archive (see uploadFile): extract its entries so the
  // user gets the real files back, not the .zip wrapper.
  for (const value of form.getAll("file")) {
    if (!(value instanceof File)) continue

    const zip = await JSZip.loadAsync(value)
    for (const entry of Object.values(zip.files)) {
      if (entry.dir) continue
      const blob = await entry.async("blob")
      files.push({ name: entry.name, blob, size: blob.size })
    }
  }

  return files
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
  createdAt?: string
}

export interface AuthSession {
  user: AuthUser
  token: string
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

  return (await res.json()) as AuthUser
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

  // The server replies with { token, expires_at, user: { id, username, email, role, created_at } }.
  const data = (await res.json()) as {
    token?: string
    user?: { id?: string; username?: string; email?: string; role?: string; created_at?: string }
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
      role: data.user.role === "admin" ? "admin" : "user",
      createdAt: data.user.created_at,
    },
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
    user?: { id?: string; username?: string; email?: string; role?: string; created_at?: string }
  }
  if (!data.user) {
    throw new ApiError(res.status, "Invalid whoami response")
  }

  return {
    id: data.user.id ?? "",
    username: data.user.username ?? "",
    email: data.user.email ?? "",
    role: data.user.role === "admin" ? "admin" : "user",
    createdAt: data.user.created_at,
  }
}

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
