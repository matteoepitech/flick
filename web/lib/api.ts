const API_PORT = 15702

function getApiBase(): string {
  if (typeof window !== "undefined") {
    return `https://${window.location.hostname}:${API_PORT}`
  }
  return `https://localhost:${API_PORT}`
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
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal
): Promise<string> {
  const url = new URL("/upload", getApiBase())
  url.searchParams.set("expiration", expiration)

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
