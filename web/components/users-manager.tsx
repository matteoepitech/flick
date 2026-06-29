"use client"

import { useCallback, useEffect, useState } from "react"
import { useLocale, useTranslations } from "next-intl"

import { CreateUserSheet } from "@/components/create-user-sheet"
import { EditUserSheet } from "@/components/edit-user-sheet"
import { ErrorState } from "@/components/error-state"
import { RoleSelect } from "@/components/role-select"
import { UserAvatar } from "@/components/user-avatar"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { ApiError, listUsers, updateUser, type AdminUser, type AuthSession, type UserRole } from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { cn } from "@/lib/utils"

const HEAD_CLASS = "font-heading font-semibold text-[10.5px] tracking-[0.1em] uppercase text-muted-foreground"

const USER_ROLES: UserRole[] = ["user", "admin"]
const USER_ROLE_BADGE: Record<UserRole, string> = {
  admin: "bg-primary/12 text-primary border-primary/20",
  user: "bg-muted text-foreground/80 border-border",
}
const USER_ROLE_DOT: Record<UserRole, string> = {
  admin: "bg-primary",
  user: "bg-muted-foreground",
}

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
        const message = err instanceof ApiError ? err.message : t("updateError", { username: user.username })
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
        <CreateUserSheet onCreated={(created) => setUsers((prev) => [created, ...prev])} />
      </div>

      {actionError && <p className="text-sm text-destructive">{actionError}</p>}

      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className={HEAD_CLASS}>{t("colUser")}</TableHead>
              <TableHead className={HEAD_CLASS}>{t("colEmail")}</TableHead>
              <TableHead className={HEAD_CLASS}>{t("colRole")}</TableHead>
              <TableHead className={HEAD_CLASS}>{t("colStatus")}</TableHead>
              <TableHead className={HEAD_CLASS}>{t("colCreated")}</TableHead>
              <TableHead className={cn(HEAD_CLASS, "text-right")}>{t("colActions")}</TableHead>
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
                  <TableRow key={user.id} className="border-border hover:bg-muted/60">
                    <TableCell className="font-medium">
                      <span className="inline-flex items-center gap-2.5">
                        <UserAvatar name={user.username} className="h-7 w-7" />
                        {user.username}
                        {isSelf && (
                          <span className="rounded-full bg-primary/12 px-2 py-0.5 font-heading text-[10px] font-semibold tracking-wide text-primary uppercase">
                            {t("you")}
                          </span>
                        )}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{user.email}</TableCell>
                    <TableCell>
                      <RoleSelect
                        value={user.role}
                        options={USER_ROLES.map((role) => ({
                          value: role,
                          label: role === "admin" ? t("roleAdmin") : t("roleUser"),
                          badgeClass: USER_ROLE_BADGE[role],
                          dotClass: USER_ROLE_DOT[role],
                        }))}
                        onSelect={(role) => patch(user, { role })}
                        disabled={isSelf}
                        busy={busy}
                        ariaLabel={t("colRole")}
                        widthClassName="w-28"
                      />
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Switch
                          checked={!user.blocked}
                          disabled={isSelf || busy}
                          onCheckedChange={(active) => patch(user, { blocked: !active })}
                          aria-label={user.blocked ? t("unblockAction") : t("blockAction")}
                        />
                        <span
                          className={cn(
                            "inline-flex w-20 justify-center rounded-full px-2.5 py-0.5 font-mono text-[11px] font-medium",
                            user.blocked ? "bg-destructive/12 text-destructive" : "bg-success/12 text-success"
                          )}
                        >
                          {user.blocked ? t("blocked") : t("active")}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
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
