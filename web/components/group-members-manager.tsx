"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Check, Loader2, Pencil, Search, X } from "lucide-react"

import { ErrorState } from "@/components/error-state"
import { UserAvatar } from "@/components/user-avatar"
import { Badge } from "@/components/ui/badge"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  ApiError,
  addGroupMember,
  listGroupMembers,
  listGroups,
  listUsers,
  removeGroupMember,
  renameGroup,
  setMemberRole,
  type AdminUser,
  type AuthSession,
  type GroupMember,
  type GroupRole,
} from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { GROUP_ROLES, GROUP_ROLE_BADGE, GROUP_ROLE_DOT } from "@/lib/group-roles"
import { cn } from "@/lib/utils"

export function GroupMembersManager({ groupId }: { groupId: string }) {
  const t = useTranslations("Groups")
  const locale = useLocale()

  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)
  const [groupName, setGroupName] = useState<string | null>(null)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [users, setUsers] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [pending, setPending] = useState<Set<string>>(new Set())
  const [query, setQuery] = useState("")
  const [addingId, setAddingId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState(false)
  const [nameInput, setNameInput] = useState("")
  const [savingName, setSavingName] = useState(false)
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
    Promise.all([
      listGroupMembers(session.token, groupId, ctrl.signal),
      listGroups(session.token, ctrl.signal),
      listUsers(session.token, ctrl.signal),
    ])
      .then(([memberList, groups, userList]) => {
        setMembers(memberList)
        setUsers(userList)
        setGroupName(groups.find((g) => g.id === groupId)?.name ?? null)
      })
      .catch((err: unknown) => {
        if (err instanceof Error && err.name === "AbortError") return
        setLoadError(err instanceof ApiError ? err.message : t("loadErrorDescription"))
      })
      .finally(() => setLoading(false))
    return () => ctrl.abort()
  }, [ready, session, isAdmin, groupId, reloadKey, t])

  const retry = useCallback(() => setReloadKey((k) => k + 1), [])

  // Search results: users not already members, matching the query. Capped so the
  // dropdown never grows unbounded.
  const results = useMemo(() => {
    const memberIds = new Set(members.map((m) => m.id))
    const q = query.trim().toLowerCase()
    if (!q) return []
    return users
      .filter((u) => !memberIds.has(u.id))
      .filter((u) => u.username.toLowerCase().includes(q) || u.email.toLowerCase().includes(q))
      .slice(0, 8)
  }, [members, users, query])

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
    async (user: AdminUser) => {
      if (!session) return
      setAddingId(user.id)
      setActionError(null)
      try {
        await addGroupMember(session.token, groupId, user.id)
        setMembers((prev) => [...prev, { ...user, groupRole: "member" as GroupRole }])
        setQuery("")
      } catch (err) {
        setActionError(err instanceof ApiError ? err.message : t("addError"))
      } finally {
        setAddingId(null)
      }
    },
    [session, groupId, t]
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

  const startRename = useCallback(() => {
    setNameInput(groupName ?? "")
    setEditingName(true)
  }, [groupName])

  const handleRename = useCallback(async () => {
    if (!session) return
    const trimmed = nameInput.trim()
    if (!trimmed || trimmed === groupName) {
      setEditingName(false)
      return
    }
    setSavingName(true)
    setActionError(null)
    try {
      const updated = await renameGroup(session.token, groupId, trimmed)
      setGroupName(updated.name)
      setEditingName(false)
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : t("renameError"))
    } finally {
      setSavingName(false)
    }
  }, [session, groupId, nameInput, groupName, t])

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
    <div className="space-y-6">
      <div>
        {editingName ? (
          <div className="flex items-center gap-2">
            <Input
              className="h-9 max-w-sm text-lg font-semibold"
              value={nameInput}
              disabled={savingName}
              autoFocus
              onChange={(e) => setNameInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRename()
                if (e.key === "Escape") setEditingName(false)
              }}
            />
            <Button
              size="icon"
              variant="ghost"
              disabled={savingName}
              onClick={handleRename}
              aria-label={t("renameSave")}
            >
              {savingName ? <Loader2 className="animate-spin" /> : <Check />}
            </Button>
            <Button
              size="icon"
              variant="ghost"
              disabled={savingName}
              onClick={() => setEditingName(false)}
              aria-label={t("renameCancel")}
            >
              <X />
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <h2 className="text-2xl font-semibold tracking-tight">{groupName ?? t("membersTitle")}</h2>
            <Button size="icon" variant="ghost" onClick={startRename} aria-label={t("renameAction")}>
              <Pencil />
            </Button>
          </div>
        )}
        <p className="text-muted-foreground">{t("membersSubtitle")}</p>
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
          {query.trim() && (
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
                return (
                  <TableRow key={member.id}>
                    <TableCell className="font-medium">
                      <span className="inline-flex items-center gap-2">
                        <UserAvatar name={member.username} className="h-7 w-7" />
                        {member.username}
                        {member.role === "admin" && <Badge variant="outline">{t("roleAdmin")}</Badge>}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{member.email}</TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild disabled={busy}>
                          <button type="button" className="disabled:opacity-50" aria-label={t("colGroupRole")}>
                            <Badge
                              className={cn("w-32 cursor-pointer justify-center", GROUP_ROLE_BADGE[member.groupRole])}
                            >
                              {busy && <Loader2 className="h-3 w-3 animate-spin" />}
                              {t(`groupRole_${member.groupRole}`)}
                            </Badge>
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="start">
                          {GROUP_ROLES.map((role) => (
                            <DropdownMenuItem key={role} onClick={() => handleRole(member, role)}>
                              <span className={cn("mr-2 h-2 w-2 rounded-full", GROUP_ROLE_DOT[role])} />
                              {t(`groupRole_${role}`)}
                            </DropdownMenuItem>
                          ))}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {member.createdAt ? new Date(member.createdAt).toLocaleDateString(locale) : "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" disabled={busy} onClick={() => handleRemove(member)}>
                        {t("removeAction")}
                      </Button>
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
