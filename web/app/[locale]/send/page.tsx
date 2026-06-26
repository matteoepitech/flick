"use client"

import { ArrowUpRight, ChevronLeft, FileText, Folder, Upload, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useRef, useState, type ChangeEvent, type DragEvent, type FormEvent } from "react"

import { ExpirationPicker } from "@/components/expiration-picker"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { ApiError, fetchQuota, fetchServerLimits, parseDurationMinutes, uploadFile, type QuotaUsage } from "@/lib/api"
import {
  fileItem,
  folderItemFromInputFiles,
  formatBytes,
  itemsFromDataTransfer,
  type UploadItem,
} from "@/lib/upload-staging"
import { Link, useRouter } from "@/i18n/navigation"
import { cn } from "@/lib/utils"

export default function SendPage() {
  const t = useTranslations("Send")
  const router = useRouter()
  const inputRef = useRef<HTMLInputElement>(null)
  const folderInputRef = useRef<HTMLInputElement>(null)

  const [items, setItems] = useState<UploadItem[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const [expiration, setExpiration] = useState<string>("1h")
  const [maxDownloadCount, setMaxDownloadCount] = useState<number>(1)
  const [maxDownloadLimit, setMaxDownloadLimit] = useState<number>(1)
  const [allowMultipleDownloads, setAllowMultipleDownloads] = useState<boolean>(false)
  const [passwordEnabled, setPasswordEnabled] = useState(false)
  const [password, setPassword] = useState("")
  const [message, setMessage] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [progress, setProgress] = useState<{
    name: string
    percent: number
    phase: "zipping" | "uploading"
    speed?: string
    eta?: string
  } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [maxFileSize, setMaxFileSize] = useState<number>(1000 * 1024 * 1024)
  const [maxExpiration, setMaxExpiration] = useState<string>("4h")
  const [quota, setQuota] = useState<QuotaUsage | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    fetchServerLimits(controller.signal)
      .then(({ default: def, max, allowMultiple, maxFileSizeMb, maxExpiration: maxExp }) => {
        setAllowMultipleDownloads(allowMultiple)
        setMaxDownloadLimit(max)
        setMaxDownloadCount(allowMultiple ? def : 1)
        if (maxFileSizeMb > 0) setMaxFileSize(maxFileSizeMb * 1024 * 1024)
        setMaxExpiration(maxExp)
      })
      .catch(() => {})
    fetchQuota(controller.signal)
      .then(setQuota)
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
    if (files.length > 0) addItems([folderItemFromInputFiles(files)])
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

      const code = await uploadFile(
        uploads,
        expiration,
        maxDownloadCount,
        (uploadProgress) => {
          if (uploadProgress.phase === "zipping") {
            setProgress({
              name: t("zipping"),
              percent: Math.round((uploadProgress.loaded / uploadProgress.total) * 100),
              phase: "zipping",
            })
          } else {
            const now = Date.now()
            const deltaTime = now - speedRef.current.lastTime
            const deltaBytes = uploadProgress.loaded - speedRef.current.lastLoaded
            if (deltaTime > 200 && deltaBytes > 0) {
              const instantSpeed = (deltaBytes / deltaTime) * 1000
              speedRef.current.smoothed = speedRef.current.smoothed
                ? speedRef.current.smoothed * 0.7 + instantSpeed * 0.3
                : instantSpeed
              speedRef.current.lastTime = now
              speedRef.current.lastLoaded = uploadProgress.loaded
            }
            if (!speedRef.current.lastTime) {
              speedRef.current.lastTime = now
              speedRef.current.lastLoaded = uploadProgress.loaded
            }

            const pct = Math.round((uploadProgress.loaded / uploadProgress.total) * 100)
            const remaining = uploadProgress.total - uploadProgress.loaded
            const eta =
              remaining > 0 && speedRef.current.smoothed > 0
                ? Math.ceil(remaining / speedRef.current.smoothed)
                : undefined

            let speedStr: string | undefined
            let etaStr: string | undefined
            if (speedRef.current.smoothed > 0) {
              speedStr =
                speedRef.current.smoothed >= 1_048_576
                  ? `${(speedRef.current.smoothed / 1_048_576).toFixed(1)} MB/s`
                  : `${(speedRef.current.smoothed / 1_024).toFixed(1)} KB/s`
            }
            if (eta !== undefined) {
              etaStr = eta < 60 ? `${eta}s` : `${Math.floor(eta / 60)}m ${eta % 60}s`
            }

            setProgress({ name: label, percent: pct, phase: "uploading", speed: speedStr, eta: etaStr })
          }
        },
        undefined,
        passwordEnabled ? password : undefined,
        message.trim() ? message.trim() : undefined
      )

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

  // Quota bar: the solid fill is what is already stored, the lighter fill
  // projects the currently staged selection on top so the user sees whether it
  // will fit before sending. A limit of 0 means unlimited.
  const selectedBytes = items.reduce((total, item) => total + item.size, 0)
  const quotaLimitBytes = quota && quota.limitMb > 0 ? quota.limitMb * 1024 * 1024 : 0
  const usedPct = quotaLimitBytes > 0 ? Math.min(100, (quota!.usedBytes / quotaLimitBytes) * 100) : 0
  const projectedPct =
    quotaLimitBytes > 0 ? Math.min(100, ((quota!.usedBytes + selectedBytes) / quotaLimitBytes) * 100) : 0
  const overQuota = quotaLimitBytes > 0 && quota!.usedBytes + selectedBytes > quotaLimitBytes

  const speedRef = useRef({ lastTime: 0, lastLoaded: 0, smoothed: 0 })

  const overMaxExpiration = expiration.length > 0 && parseDurationMinutes(expiration) > parseDurationMinutes(maxExpiration)

  // When password protection is on, an empty password would silently produce a
  // public code, so block submission until one is typed.
  const canSubmit =
    items.length > 0 && !submitting && !overQuota && !overMaxExpiration && (!passwordEnabled || password.trim().length > 0)

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

        {quota && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between text-sm">
              <span className="font-semibold text-foreground">{t("quotaTitle")}</span>
              <span className={cn("tabular-nums text-muted-foreground", overQuota && "text-destructive")}>
                {quota.limitMb > 0
                  ? t("quotaUsage", { used: formatBytes(quota.usedBytes), limit: formatBytes(quotaLimitBytes) })
                  : t("quotaUnlimited", { used: formatBytes(quota.usedBytes) })}
              </span>
            </div>
            {quota.limitMb > 0 && (
              <div className="relative h-2 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className={cn(
                    "absolute inset-y-0 left-0 rounded-full bg-orange-500/40",
                    overQuota && "bg-destructive/40"
                  )}
                  style={{ width: `${projectedPct}%` }}
                />
                <div
                  className={cn("absolute inset-y-0 left-0 rounded-full bg-orange-500", overQuota && "bg-destructive")}
                  style={{ width: `${usedPct}%` }}
                />
              </div>
            )}
            {overQuota && <p className="text-xs text-destructive">{t("quotaOver")}</p>}
          </div>
        )}

        {items.length > 0 && (
          <Card className="p-2">
            <ul className="flex flex-col">
              {items.map((item) => (
                <li key={item.id} className="flex items-center gap-3 rounded-lg px-3 py-2.5 hover:bg-muted/50">
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
            <ExpirationPicker value={expiration} onChange={setExpiration} maxExpiration={maxExpiration} />
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

          <div className="flex flex-col gap-3">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-sm font-semibold text-foreground">{t("passwordTitle")}</p>
                <p className="text-sm text-muted-foreground">{t("passwordDescription")}</p>
              </div>
              <Switch
                checked={passwordEnabled}
                onCheckedChange={(checked) => {
                  setPasswordEnabled(checked)
                  if (!checked) setPassword("")
                }}
                aria-label={t("passwordTitle")}
              />
            </div>
            {passwordEnabled && (
              <Input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder={t("passwordPlaceholder")}
                autoComplete="new-password"
              />
            )}
          </div>

          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <Label htmlFor="message" className="text-sm font-semibold text-foreground">
                {t("messageLabel")}
              </Label>
              <span className="text-xs text-muted-foreground tabular-nums">{message.length}/500</span>
            </div>
            <Textarea
              id="message"
              value={message}
              onChange={(event) => setMessage(event.target.value.slice(0, 500))}
              maxLength={500}
              placeholder={t("messagePlaceholder")}
              rows={4}
            />
          </div>
        </Card>

        {progress && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between text-sm text-muted-foreground">
              <span className="truncate">{progress.name}</span>
              <span className="tabular-nums">{progress.percent}%</span>
            </div>
            {progress.phase === "uploading" && (progress.speed || progress.eta) && (
              <div className="flex justify-between text-xs text-muted-foreground tabular-nums">
                {progress.speed && <span>{progress.speed}</span>}
                {progress.eta && <span>ETA {progress.eta}</span>}
              </div>
            )}
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
