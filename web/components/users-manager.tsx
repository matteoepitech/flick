"use client"

import { useCallback, useEffect, useState } from "react"
import { useLocale, useTranslations } from "next-intl"

import { CreateUserSheet } from "@/components/create-user-sheet"
import { EditUserSheet } from "@/components/edit-user-sheet"
import { ErrorState } from "@/components/error-state"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  ApiError,
  listUsers,
  updateUser,
  type AdminUser,
  type AuthSession,
  type UserRole,
} from "@/lib/api"
import { loadSession } from "@/lib/auth"

export function UsersManager() {
  const t = useTranslations("Users")
  const locale = useLocale()

  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [pending, setPending] = useState<Set<string>>(new Set())
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
    listUsers(session.token, ctrl.signal)
      .then((list) => setUsers(list))
      .catch((err: unknown) => {
        if (err instanceof Error && err.name === "AbortError") return
        setLoadError(err instanceof ApiError ? err.message : t("loadErrorDescription"))
      })
      .finally(() => setLoading(false))
    return () => ctrl.abort()
  }, [ready, session, isAdmin, reloadKey, t])

  const retry = useCallback(() => setReloadKey((k) => k + 1), [])

  const patch = useCallback(
    async (user: AdminUser, changes: { role?: UserRole; blocked?: boolean }) => {
      if (!session) return

      setActionError(null)
      setPending((prev) => new Set(prev).add(user.id))
      try {
        const updated = await updateUser(session.token, user.id, changes)
        setUsers((prev) => prev.map((u) => (u.id === updated.id ? updated : u)))
      } catch (err) {
        const message =
          err instanceof ApiError ? err.message : t("updateError", { username: user.username })
        setActionError(message)
      } finally {
        setPending((prev) => {
          const next = new Set(prev)
          next.delete(user.id)
          return next
        })
      }
    },
    [session, t]
  )

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
        <CreateUserSheet
          onCreated={(created) => setUsers((prev) => [created, ...prev])}
        />
      </div>

      {actionError && <p className="text-sm text-destructive">{actionError}</p>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("colUser")}</TableHead>
              <TableHead>{t("colEmail")}</TableHead>
              <TableHead>{t("colRole")}</TableHead>
              <TableHead>{t("colStatus")}</TableHead>
              <TableHead>{t("colCreated")}</TableHead>
              <TableHead className="text-right">{t("colActions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.length === 0 ? (
              <TableRow>
                <TableCell className="text-muted-foreground" colSpan={6}>
                  {t("empty")}
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => {
                const isSelf = user.id === session.user.id
                const busy = pending.has(user.id)
                return (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">
                      <span className="inline-flex items-center gap-2">
                        {user.username}
                        {isSelf && <Badge variant="outline">{t("you")}</Badge>}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{user.email}</TableCell>
                    <TableCell>
                      <select
                        className="h-8 rounded-md border bg-background px-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
                        value={user.role}
                        disabled={isSelf || busy}
                        onChange={(e) => patch(user, { role: e.target.value as UserRole })}
                        aria-label={t("colRole")}
                      >
                        <option value="user">{t("roleUser")}</option>
                        <option value="admin">{t("roleAdmin")}</option>
                      </select>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Switch
                          checked={!user.blocked}
                          disabled={isSelf || busy}
                          onCheckedChange={(active) => patch(user, { blocked: !active })}
                          aria-label={user.blocked ? t("unblockAction") : t("blockAction")}
                        />
                        <Badge
                          variant={user.blocked ? "destructive" : "secondary"}
                          className="w-20 justify-center"
                        >
                          {user.blocked ? t("blocked") : t("active")}
                        </Badge>
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {user.createdAt ? new Date(user.createdAt).toLocaleDateString(locale) : "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      <EditUserSheet
                        user={user}
                        token={session.token}
                        onUpdated={(updated) =>
                          setUsers((prev) => prev.map((u) => (u.id === updated.id ? updated : u)))
                        }
                      />
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
