const API_PORT = 15702

function getApiBase(): string {
  if (typeof window !== "undefined") {
    return `https://${window.location.hostname}:${API_PORT}`
  }
  return `https://localhost:${API_PORT}`
}

export function getConfigureUrl(): string {
  return new URL("/configure", getApiBase()).toString()
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
  const url = new URL("/upload", getApiBase())
  url.searchParams.set("expiration", expiration)
  url.searchParams.set("maxDownloadCount", String(maxDownloadCount))

  const form = new FormData()
  form.append("file", file)

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
        reject(new ApiError(xhr.status, xhr.responseText || xhr.statusText))
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
  const url = new URL("/download", getApiBase())
  url.searchParams.set("code", code)

  const res = await fetch(url.toString(), { method: "GET", signal })

  if (res.status === 404) throw new CodeNotFoundError(code)
  if (!res.ok) throw new ApiError(res.status, await res.text().catch(() => res.statusText))

  const form = await res.formData()
  const files: DownloadedFile[] = []

  for (const value of form.getAll("file")) {
    if (value instanceof File) {
      files.push({ name: value.name, blob: value, size: value.size })
    }
  }

  return files
}

export async function loadConfiguration(
  signal?: AbortSignal
): Promise<Record<string, string | number | boolean>> {
  const url = new URL("/configure", getApiBase())

  const res = await fetch(url.toString(), { method: "GET", signal })
  if (!res.ok) {
    throw new ApiError(res.status, await res.text().catch(() => res.statusText))
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
  const url = new URL("/configure", getApiBase())

  const res = await fetch(url.toString(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(values),
    signal,
  })
  if (!res.ok) {
    throw new ApiError(res.status, await res.text().catch(() => res.statusText))
  }
}

export interface DownloadCountLimits {
  default: number
  max: number
  allowMultiple: boolean
}

export async function fetchDownloadCountLimits(signal?: AbortSignal): Promise<DownloadCountLimits> {
  const conf = await loadConfiguration(signal)
  const def = typeof conf.default_download_count === "number" ? conf.default_download_count : 1
  const max = typeof conf.max_download_count === "number" ? conf.max_download_count : def
  const allowMultiple = conf.allow_multiple_downloads === true
  return { default: def, max: Math.max(max, def), allowMultiple }
}

export interface StatsSnapshot {
  timestamp: string
  activeCodes: number
  totalUploads: number
  totalDownloads: number
}

export async function fetchStats(signal?: AbortSignal): Promise<StatsSnapshot> {
  const url = new URL("/stats", getApiBase())

  const res = await fetch(url.toString(), { method: "GET", signal })
  if (!res.ok) {
    throw new ApiError(res.status, await res.text().catch(() => res.statusText))
  }

  const data = (await res.json()) as unknown
  if (data === null || typeof data !== "object") {
    throw new ApiError(res.status, "Invalid stats response")
  }

  const obj = data as Record<string, unknown>
  const activeCodes = typeof obj.activeCodes === "number" ? obj.activeCodes : 0
  const totalUploads = typeof obj.totalUploads === "number" ? obj.totalUploads : 0
  const totalDownloads = typeof obj.totalDownloads === "number" ? obj.totalDownloads : 0
  const timestamp =
    typeof obj.timestamp === "string" ? obj.timestamp : new Date().toISOString()

  return { timestamp, activeCodes, totalUploads, totalDownloads }
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
