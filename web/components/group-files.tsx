"use client"

import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent, type DragEvent } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Download, File as FileIcon, Folder, FolderPlus, Loader2, Trash2, Upload as UploadIcon, X } from "lucide-react"

import { ErrorState } from "@/components/error-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import {
  ApiError,
  createGroupFolder,
  deleteGroupFolder,
  deleteGroupUpload,
  downloadGroupUpload,
  exploreGroup,
  groupUploadInfo,
  triggerBlobDownload,
  uploadToGroup,
  type GroupExplore,
} from "@/lib/api"
import {
  fileItem,
  folderItemFromInputFiles,
  formatBytes,
  itemsFromDataTransfer,
  type UploadItem,
} from "@/lib/upload-staging"
import { cn } from "@/lib/utils"

type Expiration = "1h" | "2h" | "3h" | "4h"
const EXPIRATIONS: Expiration[] = ["1h", "2h", "3h", "4h"]

interface GroupFilesProps {
  groupId: string
  token: string
  canManage: boolean
}

// One step in the explored path; an empty stack means the group root.
interface Crumb {
  id: string
  name: string
}

export function GroupFiles({ groupId, token, canManage }: GroupFilesProps) {
  const t = useTranslations("MyGroups")
  const locale = useLocale()

  const [path, setPath] = useState<Crumb[]>([])
  const [data, setData] = useState<GroupExplore>({ folders: [], uploads: [] })
  // Real file names per upload (resolved from /download/info), keyed by upload id.
  const [names, setNames] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [actionError, setActionError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const [newFolder, setNewFolder] = useState("")
  const [creating, setCreating] = useState(false)

  const [staged, setStaged] = useState<UploadItem[]>([])
  const [expiration, setExpiration] = useState<Expiration>("1h")
  const [sending, setSending] = useState(false)
  const [percent, setPercent] = useState<number | null>(null)
  const [isDragging, setIsDragging] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)
  const folderInputRef = useRef<HTMLInputElement>(null)

  const currentFolderId = path.length > 0 ? path[path.length - 1].id : undefined

  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    setLoadError(null)
    exploreGroup(token, groupId, currentFolderId, ctrl.signal)
      .then((res) => setData(res))
      .catch((err: unknown) => {
        if (err instanceof Error && err.name === "AbortError") return
        setLoadError(err instanceof ApiError ? err.message : t("filesLoadError"))
      })
      .finally(() => setLoading(false))
    return () => ctrl.abort()
  }, [token, groupId, currentFolderId, reloadKey, t])

  // Resolve each transfer's real file names so the list shows them rather than
  // the internal share code.
  useEffect(() => {
    if (data.uploads.length === 0) {
      setNames({})
      return
    }
    const ctrl = new AbortController()
    let cancelled = false
    Promise.all(
      data.uploads.map(async (upload) => {
        try {
          const info = await groupUploadInfo(token, upload.code, ctrl.signal)
          const label = info.items.map((it) => it.name).join(", ")
          return [upload.id, label || upload.code] as const
        } catch {
          return [upload.id, upload.code] as const
        }
      })
    ).then((entries) => {
      if (!cancelled) setNames(Object.fromEntries(entries))
    })
    return () => {
      cancelled = true
      ctrl.abort()
    }
  }, [data, token])

  useEffect(() => {
    const input = folderInputRef.current
    if (input) {
      input.setAttribute("webkitdirectory", "")
      input.setAttribute("directory", "")
    }
  }, [])

  const reload = useCallback(() => setReloadKey((k) => k + 1), [])

  const addStaged = useCallback((incoming: UploadItem[]) => {
    if (incoming.length > 0) setStaged((current) => [...current, ...incoming])
  }, [])

  function handleBrowse(event: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(event.target.files ?? [])
    if (files.length > 0) addStaged(files.map(fileItem))
    event.target.value = ""
  }

  function handleFolder(event: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(event.target.files ?? [])
    if (files.length > 0) addStaged([folderItemFromInputFiles(files)])
    event.target.value = ""
  }

  async function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault()
    setIsDragging(false)
    const list = event.dataTransfer.items
    if (list && list.length > 0 && typeof list[0].webkitGetAsEntry === "function") {
      addStaged(await itemsFromDataTransfer(list))
    } else if (event.dataTransfer.files.length > 0) {
      addStaged(Array.from(event.dataTransfer.files).map(fileItem))
    }
  }

  const removeStaged = useCallback((id: string) => {
    setStaged((current) => current.filter((item) => item.id !== id))
  }, [])

  const handleCreateFolder = useCallback(async () => {
    const name = newFolder.trim()
    if (!name) return
    setCreating(true)
    setActionError(null)
    try {
      await createGroupFolder(token, groupId, name, currentFolderId)
      setNewFolder("")
      reload()
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : t("createFolderError"))
    } finally {
      setCreating(false)
    }
  }, [newFolder, token, groupId, currentFolderId, reload, t])

  const handleDeleteFolder = useCallback(
    async (id: string) => {
      setBusyId(id)
      setActionError(null)
      try {
        await deleteGroupFolder(token, groupId, id)
        reload()
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("deleteFolderError"))
      } finally {
        setBusyId(null)
      }
    },
    [token, groupId, reload, t]
  )

  const handleDeleteUpload = useCallback(
    async (id: string) => {
      setBusyId(id)
      setActionError(null)
      try {
        await deleteGroupUpload(token, groupId, id)
        reload()
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("revokeFileError"))
      } finally {
        setBusyId(null)
      }
    },
    [token, groupId, reload, t]
  )

  const handleSend = useCallback(async () => {
    if (staged.length === 0) return
    setSending(true)
    setActionError(null)
    setPercent(0)
    try {
      const items = staged.map((item) => ({ name: item.name, isFolder: item.isFolder, entries: item.entries }))
      await uploadToGroup(token, groupId, items, expiration, currentFolderId, ({ loaded, total }) =>
        setPercent(Math.round((loaded / total) * 100))
      )
      setStaged([])
      reload()
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : t("sendError"))
    } finally {
      setSending(false)
      setPercent(null)
    }
  }, [staged, token, groupId, expiration, currentFolderId, reload, t])

  const handleDownload = useCallback(
    async (id: string, code: string) => {
      setBusyId(id)
      setActionError(null)
      try {
        const archives = await downloadGroupUpload(token, code)
        for (const archive of archives) {
          triggerBlobDownload(archive.blob, archive.name)
        }
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("downloadError"))
      } finally {
        setBusyId(null)
      }
    },
    [token, t]
  )

  const empty = useMemo(() => data.folders.length === 0 && data.uploads.length === 0, [data])

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold tracking-tight">{t("filesTitle")}</h3>
        <p className="text-sm text-muted-foreground">{t("filesSubtitle")}</p>
      </div>

      {/* Breadcrumb */}
      <div className="flex flex-wrap items-center gap-1 text-sm">
        <button
          type="button"
          className={cn("cursor-pointer hover:underline", path.length === 0 ? "font-medium" : "text-muted-foreground")}
          onClick={() => setPath([])}
        >
          {t("root")}
        </button>
        {path.map((crumb, i) => (
          <span key={crumb.id} className="flex items-center gap-1">
            <span className="text-muted-foreground">/</span>
            <button
              type="button"
              className={cn(
                "cursor-pointer hover:underline",
                i === path.length - 1 ? "font-medium" : "text-muted-foreground"
              )}
              onClick={() => setPath((p) => p.slice(0, i + 1))}
            >
              {crumb.name}
            </button>
          </span>
        ))}
      </div>

      {canManage && (
        <div className="space-y-3">
          <div className="flex items-end gap-2">
            <Input
              className="max-w-xs"
              value={newFolder}
              placeholder={t("newFolderPlaceholder")}
              disabled={creating}
              onChange={(e) => setNewFolder(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleCreateFolder()
              }}
            />
            <Button variant="outline" onClick={handleCreateFolder} disabled={creating || !newFolder.trim()}>
              {creating ? <Loader2 className="animate-spin" /> : <FolderPlus />}
              {t("createFolder")}
            </Button>
          </div>

          <div
            className={cn(
              "rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground transition-colors",
              isDragging && "border-primary bg-accent"
            )}
            onDragOver={(e) => {
              e.preventDefault()
              setIsDragging(true)
            }}
            onDragLeave={() => setIsDragging(false)}
            onDrop={handleDrop}
          >
            <UploadIcon className="mx-auto mb-2 h-5 w-5" />
            <p>{t("dropHint")}</p>
            <div className="mt-3 flex justify-center gap-2">
              <Button type="button" variant="outline" size="sm" onClick={() => fileInputRef.current?.click()}>
                {t("browseFiles")}
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={() => folderInputRef.current?.click()}>
                {t("browseFolder")}
              </Button>
            </div>
            <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleBrowse} />
            <input ref={folderInputRef} type="file" className="hidden" onChange={handleFolder} />
          </div>

          {staged.length > 0 && (
            <div className="space-y-2">
              <ul className="space-y-1">
                {staged.map((item) => (
                  <li key={item.id} className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
                    {item.isFolder ? <Folder className="h-4 w-4" /> : <FileIcon className="h-4 w-4" />}
                    <span className="truncate font-medium">{item.name}</span>
                    <span className="ml-auto text-xs text-muted-foreground">{formatBytes(item.size)}</span>
                    <button
                      type="button"
                      aria-label={t("removeStaged")}
                      className="text-muted-foreground hover:text-foreground"
                      onClick={() => removeStaged(item.id)}
                      disabled={sending}
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </li>
                ))}
              </ul>

              <div className="flex items-center gap-2">
                <label className="text-sm text-muted-foreground" htmlFor="group-exp">
                  {t("expirationLabel")}
                </label>
                <select
                  id="group-exp"
                  className="h-9 rounded-md border bg-background px-2 text-sm disabled:opacity-50"
                  value={expiration}
                  disabled={sending}
                  onChange={(e) => setExpiration(e.target.value as Expiration)}
                >
                  {EXPIRATIONS.map((exp) => (
                    <option key={exp} value={exp}>
                      {t(`exp_${exp}`)}
                    </option>
                  ))}
                </select>
                <Button className="ml-auto" onClick={handleSend} disabled={sending}>
                  {sending ? <Loader2 className="animate-spin" /> : <UploadIcon />}
                  {sending && percent !== null ? `${percent}%` : t("sendFiles")}
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {actionError && <p className="text-sm text-destructive">{actionError}</p>}

      {loading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : loadError ? (
        <ErrorState
          title={t("filesLoadErrorTitle")}
          description={t("filesLoadError")}
          details={loadError}
          retryLabel={t("retry")}
          onRetry={reload}
        />
      ) : empty ? (
        <p className="text-sm text-muted-foreground">{t("emptyFolder")}</p>
      ) : (
        <ul className="space-y-2">
          {data.folders.map((folder) => (
            <li key={folder.id} className="flex items-center gap-3 rounded-md border px-3 py-2">
              <button
                type="button"
                className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-left"
                onClick={() => setPath((p) => [...p, { id: folder.id, name: folder.name }])}
              >
                <Folder className="h-4 w-4 text-sky-500" />
                <span className="truncate text-sm font-medium">{folder.name}</span>
              </button>
              {canManage && (
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t("deleteFolder")}
                  disabled={busyId === folder.id}
                  onClick={() => handleDeleteFolder(folder.id)}
                >
                  {busyId === folder.id ? (
                    <Loader2 className="animate-spin" />
                  ) : (
                    <Trash2 className="text-destructive" />
                  )}
                </Button>
              )}
            </li>
          ))}
          {data.uploads.map((upload) => (
            <li key={upload.id} className="flex items-center gap-3 rounded-md border px-3 py-2">
              <FileIcon className="h-4 w-4 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{names[upload.id] ?? upload.code}</p>
                <p className="text-xs text-muted-foreground">
                  {t("sentBy", { user: upload.uploader })}
                  {upload.createdAt ? ` · ${new Date(upload.createdAt).toLocaleDateString(locale)}` : ""}
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                disabled={busyId === upload.id}
                onClick={() => handleDownload(upload.id, upload.code)}
              >
                {busyId === upload.id ? <Loader2 className="animate-spin" /> : <Download />}
                {t("download")}
              </Button>
              {canManage && (
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t("revokeFile")}
                  disabled={busyId === upload.id}
                  onClick={() => handleDeleteUpload(upload.id)}
                >
                  {busyId === upload.id ? (
                    <Loader2 className="animate-spin" />
                  ) : (
                    <Trash2 className="text-destructive" />
                  )}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
