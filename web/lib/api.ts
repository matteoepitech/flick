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

// Upload: a single archive the user wants to send. `name` is the base name used
// for the .zip (the file name, or the folder name for a directory upload).
export interface Upload {
  name: string
  entries: UploadEntry[]
}

export async function uploadFile(
  upload: Upload,
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

  // Like the CLI, the web always uploads a zip archive: the server stores it
  // compressed and the client (CLI or web) extracts it on download. A folder is
  // stored with its full directory structure so it comes back intact.
  const zip = new JSZip()
  for (const entry of upload.entries) {
    zip.file(entry.path, entry.file)
  }
  const archive = await zip.generateAsync({ type: "blob", compression: "DEFLATE" })

  const form = new FormData()
  form.append("file", archive, `${upload.name}.zip`)

  return new Promise<string>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open("POST", url.toString())
    xhr.setRequestHeader("X-Flick-User-ID", uploaderId)

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
  // path: the full relative path inside the archive, e.g. "myfolder/sub/a.txt".
  // Preserved so a folder can be re-zipped with its structure intact.
  path: string
  // name: the basename of path, used for display and single-file downloads.
  name: string
  blob: Blob
  size: number
}

function basename(path: string): string {
  const i = path.lastIndexOf("/")
  return i === -1 ? path : path.slice(i + 1)
}

export interface DownloadResult {
  files: DownloadedFile[]
  // archiveName: base name (no ".zip") taken from the uploaded archive, used to
  // name the folder and its re-zipped download. Falls back to "flick-download".
  archiveName: string
}

export async function downloadByCode(code: string, signal?: AbortSignal): Promise<DownloadResult> {
  const url = apiUrl("/download")
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), { method: "GET", signal })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (!res.ok) throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))

  const form = await res.formData()
  const files: DownloadedFile[] = []
  let archiveName = "flick-download"

  // Every part is a zip archive (see uploadFile): extract its entries so the
  // user gets the real files back, not the .zip wrapper. A folder upload keeps
  // its directory structure in the entry paths (e.g. "myfolder/sub/a.txt").
  for (const value of form.getAll("file")) {
    if (!(value instanceof File)) continue

    // The part filename is "<original name>.zip" (folder or file name); keep it
    // so the folder download is named after what was actually sent.
    if (value.name) archiveName = value.name.replace(/\.zip$/i, "")

    const zip = await JSZip.loadAsync(value)
    for (const entry of Object.values(zip.files)) {
      if (entry.dir) continue
      const blob = await entry.async("blob")
      files.push({ path: entry.name, name: basename(entry.name), blob, size: blob.size })
    }
  }

  return { files, archiveName }
}

// hasFolderStructure: true when any downloaded entry lives inside a directory,
// meaning the user sent a folder rather than loose files.
export function hasFolderStructure(files: DownloadedFile[]): boolean {
  return files.some((file) => file.path.includes("/"))
}

// DownloadItem: one top-level entry inside a code, mirroring how the send page
// stages uploads. A folder groups every file under it; a loose file is a single
// entry. Both render the same way, only the icon and subtitle differ.
export interface DownloadItem {
  name: string
  isFolder: boolean
  entries: DownloadedFile[]
  size: number
}

// groupDownloadItems: Rebuild the top-level items from the flat entry list. Each
// first path segment becomes one item: a folder when it has nested files, a
// plain file otherwise. So a code holding "report.pdf" and "photos/a.jpg" shows
// one file and one folder side by side, like the upload list.
export function groupDownloadItems(files: DownloadedFile[]): DownloadItem[] {
  const order: string[] = []
  const groups = new Map<string, DownloadedFile[]>()

  for (const file of files) {
    const top = file.path.split("/")[0]
    const group = groups.get(top)
    if (group) {
      group.push(file)
    } else {
      groups.set(top, [file])
      order.push(top)
    }
  }

  return order.map((name) => {
    const entries = groups.get(name) ?? []
    return {
      name,
      isFolder: entries.some((entry) => entry.path.includes("/")),
      entries,
      size: entries.reduce((total, entry) => total + entry.size, 0),
    }
  })
}

// buildFolderArchive: Re-zip the downloaded entries into a single archive that
// preserves their relative paths, so the browser delivers the whole folder in
// one download (and it extracts straight back into that folder) instead of
// flattening every file. Returns the blob and the suggested filename.
export async function buildFolderArchive(
  files: DownloadedFile[],
  archiveName: string
): Promise<{ blob: Blob; name: string }> {
  const zip = new JSZip()
  for (const file of files) {
    zip.file(file.path, file.blob)
  }
  const blob = await zip.generateAsync({ type: "blob", compression: "DEFLATE" })
  return { blob, name: `${archiveName}.zip` }
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
