"use client"

import { ArrowUpRight, ChevronLeft, FileText, Folder, Upload, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useRef, useState, type ChangeEvent, type DragEvent, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { ApiError, fetchServerLimits, uploadFile, type UploadEntry } from "@/lib/api"
import { Link, useRouter } from "@/i18n/navigation"
import { cn } from "@/lib/utils"

type Expiration = "1h" | "2h" | "3h" | "4h"

// UploadItem: one entry in the staging list. A loose file is a single-entry
// item; a folder keeps every file with its relative path so it can be zipped
// with its structure intact, just like the CLI.
interface UploadItem {
  id: string
  name: string
  isFolder: boolean
  entries: UploadEntry[]
  size: number
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 o"
  const units = ["o", "Ko", "Mo", "Go"]
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** i).toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

let itemCounter = 0
function nextId(): string {
  itemCounter += 1
  return `item-${itemCounter}`
}

function fileItem(file: File): UploadItem {
  // A folder picked through <input webkitdirectory> yields files carrying a
  // webkitRelativePath; treat those as a folder item grouped below instead.
  return { id: nextId(), name: file.name, isFolder: false, entries: [{ path: file.name, file }], size: file.size }
}

function folderItem(name: string, entries: UploadEntry[]): UploadItem {
  const size = entries.reduce((total, entry) => total + entry.file.size, 0)
  return { id: nextId(), name, isFolder: true, entries, size }
}

// fileFromEntry / walkDirectory: read a dropped FileSystemEntry tree into flat
// UploadEntry list, preserving each file's path relative to the dropped folder.
function fileFromEntry(entry: FileSystemFileEntry): Promise<File> {
  return new Promise((resolve, reject) => entry.file(resolve, reject))
}

async function walkDirectory(dir: FileSystemDirectoryEntry, prefix: string, out: UploadEntry[]): Promise<void> {
  const reader = dir.createReader()
  const readBatch = () =>
    new Promise<FileSystemEntry[]>((resolve, reject) => reader.readEntries(resolve, reject))

  // readEntries returns the directory in batches; keep reading until it drains.
  for (let batch = await readBatch(); batch.length > 0; batch = await readBatch()) {
    for (const child of batch) {
      const childPath = `${prefix}/${child.name}`
      if (child.isFile) {
        const file = await fileFromEntry(child as FileSystemFileEntry)
        out.push({ path: childPath, file })
      } else if (child.isDirectory) {
        await walkDirectory(child as FileSystemDirectoryEntry, childPath, out)
      }
    }
  }
}

async function itemsFromDataTransfer(list: DataTransferItemList): Promise<UploadItem[]> {
  // webkitGetAsEntry() must be called synchronously while the event is live, so
  // collect every entry first, then traverse the directories asynchronously.
  const entries: FileSystemEntry[] = []
  for (let i = 0; i < list.length; i++) {
    const entry = list[i].webkitGetAsEntry()
    if (entry) entries.push(entry)
  }

  const items: UploadItem[] = []
  for (const entry of entries) {
    if (entry.isFile) {
      const file = await fileFromEntry(entry as FileSystemFileEntry)
      items.push(fileItem(file))
    } else if (entry.isDirectory) {
      const collected: UploadEntry[] = []
      await walkDirectory(entry as FileSystemDirectoryEntry, entry.name, collected)
      if (collected.length > 0) items.push(folderItem(entry.name, collected))
    }
  }
  return items
}

export default function SendPage() {
  const t = useTranslations("Send")
  const router = useRouter()
  const inputRef = useRef<HTMLInputElement>(null)
  const folderInputRef = useRef<HTMLInputElement>(null)

  const [items, setItems] = useState<UploadItem[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const [expiration, setExpiration] = useState<Expiration>("1h")
  const [maxDownloadCount, setMaxDownloadCount] = useState<number>(1)
  const [maxDownloadLimit, setMaxDownloadLimit] = useState<number>(1)
  const [allowMultipleDownloads, setAllowMultipleDownloads] = useState<boolean>(false)
  const [submitting, setSubmitting] = useState(false)
  const [progress, setProgress] = useState<{ name: string; percent: number } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [maxFileSize, setMaxFileSize] = useState<number>(1000 * 1024 * 1024)

  useEffect(() => {
    const controller = new AbortController()
    fetchServerLimits(controller.signal)
      .then(({ default: def, max, allowMultiple, maxFileSizeMb }) => {
        setAllowMultipleDownloads(allowMultiple)
        setMaxDownloadLimit(max)
        setMaxDownloadCount(allowMultiple ? def : 1)
        if (maxFileSizeMb > 0) setMaxFileSize(maxFileSizeMb * 1024 * 1024)
      })
      .catch(() => {})
    return () => controller.abort()
  }, [])

  // webkitdirectory / directory are non-standard attributes React won't type, so
  // set them imperatively to let the second input pick a whole folder.
  useEffect(() => {
    const input = folderInputRef.current
    if (input) {
      input.setAttribute("webkitdirectory", "")
      input.setAttribute("directory", "")
    }
  }, [])

  function addItems(incoming: UploadItem[]) {
    // No cap on the number of files: everything is zipped into a single archive,
    // so what bounds the browser's memory is the total byte size, not the count.
    // We guard the cumulative size against the server's max upload size, which is
    // exactly what that one archive must fit under anyway.
    let running = items.reduce((total, item) => total + item.size, 0)
    let rejected = false
    const valid: UploadItem[] = []

    for (const item of incoming) {
      if (running + item.size > maxFileSize) {
        setError(t("totalTooLarge", { size: formatBytes(maxFileSize) }))
        rejected = true
        break
      }
      running += item.size
      valid.push(item)
    }

    if (valid.length > 0 && !rejected) {
      setError(null) // clear possible previous error if we successfully add items without new errors
    }

    if (valid.length > 0) {
      setItems((current) => [...current, ...valid])
    }
  }

  function handleBrowseChange(event: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(event.target.files ?? [])
    if (files.length > 0) addItems(files.map(fileItem))
    event.target.value = ""
  }

  function handleFolderChange(event: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(event.target.files ?? [])
    if (files.length > 0) {
      // Every file from a directory input carries its webkitRelativePath, e.g.
      // "myfolder/sub/a.txt"; the first segment is the folder name.
      const top = files[0].webkitRelativePath.split("/")[0] || "folder"
      const entries: UploadEntry[] = files.map((file) => ({
        path: file.webkitRelativePath || file.name,
        file,
      }))
      addItems([folderItem(top, entries)])
    }
    event.target.value = ""
  }

  async function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault()
    setIsDragging(false)

    const list = event.dataTransfer.items
    if (list && list.length > 0 && typeof list[0].webkitGetAsEntry === "function") {
      // Entry API lets us walk dropped folders; fall back to flat files below.
      addItems(await itemsFromDataTransfer(list))
    } else if (event.dataTransfer.files.length > 0) {
      addItems(Array.from(event.dataTransfer.files).map(fileItem))
    }
  }

  function removeItem(id: string) {
    setItems((current) => current.filter((item) => item.id !== id))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (items.length === 0) return
    setSubmitting(true)
    setError(null)

    try {
      // Every staged item is packed into one combined archive under a single
      // code; folders keep their structure, loose files sit at the archive root.
      const uploads = items.map((item) => ({ name: item.name, isFolder: item.isFolder, entries: item.entries }))
      const label = items.length === 1 ? items[0].name : t("batchLabel", { count: items.length })

      const code = await uploadFile(uploads, expiration, maxDownloadCount, ({ loaded, total }) => {
        setProgress({ name: label, percent: Math.round((loaded / total) * 100) })
      })

      const params = new URLSearchParams()
      params.set("code", code)
      params.set("exp", expiration)
      router.push(`/send/success?${params.toString()}`)
    } catch (err) {
      console.error(err)
      setError(err instanceof ApiError ? err.message : t("uploadError"))
      setSubmitting(false)
      setProgress(null)
    }
  }

  const expirations: { value: Expiration; label: string }[] = [
    { value: "1h", label: t("expiration1h") },
    { value: "2h", label: t("expiration2h") },
    { value: "3h", label: t("expiration3h") },
    { value: "4h", label: t("expiration4h") },
  ]

  const canSubmit = items.length > 0 && !submitting

  return (
    <main className="mx-auto flex w-full max-w-2xl flex-col items-center px-6 py-16">
      <Link
        href="/"
        className="mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("back")}
      </Link>

      <div className="w-full text-center">
        <h1 className="text-3xl font-bold tracking-tight md:text-4xl">{t("title")}</h1>
        <p className="mt-3 text-base text-muted-foreground">{t("description")}</p>
      </div>

      <form onSubmit={handleSubmit} className="mt-10 flex w-full flex-col gap-8 text-left">
        <div
          onDragEnter={(event) => {
            event.preventDefault()
            setIsDragging(true)
          }}
          onDragOver={(event) => event.preventDefault()}
          onDragLeave={(event) => {
            event.preventDefault()
            if (event.currentTarget.contains(event.relatedTarget as Node)) return
            setIsDragging(false)
          }}
          onDrop={handleDrop}
          className={cn(
            "relative flex flex-col items-center justify-center gap-4 rounded-2xl border-2 border-dashed border-border bg-card/40 px-6 py-14 text-center transition-colors",
            isDragging && "border-primary bg-primary/5"
          )}
        >
          <input ref={inputRef} type="file" multiple className="hidden" onChange={handleBrowseChange} />
          <input ref={folderInputRef} type="file" multiple className="hidden" onChange={handleFolderChange} />

          <span className="flex h-14 w-14 items-center justify-center rounded-xl bg-muted text-muted-foreground">
            <Upload className="h-6 w-6" />
          </span>

          <div>
            <p className="text-base font-semibold text-foreground">{t("dropTitle")}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("dropOr")}{" "}
              <button
                type="button"
                onClick={() => inputRef.current?.click()}
                className="font-medium text-primary underline-offset-2 hover:underline"
              >
                {t("dropBrowse")}
              </button>{" "}
              {t("dropOrFolder")}{" "}
              <button
                type="button"
                onClick={() => folderInputRef.current?.click()}
                className="font-medium text-primary underline-offset-2 hover:underline"
              >
                {t("dropBrowseFolder")}
              </button>{" "}
              · {t("dropLimits", { maxSize: formatBytes(maxFileSize) })}
            </p>
          </div>
        </div>

        {items.length > 0 && (
          <Card className="p-2">
            <ul className="flex flex-col">
              {items.map((item) => (
                <li
                  key={item.id}
                  className="flex items-center gap-3 rounded-lg px-3 py-2.5 hover:bg-muted/50"
                >
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                    {item.isFolder ? <Folder className="h-4 w-4" /> : <FileText className="h-4 w-4" />}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{item.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {item.isFolder
                        ? t("folderMeta", { count: item.entries.length, size: formatBytes(item.size) })
                        : formatBytes(item.size)}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeItem(item.id)}
                    aria-label={t("remove")}
                    className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </li>
              ))}
            </ul>
          </Card>
        )}

        <Card className="gap-6 p-6">
          <p className="text-xs font-semibold tracking-widest text-muted-foreground uppercase">{t("options")}</p>

          <div className="flex flex-col gap-3">
            <Label className="text-sm font-semibold text-foreground">{t("expiration")}</Label>
            <div className="flex flex-wrap gap-2">
              {expirations.map((option) => {
                const active = expiration === option.value
                return (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() => setExpiration(option.value)}
                    className={cn(
                      "rounded-lg border px-4 py-2 text-sm font-medium transition-colors",
                      active
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border bg-background text-foreground hover:bg-muted"
                    )}
                  >
                    {option.label}
                  </button>
                )
              })}
            </div>
          </div>

          {allowMultipleDownloads && (
            <div className="flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <Label htmlFor="maxDownloadCount" className="text-sm font-semibold text-foreground">
                  {t("maxDownloadCount")}
                </Label>
                <span className="text-sm font-semibold text-primary tabular-nums">{maxDownloadCount}</span>
              </div>
              <input
                id="maxDownloadCount"
                type="range"
                min={1}
                max={maxDownloadLimit}
                step={1}
                value={maxDownloadCount}
                onChange={(event) => setMaxDownloadCount(Number(event.target.value))}
                className="h-1.5 w-full cursor-pointer appearance-none rounded-full bg-muted accent-primary"
              />
              <div className="flex justify-between text-xs text-muted-foreground tabular-nums">
                <span>1</span>
                <span>{maxDownloadLimit}</span>
              </div>
            </div>
          )}

          <div className="flex flex-col gap-3 opacity-60">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="flex items-center gap-2">
                  <p className="text-sm font-semibold text-foreground">{t("passwordTitle")}</p>
                  <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
                    {t("comingSoon")}
                  </span>
                </div>
                <p className="text-sm text-muted-foreground">{t("passwordDescription")}</p>
              </div>
              <Switch checked={false} disabled aria-disabled />
            </div>
          </div>

          <div className="flex flex-col gap-2 opacity-60">
            <div className="flex items-center gap-2">
              <Label htmlFor="message" className="text-sm font-semibold text-foreground">
                {t("messageLabel")}
              </Label>
              <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
                {t("comingSoon")}
              </span>
            </div>
            <Textarea id="message" value="" disabled placeholder={t("messagePlaceholder")} rows={4} />
          </div>
        </Card>

        {progress && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between text-sm text-muted-foreground">
              <span className="truncate">{progress.name}</span>
              <span className="tabular-nums">{progress.percent}%</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-orange-500 transition-all duration-150"
                style={{ width: `${progress.percent}%` }}
              />
            </div>
          </div>
        )}

        {error && <p className="rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">{error}</p>}

        <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button asChild type="button" variant="outline" size="lg" className="h-12 px-6">
            <Link href="/">{t("cancel")}</Link>
          </Button>
          <Button type="submit" size="lg" className="h-12 px-6" disabled={!canSubmit}>
            <ArrowUpRight className="size-5" />
            {submitting ? t("submitting") : t("submit")}
          </Button>
        </div>
      </form>
    </main>
  )
}
