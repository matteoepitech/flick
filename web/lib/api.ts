import { makeZip } from "client-zip"
import * as tus from "tus-js-client"
import { hashBlob, equal } from "@/lib/checksum"

const API_PREFIX = "/api/v1"

// Size of each chunk streamed to the server during a tus upload. A fixed chunk
// size keeps the browser from holding the whole archive in a single request and
// lets an interrupted upload resume; it matches the CLI's 16 MiB so both senders
// behave identically.
const TUS_CHUNK_SIZE = 16 * 1024 * 1024 // 16 MiB

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

// Thrown when a download is refused because the supplied password is missing or
// wrong (HTTP 401). The receive page catches it to (re-)prompt for the password.
export class PasswordRequiredError extends Error {
  constructor(public code: string) {
    super(`Password required for code: ${code}`)
    this.name = "PasswordRequiredError"
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
  phase?: "zipping" | "uploading"
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

// archiveEntry: one file destined for the zip, named by its path inside the
// archive. `input` is streamed (never read fully into memory) by the zip writer.
interface archiveEntry {
  name: string
  input: Blob
  lastModified: Date
}

// archiveEntries: Flatten everything the user staged into the list of zip
// entries. A loose file sits at the archive root; a folder keeps its full
// structure. Top-level names are de-duplicated so two staged items never collide.
function archiveEntries(items: Upload[]): archiveEntry[] {
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

  const entries: archiveEntry[] = []
  for (const item of items) {
    if (item.isFolder) {
      const top = dedupTop(item.name, false)
      for (const entry of item.entries) {
        // entry.path is "<folder>/<rest>"; re-root it under the deduped name.
        const slash = entry.path.indexOf("/")
        const rest = slash === -1 ? entry.path : entry.path.slice(slash + 1)
        entries.push({ name: `${top}/${rest}`, input: entry.file, lastModified: new Date(entry.file.lastModified) })
      }
    } else {
      const file = item.entries[0].file
      entries.push({ name: dedupTop(item.name, true), input: file, lastModified: new Date(file.lastModified) })
    }
  }
  return entries
}

// PreparedArchive: a staged upload archive ready to hand to tus. `archive` is a
// sliceable Blob (disk-backed when OPFS is available), `checksum` is its BLAKE3
// digest, and `cleanup` releases any temporary storage once the upload is done.
interface PreparedArchive {
  archive: Blob
  checksum: string
  cleanup: () => Promise<void>
}

const noopCleanup = async (): Promise<void> => {}

// opfsRoot: The Origin Private File System root, or null when the browser does
// not expose it. OPFS is disk-backed storage private to the origin; staging the
// archive there is what keeps a multi-gigabyte upload from ever touching RAM.
async function opfsRoot(): Promise<FileSystemDirectoryHandle | null> {
  try {
    if (typeof navigator === "undefined" || !navigator.storage?.getDirectory) return null
    return await navigator.storage.getDirectory()
  } catch {
    return null
  }
}

// buildUploadArchive: Pack everything the user staged into ONE zip and checksum
// the exact bytes. client-zip streams each source file straight into the archive
// and the archive itself is piped to a disk-backed OPFS file, so neither the
// inputs nor the finished zip are ever held in memory — a huge upload stays flat
// on RAM. The resulting File is still sliceable, which is all tus needs to send
// it back up chunk by chunk. Entries are stored uncompressed (client-zip does no
// deflate); for a transfer tool that trades a little bandwidth for bounded memory
// and is moot for already-compressed media. Shared by public and group uploads.
async function buildUploadArchive(items: Upload[], onProgress?: (loaded: number) => void): Promise<PreparedArchive> {
  const entries = archiveEntries(items)
  const total = items.reduce((sum, i) => sum + i.entries.reduce((s, e) => s + e.file.size, 0), 0)

  // A fresh stream per attempt: a ReadableStream can only be consumed once.
  const zipStream = (): ReadableStream<Uint8Array> => {
    if (!onProgress) return makeZip(entries)

    let written = 0
    return makeZip(
      entries.map(entry => ({
        name: entry.name,
        input: chunkedStream(entry.input, 65536, n => {
          written += n
          onProgress(written)
        }),
        lastModified: entry.lastModified,
        size: entry.input.size,
      }))
    )
  }

  const root = await opfsRoot()
  if (root) {
    const tmpName = randomArchiveName()
    try {
      const handle = await root.getFileHandle(tmpName, { create: true })
      const writable = await handle.createWritable()
      // pipeTo drains the zip to disk and closes the file when the stream ends.
      await zipStream().pipeTo(writable as unknown as WritableStream<Uint8Array>)

      const archive = await handle.getFile()
      // Checksum the exact archive bytes (streamed from disk); the server stores
      // this digest and hands it back on download so the downloader can confirm
      // the transfer is intact (BLAKE3 hex, identical to the CLI and Go server).
      const checksum = await hashBlob(archive)
      return {
        archive,
        checksum,
        cleanup: async () => {
          // Best effort: a leftover temp file is harmless and the browser
          // reclaims OPFS storage on its own.
          try {
            await root.removeEntry(tmpName)
          } catch {
            // ignore
          }
        },
      }
    } catch {
      try {
        await root.removeEntry(tmpName)
      } catch {
        // ignore
      }
      // Fall through to the in-memory path below if OPFS staging failed.
    }
  }

  // Fallback (no OPFS, or OPFS staging failed): collect the stream into one Blob.
  // Each input file is still streamed in turn, so peak memory is the finished
  // archive rather than every source file at once, and browsers spill large
  // blobs to disk on their own.
  const archive = await new Response(zipStream()).blob()
  const checksum = await hashBlob(archive)
  return { archive, checksum, cleanup: noopCleanup }
}

function chunkedStream(blob: Blob, chunkSize: number, onChunk: (bytesRead: number) => void): ReadableStream<Uint8Array> {
  let pos = 0
  return new ReadableStream({
    async pull(controller) {
      if (pos >= blob.size) {
        controller.close()
        return
      }
      const end = Math.min(pos + chunkSize, blob.size)
      const slice = blob.slice(pos, end)
      const buf = await slice.arrayBuffer()
      controller.enqueue(new Uint8Array(buf))
      const bytesRead = end - pos
      pos = end
      onChunk(bytesRead)
    },
  })
}

// resolveShareCode: Fetch the share code the server assigned to a finished tus
// upload. The tus protocol's final response carries no body, so the code is
// pulled in a short follow-up request keyed by the upload id (the last path
// segment of the upload URL). The headers carry the same identity (uploader id
// or Bearer token) that authorized the upload.
async function resolveShareCode(
  uploadUrl: string,
  headers: Record<string, string>,
  signal?: AbortSignal
): Promise<string> {
  const id = uploadUrl.replace(/\/+$/, "").split("/").pop() ?? ""
  const url = apiUrl("/upload-result")
  url.searchParams.set("id", id)

  const res = await fetch(url.toString(), { method: "GET", headers, signal })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
  return (await res.text()).trim()
}

// uploadArchiveViaTus: Stream one archive to the server with the tus resumable
// protocol, chunked so the browser never holds the whole upload in a single
// request. Every per-upload setting travels as tus metadata (the server reads it
// back when finalizing); identity travels in `headers`. Resolves with the bare
// share code. Shared by the public upload and group uploads.
function uploadArchiveViaTus(
  archive: Blob,
  filename: string,
  metadata: Record<string, string>,
  headers: Record<string, string>,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal
): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    const upload = new tus.Upload(archive, {
      endpoint: apiUrl("/upload/").toString(),
      chunkSize: TUS_CHUNK_SIZE,
      retryDelays: [0, 1000, 3000, 5000],
      removeFingerprintOnSuccess: true,
      // tus-js-client base64-encodes metadata values (UTF-8 safe), so any text
      // (e.g. a unicode message) travels intact without manual encoding.
      metadata: { filename, ...metadata },
      headers,
      onProgress: (bytesSent, bytesTotal) => {
        if (onProgress) onProgress({ loaded: bytesSent, total: bytesTotal })
      },
      onError: (err) => reject(err instanceof Error ? err : new ApiError(0, String(err))),
      onSuccess: () => {
        resolveShareCode(upload.url ?? "", headers, signal).then(resolve).catch(reject)
      },
    })

    signal?.addEventListener("abort", () => {
      void upload.abort()
      reject(new DOMException("Aborted", "AbortError"))
    })

    upload.start()
  })
}

export async function uploadFile(
  items: Upload[],
  expiration: string,
  maxDownloadCount: number,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal,
  password?: string,
  message?: string
): Promise<string> {
  // The server requires a known uploader (X-Flick-User-ID), exactly like the CLI.
  const uploaderId = await ensureUploaderId(signal)

  const zipTotal = items.reduce((sum, i) => sum + i.entries.reduce((s, e) => s + e.file.size, 0), 0)
  const { archive, checksum: archiveChecksum, cleanup } = await buildUploadArchive(items, onProgress ? (loaded) => {
    onProgress({ loaded, total: zipTotal, phase: "zipping" })
  } : undefined)
  try {
    const metadata: Record<string, string> = {
      checksum: archiveChecksum,
      encrypted: "false",
      expiration,
      maxDownloadCount: String(maxDownloadCount),
    }
    // An empty password leaves the code public; the server treats it as unset.
    if (password) metadata.password = password
    // Optional personal note surfaced to the downloader on the receive page.
    if (message) metadata.message = message

    return await uploadArchiveViaTus(
      archive,
      randomArchiveName(),
      metadata,
      { "X-Flick-User-ID": uploaderId },
      onProgress ? (p) => onProgress({ ...p, phase: "uploading" }) : undefined,
      signal
    )
  } finally {
    await cleanup()
  }
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
export async function downloadByCode(
  code: string,
  signal?: AbortSignal,
  password?: string
): Promise<DownloadedArchive[]> {
  const url = apiUrl("/download")
  url.searchParams.set("code", code)

  const headers: HeadersInit = {}
  if (password) headers["X-Flick-Password"] = password

  const res = await fetch(url.toString(), { method: "GET", signal, headers })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (res.status === 401) throw new PasswordRequiredError(code)
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
  // True while the code is locked: a password guards it and none has been
  // supplied yet, so items is a placeholder and the real listing stays withheld.
  // The receive page prompts for the password and sends it on the download.
  passwordProtected: boolean
  // Optional personal note the uploader attached, shown to the downloader. Empty
  // when no message was set.
  message: string
}

// fetchDownloadInfo: List the items behind a code WITHOUT consuming a download.
// The receive page uses this on load so merely opening the page never burns the
// single-use code; the real (consuming) transfer happens later via downloadByCode.
export async function fetchDownloadInfo(code: string, signal?: AbortSignal, password?: string): Promise<DownloadInfo> {
  const url = apiUrl("/download/info")
  url.searchParams.set("code", code)

  const headers: HeadersInit = {}
  if (password) headers["X-Flick-Password"] = password

  const res = await fetch(url.toString(), { method: "GET", signal, headers })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (!res.ok) throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))

  const data = (await res.json()) as {
    items?: DownloadInfoItem[]
    encrypted?: boolean
    passwordProtected?: boolean
    message?: string
  }
  return {
    items: data.items ?? [],
    encrypted: data.encrypted === true,
    passwordProtected: data.passwordProtected === true,
    message: typeof data.message === "string" ? data.message : "",
  }
}

export async function loadUserConfiguration(signal?: AbortSignal): Promise<Record<string, string | number | boolean>> {
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

export async function loadConfiguration(signal?: AbortSignal): Promise<Record<string, string | number | boolean>> {
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
  maxExpiration: string
}

// Convert a duration string like "30m", "4h", "7d" to total minutes.
export function parseDurationMinutes(duration: string): number {
  const match = duration.match(/^(\d+)([mhd])$/)
  if (!match) return 0
  const val = Number.parseInt(match[1], 10)
  switch (match[2]) {
    case "d": return val * 1440
    case "h": return val * 60
    case "m": return val
    default: return 0
  }
}

export async function fetchServerLimits(signal?: AbortSignal): Promise<ServerLimits> {
  const conf = await loadUserConfiguration(signal)
  const def = typeof conf.default_download_count === "number" ? conf.default_download_count : 1
  const max = typeof conf.max_download_count === "number" ? conf.max_download_count : def
  const allowMultiple = conf.allow_multiple_downloads === true
  const maxFileSizeMb = typeof conf.max_file_size_mb === "number" ? conf.max_file_size_mb : 1000
  const maxExpiration = typeof conf.max_expiration === "string" ? conf.max_expiration : "4h"
  return { default: def, max: Math.max(max, def), allowMultiple, maxFileSizeMb, maxExpiration }
}

export interface QuotaUsage {
  usedBytes: number
  limitMb: number
}

// fetchQuota: Read the current storage usage for the visitor, scoped to their
// uploader id (anonymous or signed-in), exactly like the upload request. The
// usage is precise in bytes; the limit stays in megabytes, as configured
// server-side. A limit of 0 means unlimited.
export async function fetchQuota(signal?: AbortSignal): Promise<QuotaUsage> {
  const uploaderId = await ensureUploaderId(signal)
  const url = apiUrl("/quota")

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { "X-Flick-User-ID": uploaderId },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as { usedBytes?: number; limitMb?: number }
  return {
    usedBytes: typeof data.usedBytes === "number" ? data.usedBytes : 0,
    limitMb: typeof data.limitMb === "number" ? data.limitMb : 0,
  }
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
  const timestamp = typeof obj.timestamp === "string" ? obj.timestamp : new Date().toISOString()

  return { timestamp, activeCodes, totalUploads, totalDownloads, userCount, storageBytes }
}

// Global account role, mirrors the API's user_role enum.
export type UserRole = "admin" | "user"

// Role a user holds inside a group, mirrors the API's group_role enum. Carried
// on the session so the dashboard can show maintainers their group view. The API
// does not expose group memberships yet, so this stays undefined for now.
export type GroupRole = "member" | "maintainer" | "owner"

// A group the signed-in user belongs to, with their role inside it. Carried on
// the session so the dashboard can show a member their groups and gate the
// "My groups" tab.
export interface GroupMembership {
  id: string
  name: string
  role: GroupRole
}

export interface AuthUser {
  id: string
  username: string
  email: string
  role: UserRole
  groups: GroupMembership[]
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

// parseGroupMemberships: Maps the raw `groups` array from a login/whoami user
// payload (snake_case) to GroupMembership[]. Tolerates a missing/invalid value
// by returning an empty array.
function parseGroupMemberships(value: unknown): GroupMembership[] {
  if (!Array.isArray(value)) return []
  return value.map((raw) => {
    const obj = (raw ?? {}) as Record<string, unknown>
    return {
      id: typeof obj.id === "string" ? obj.id : "",
      name: typeof obj.name === "string" ? obj.name : "",
      role: coerceGroupRole(obj.role),
    }
  })
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
    groups: [],
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

  // The server replies with { token, expires_at, user: { id, username, email, role, blocked, created_at, groups } }.
  const data = (await res.json()) as {
    token?: string
    user?: {
      id?: string
      username?: string
      email?: string
      role?: unknown
      blocked?: unknown
      created_at?: string
      groups?: unknown
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
      groups: parseGroupMemberships(data.user.groups),
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
      groups?: unknown
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
    groups: parseGroupMemberships(data.user.groups),
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

export interface AdminGroup {
  id: string
  name: string
  createdAt?: string
}

// toAdminGroup: Maps a raw API group object (snake_case) to AdminGroup.
function toAdminGroup(raw: unknown): AdminGroup {
  const obj = (raw ?? {}) as Record<string, unknown>
  return {
    id: typeof obj.id === "string" ? obj.id : "",
    name: typeof obj.name === "string" ? obj.name : "",
    createdAt: typeof obj.created_at === "string" ? obj.created_at : undefined,
  }
}

// listGroups: Admin-only fetch of every group. Requires an admin session token.
export async function listGroups(token: string, signal?: AbortSignal): Promise<AdminGroup[]> {
  const url = apiUrl("/admin/groups")

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
    throw new ApiError(res.status, "Invalid groups response")
  }
  return data.map(toAdminGroup)
}

// createGroup: Admin-only creation of a group. The API replies 201 with the
// created group { id, name, created_at }.
export async function createGroup(token: string, name: string, signal?: AbortSignal): Promise<AdminGroup> {
  const url = apiUrl("/admin/groups")

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  return toAdminGroup(await res.json())
}

// deleteGroup: Admin-only deletion of a group by id. The API replies 204 with
// no body on success.
export async function deleteGroup(token: string, id: string, signal?: AbortSignal): Promise<void> {
  const url = apiUrl(`/admin/groups/${id}`)

  const res = await fetch(url.toString(), {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// renameGroup: Admin-only rename of a group. The API replies 200 with the
// updated group.
export async function renameGroup(token: string, id: string, name: string, signal?: AbortSignal): Promise<AdminGroup> {
  const url = apiUrl(`/admin/groups/${id}`)

  const res = await fetch(url.toString(), {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  return toAdminGroup(await res.json())
}

// A member of a group: a user plus the role they hold inside that group.
export interface GroupMember {
  id: string
  username: string
  email: string
  role: UserRole
  blocked: boolean
  createdAt?: string
  groupRole: GroupRole
}

// coerceGroupRole: Defensive coercion of an unknown group role from the API.
export function coerceGroupRole(value: unknown): GroupRole {
  return value === "owner" ? "owner" : value === "maintainer" ? "maintainer" : "member"
}

// toGroupMember: Maps a raw API member object (snake_case) to GroupMember.
function toGroupMember(raw: unknown): GroupMember {
  const obj = (raw ?? {}) as Record<string, unknown>
  return {
    id: typeof obj.id === "string" ? obj.id : "",
    username: typeof obj.username === "string" ? obj.username : "",
    email: typeof obj.email === "string" ? obj.email : "",
    role: coerceRole(obj.role),
    blocked: obj.blocked === true,
    createdAt: typeof obj.created_at === "string" ? obj.created_at : undefined,
    groupRole: coerceGroupRole(obj.group_role),
  }
}

// listGroupMembers: Fetch the members of a group, with each member's role inside
// the group. Allowed for a global admin or a maintainer/owner of the group.
export async function listGroupMembers(token: string, groupId: string, signal?: AbortSignal): Promise<GroupMember[]> {
  const url = apiUrl(`/admin/groups/${groupId}/members`)

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
    throw new ApiError(res.status, "Invalid members response")
  }
  return data.map(toGroupMember)
}

// addGroupMember: Add a user to a group. The API replies 204 with no body.
export async function addGroupMember(
  token: string,
  groupId: string,
  userId: string,
  signal?: AbortSignal
): Promise<void> {
  const url = apiUrl(`/admin/groups/${groupId}/members`)

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ user_id: userId }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// removeGroupMember: Remove a user from a group. The API replies 204 with no body.
export async function removeGroupMember(
  token: string,
  groupId: string,
  userId: string,
  signal?: AbortSignal
): Promise<void> {
  const url = apiUrl(`/admin/groups/${groupId}/members/${userId}`)

  const res = await fetch(url.toString(), {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// setMemberRole: Change a member's role inside a group (admin-only). The API
// replies 204 with no body.
export async function setMemberRole(
  token: string,
  groupId: string,
  userId: string,
  role: GroupRole,
  signal?: AbortSignal
): Promise<void> {
  const url = apiUrl(`/admin/groups/${groupId}/members/${userId}`)

  const res = await fetch(url.toString(), {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ role }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// A minimal user match returned by the search endpoint.
export interface UserSearchResult {
  id: string
  username: string
  email: string
}

// searchUsers: Find users by username or email. Available to any authenticated
// user (used by group maintainers to pick someone to add). Returns an empty list
// for queries shorter than two characters.
export async function searchUsers(token: string, q: string, signal?: AbortSignal): Promise<UserSearchResult[]> {
  const url = apiUrl("/users/search")
  url.searchParams.set("q", q)

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
    throw new ApiError(res.status, "Invalid search response")
  }
  return data.map((raw) => {
    const obj = (raw ?? {}) as Record<string, unknown>
    return {
      id: typeof obj.id === "string" ? obj.id : "",
      username: typeof obj.username === "string" ? obj.username : "",
      email: typeof obj.email === "string" ? obj.email : "",
    }
  })
}

// A file transfer shared with a group. The code is included so members fetch
// contents and download through the native endpoints, which enforce membership
// for group-bound codes.
export interface GroupUpload {
  id: string
  code: string
  uploader: string
  createdAt?: string
}

// toGroupUpload: Maps a raw API group upload (snake_case) to GroupUpload.
function toGroupUpload(raw: unknown): GroupUpload {
  const obj = (raw ?? {}) as Record<string, unknown>
  return {
    id: typeof obj.id === "string" ? obj.id : "",
    code: typeof obj.code === "string" ? obj.code : "",
    uploader: typeof obj.uploader === "string" ? obj.uploader : "",
    createdAt: typeof obj.created_at === "string" ? obj.created_at : undefined,
  }
}

// A sub-folder inside a group's folder tree.
export interface GroupFolder {
  id: string
  name: string
}

// The contents of one folder level: its sub-folders and the transfers it holds.
export interface GroupExplore {
  folders: GroupFolder[]
  uploads: GroupUpload[]
}

// exploreGroup: List the sub-folders and transfers at a level of the group's
// folder tree (the root when folderId is omitted). Allowed for a global admin or
// any member of the group.
export async function exploreGroup(
  token: string,
  groupId: string,
  folderId?: string,
  signal?: AbortSignal
): Promise<GroupExplore> {
  const url = apiUrl(`/admin/groups/${groupId}/explore`)
  if (folderId) url.searchParams.set("folder", folderId)

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as { folders?: unknown; uploads?: unknown }
  return {
    folders: Array.isArray(data.folders)
      ? data.folders.map((raw) => {
          const obj = (raw ?? {}) as Record<string, unknown>
          return { id: typeof obj.id === "string" ? obj.id : "", name: typeof obj.name === "string" ? obj.name : "" }
        })
      : [],
    uploads: Array.isArray(data.uploads) ? data.uploads.map(toGroupUpload) : [],
  }
}

// createGroupFolder: Create a folder in the group (maintainer/owner only). An
// omitted parentId creates it at the group root.
export async function createGroupFolder(
  token: string,
  groupId: string,
  name: string,
  parentId?: string,
  signal?: AbortSignal
): Promise<GroupFolder> {
  const url = apiUrl(`/admin/groups/${groupId}/folders`)

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name, parent_id: parentId ?? "" }),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const obj = (await res.json()) as Record<string, unknown>
  return { id: typeof obj.id === "string" ? obj.id : "", name: typeof obj.name === "string" ? obj.name : "" }
}

// deleteGroupFolder: Delete a folder and its contents (maintainer/owner only).
export async function deleteGroupFolder(
  token: string,
  groupId: string,
  folderId: string,
  signal?: AbortSignal
): Promise<void> {
  const url = apiUrl(`/admin/groups/${groupId}/folders/${folderId}`)

  const res = await fetch(url.toString(), {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// deleteGroupUpload: Revoke a file shared with a group (maintainer/owner only).
// The stored code (files + cache) is removed server-side too.
export async function deleteGroupUpload(
  token: string,
  groupId: string,
  uploadId: string,
  signal?: AbortSignal
): Promise<void> {
  const url = apiUrl(`/admin/groups/${groupId}/uploads/${uploadId}`)

  const res = await fetch(url.toString(), {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }
}

// groupUploadInfo: List a group transfer's contents through the native info
// endpoint (membership is enforced via the Bearer token for group-bound codes).
// Lets the explorer show the real file names instead of the share code.
export async function groupUploadInfo(token: string, code: string, signal?: AbortSignal): Promise<DownloadInfo> {
  const url = apiUrl("/download/info")
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const data = (await res.json()) as {
    items?: DownloadInfoItem[]
    encrypted?: boolean
    passwordProtected?: boolean
    message?: string
  }
  return {
    items: data.items ?? [],
    encrypted: data.encrypted === true,
    passwordProtected: data.passwordProtected === true,
    message: typeof data.message === "string" ? data.message : "",
  }
}

// uploadToGroup: Share files with a group (maintainer/owner only) through the
// native upload endpoint bound to the group via `group_id`. The transfer is
// private to the group and downloadable by every member.
export async function uploadToGroup(
  token: string,
  groupId: string,
  items: Upload[],
  expiration: string,
  folderId?: string,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal
): Promise<void> {
  const zipTotal = items.reduce((sum, i) => sum + i.entries.reduce((s, e) => s + e.file.size, 0), 0)
  const { archive, checksum, cleanup } = await buildUploadArchive(items, onProgress ? (loaded) => {
    onProgress({ loaded, total: zipTotal, phase: "zipping" })
  } : undefined)
  try {
    const metadata: Record<string, string> = {
      checksum,
      encrypted: "false",
      expiration,
      groupId,
    }
    if (folderId) metadata.folderId = folderId

    await uploadArchiveViaTus(
      archive,
      randomArchiveName(),
      metadata,
      { Authorization: `Bearer ${token}` },
      onProgress ? (p) => onProgress({ ...p, phase: "uploading" }) : undefined,
      signal
    )
  } finally {
    await cleanup()
  }
}

// downloadGroupUpload: Fetch a group transfer's archive(s) through the native
// download endpoint. The code is group-bound, so the server enforces membership
// (the Bearer token) before serving. Verifies the announced checksum like
// downloadByCode.
export async function downloadGroupUpload(
  token: string,
  code: string,
  signal?: AbortSignal
): Promise<DownloadedArchive[]> {
  const url = apiUrl("/download")
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, parseErrorMessage(await res.text().catch(() => ""), res.statusText))
  }

  const expectedChecksum = res.headers.get("X-Flick-Checksum")

  const form = await res.formData()
  const archives: DownloadedArchive[] = []
  for (const value of form.getAll("file")) {
    if (!(value instanceof File)) continue

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
