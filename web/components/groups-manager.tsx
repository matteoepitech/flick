"use client"

import { useCallback, useEffect, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Users } from "lucide-react"

import { CreateGroupSheet } from "@/components/create-group-sheet"
import { DeleteGroupSheet } from "@/components/delete-group-sheet"
import { ErrorState } from "@/components/error-state"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useRouter } from "@/i18n/navigation"
import { ApiError, listGroups, type AdminGroup, type AuthSession } from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { cn } from "@/lib/utils"

const HEAD_CLASS = "font-heading font-semibold text-[10.5px] tracking-[0.1em] uppercase text-muted-foreground"

export function GroupsManager() {
  const t = useTranslations("Groups")
  const locale = useLocale()
  const router = useRouter()

  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)
  const [groups, setGroups] = useState<AdminGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    setSession(loadSession())
    setReady(true)
  }, [])

  const isAdmin = session?.user.role === "admin"

  useEffect(() => {
    if (!ready || !session || !isAdmin) return

    const ctrl = new AbortController()
    setLoading(true)
    setLoadError(null)
    listGroups(session.token, ctrl.signal)
      .then((list) => setGroups(list))
      .catch((err: unknown) => {
        if (err instanceof Error && err.name === "AbortError") return
        setLoadError(err instanceof ApiError ? err.message : t("loadErrorDescription"))
      })
      .finally(() => setLoading(false))
    return () => ctrl.abort()
  }, [ready, session, isAdmin, reloadKey, t])

  const retry = useCallback(() => setReloadKey((k) => k + 1), [])

  const onCreated = useCallback((group: AdminGroup) => {
    setGroups((prev) => [...prev, group])
  }, [])

  const onDeleted = useCallback((id: string) => {
    setGroups((prev) => prev.filter((g) => g.id !== id))
  }, [])

  if (!ready || (isAdmin && loading)) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  }

  if (!session || !isAdmin) {
    return <ErrorState title={t("forbiddenTitle")} description={t("forbiddenDescription")} />
  }

  if (loadError) {
    return (
      <ErrorState
        title={t("loadErrorTitle")}
        description={t("loadErrorDescription")}
        details={loadError}
        retryLabel={t("retry")}
        onRetry={retry}
      />
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <CreateGroupSheet token={session.token} onCreated={onCreated} />
      </div>

      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className={HEAD_CLASS}>{t("colName")}</TableHead>
              <TableHead className={HEAD_CLASS}>{t("colCreated")}</TableHead>
              <TableHead className={cn(HEAD_CLASS, "text-right")}>{t("colActions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {groups.length === 0 ? (
              <TableRow>
                <TableCell className="text-muted-foreground" colSpan={3}>
                  {t("empty")}
                </TableCell>
              </TableRow>
            ) : (
              groups.map((group) => (
                <TableRow
                  key={group.id}
                  className="cursor-pointer border-border hover:bg-muted/60"
                  onClick={() => router.push(`/dashboard/groups/${group.id}`)}
                >
                  <TableCell>
                    <span className="inline-flex items-center gap-2.5">
                      <span className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                        <Users className="size-4" />
                      </span>
                      <span className="font-heading font-semibold">{group.name}</span>
                    </span>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {group.createdAt ? new Date(group.createdAt).toLocaleDateString(locale) : "—"}
                  </TableCell>
                  <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                    <DeleteGroupSheet group={group} token={session.token} onDeleted={onDeleted} />
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
