"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Loader2, Search } from "lucide-react"

import { ErrorState } from "@/components/error-state"
import { UserAvatar } from "@/components/user-avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  ApiError,
  addGroupMember,
  listGroupMembers,
  removeGroupMember,
  searchUsers,
  setMemberRole,
  type AuthSession,
  type GroupMember,
  type GroupMembership,
  type GroupRole,
  type UserSearchResult,
} from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { GROUP_ROLES, GROUP_ROLE_BADGE, GROUP_ROLE_DOT } from "@/lib/group-roles"
import { cn } from "@/lib/utils"

function canManage(membership: GroupMembership | undefined): boolean {
  return membership?.role === "maintainer" || membership?.role === "owner"
}

export function MyGroupManager({ groupId }: { groupId: string }) {
  const t = useTranslations("MyGroups")
  const locale = useLocale()

  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [pending, setPending] = useState<Set<string>>(new Set())
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<UserSearchResult[]>([])
  const [addingId, setAddingId] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    setSession(loadSession())
    setReady(true)
  }, [])

  const membership = session?.user.groups.find((g) => g.id === groupId)
  const allowed = canManage(membership)
  // Only the owner (or a global admin) may change roles; maintainers cannot.
  const canEditRoles = membership?.role === "owner" || session?.user.role === "admin"

  useEffect(() => {
    if (!ready || !session || !allowed) return

    const ctrl = new AbortController()
    setLoading(true)
    setLoadError(null)
    listGroupMembers(session.token, groupId, ctrl.signal)
      .then((list) => setMembers(list))
      .catch((err: unknown) => {
        if (err instanceof Error && err.name === "AbortError") return
        setLoadError(err instanceof ApiError ? err.message : t("loadErrorDescription"))
      })
      .finally(() => setLoading(false))
    return () => ctrl.abort()
  }, [ready, session, allowed, groupId, reloadKey, t])

  const memberIds = useMemo(() => new Set(members.map((m) => m.id)), [members])

  // Search the directory as the user types (server-side), hiding existing members.
  useEffect(() => {
    if (!session) return
    const q = query.trim()
    if (q.length < 2) {
      setResults([])
      return
    }
    const ctrl = new AbortController()
    searchUsers(session.token, q, ctrl.signal)
      .then((found) => setResults(found.filter((u) => !memberIds.has(u.id))))
      .catch(() => {
        // Ignore search errors; the field stays usable.
      })
    return () => ctrl.abort()
  }, [query, session, memberIds])

  const withPending = useCallback(async (id: string, fn: () => Promise<void>) => {
    setActionError(null)
    setPending((prev) => new Set(prev).add(id))
    try {
      await fn()
    } finally {
      setPending((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }, [])

  const handleAdd = useCallback(
    async (user: UserSearchResult) => {
      if (!session) return
      setAddingId(user.id)
      setActionError(null)
      try {
        await addGroupMember(session.token, groupId, user.id)
        setQuery("")
        setResults([])
        setReloadKey((k) => k + 1)
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("addError"))
      } finally {
        setAddingId(null)
      }
    },
    [session, groupId, t]
  )

  const handleRemove = useCallback(
    async (member: GroupMember) => {
      if (!session) return
      try {
        await withPending(member.id, () => removeGroupMember(session.token, groupId, member.id))
        setMembers((prev) => prev.filter((m) => m.id !== member.id))
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("removeError", { username: member.username }))
      }
    },
    [session, groupId, withPending, t]
  )

  const handleRole = useCallback(
    async (member: GroupMember, role: GroupRole) => {
      if (!session || member.groupRole === role) return
      try {
        await withPending(member.id, () => setMemberRole(session.token, groupId, member.id, role))
        setMembers((prev) => prev.map((m) => (m.id === member.id ? { ...m, groupRole: role } : m)))
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("roleError", { username: member.username }))
      }
    },
    [session, groupId, withPending, t]
  )

  if (!ready || (allowed && loading)) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  }

  if (!session || !membership || !allowed) {
    return <ErrorState title={t("forbiddenTitle")} description={t("forbiddenDescription")} />
  }

  if (loadError) {
    return (
      <ErrorState
        title={t("loadErrorTitle")}
        description={t("loadErrorDescription")}
        details={loadError}
        retryLabel={t("retry")}
        onRetry={() => setReloadKey((k) => k + 1)}
      />
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <div className="flex items-center gap-2">
          <h2 className="text-2xl font-semibold tracking-tight">{membership.name}</h2>
          <Badge className={cn("justify-center", GROUP_ROLE_BADGE[membership.role])}>
            {t(`role_${membership.role}`)}
          </Badge>
        </div>
        <p className="text-muted-foreground">{t("manageSubtitle")}</p>
      </div>

      <div className="space-y-1">
        <label className="text-sm font-medium" htmlFor="add-member">
          {t("addMemberLabel")}
        </label>
        <div className="relative max-w-sm">
          <Search className="pointer-events-none absolute top-2.5 left-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            id="add-member"
            className="pl-8"
            value={query}
            placeholder={t("searchPlaceholder")}
            onChange={(e) => setQuery(e.target.value)}
            autoComplete="off"
          />
          {query.trim().length >= 2 && (
            <div className="absolute z-10 mt-1 w-full overflow-hidden rounded-md border bg-popover shadow-md">
              {results.length === 0 ? (
                <p className="px-3 py-2 text-sm text-muted-foreground">{t("noResults")}</p>
              ) : (
                results.map((u) => (
                  <button
                    key={u.id}
                    type="button"
                    disabled={addingId === u.id}
                    onClick={() => handleAdd(u)}
                    className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent disabled:opacity-50"
                  >
                    <UserAvatar name={u.username} className="h-7 w-7" />
                    <span className="flex flex-col">
                      <span className="font-medium">{u.username}</span>
                      <span className="text-xs text-muted-foreground">{u.email}</span>
                    </span>
                    {addingId === u.id && <Loader2 className="ml-auto h-4 w-4 animate-spin" />}
                  </button>
                ))
              )}
            </div>
          )}
        </div>
      </div>

      {actionError && <p className="text-sm text-destructive">{actionError}</p>}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("colUser")}</TableHead>
              <TableHead>{t("colEmail")}</TableHead>
              <TableHead>{t("colGroupRole")}</TableHead>
              <TableHead>{t("colCreated")}</TableHead>
              <TableHead className="text-right">{t("colActions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {members.length === 0 ? (
              <TableRow>
                <TableCell className="text-muted-foreground" colSpan={5}>
                  {t("noMembers")}
                </TableCell>
              </TableRow>
            ) : (
              members.map((member) => {
                const busy = pending.has(member.id)
                // A maintainer/owner cannot remove or re-role themselves; the API
                // enforces this, so we hide the controls on their own row too.
                const isSelf = member.id === session.user.id
                return (
                  <TableRow key={member.id}>
                    <TableCell className="font-medium">
                      <span className="inline-flex items-center gap-2">
                        <UserAvatar name={member.username} className="h-7 w-7" />
                        {member.username}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{member.email}</TableCell>
                    <TableCell>
                      {canEditRoles && !isSelf ? (
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild disabled={busy}>
                            <button type="button" className="disabled:opacity-50" aria-label={t("colGroupRole")}>
                              <Badge
                                className={cn("w-32 cursor-pointer justify-center", GROUP_ROLE_BADGE[member.groupRole])}
                              >
                                {busy && <Loader2 className="h-3 w-3 animate-spin" />}
                                {t(`role_${member.groupRole}`)}
                              </Badge>
                            </button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="start">
                            {GROUP_ROLES.map((role) => (
                              <DropdownMenuItem key={role} onClick={() => handleRole(member, role)}>
                                <span className={cn("mr-2 h-2 w-2 rounded-full", GROUP_ROLE_DOT[role])} />
                                {t(`role_${role}`)}
                              </DropdownMenuItem>
                            ))}
                          </DropdownMenuContent>
                        </DropdownMenu>
                      ) : (
                        <Badge className={cn("w-32 justify-center", GROUP_ROLE_BADGE[member.groupRole])}>
                          {t(`role_${member.groupRole}`)}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {member.createdAt ? new Date(member.createdAt).toLocaleDateString(locale) : "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      {isSelf ? (
                        <span className="text-sm text-muted-foreground">{t("you")}</span>
                      ) : (
                        <Button variant="ghost" size="sm" disabled={busy} onClick={() => handleRemove(member)}>
                          {busy && <Loader2 className="animate-spin" />}
                          {t("removeAction")}
                        </Button>
                      )}
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
