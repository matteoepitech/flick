"use client"

import { useEffect, useState } from "react"
import { useTranslations } from "next-intl"

import { ErrorState } from "@/components/error-state"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useRouter } from "@/i18n/navigation"
import { type AuthSession, type GroupMembership } from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { GROUP_ROLE_BADGE } from "@/lib/group-roles"
import { cn } from "@/lib/utils"

function canManage(membership: GroupMembership): boolean {
  return membership.role === "maintainer" || membership.role === "owner"
}

export function MyGroupsList() {
  const t = useTranslations("MyGroups")
  const router = useRouter()
  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    setSession(loadSession())
    setReady(true)
  }, [])

  if (!ready) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  }

  const groups = session?.user.groups ?? []

  if (groups.length === 0) {
    return <ErrorState title={t("emptyTitle")} description={t("emptyDescription")} />
  }

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("colGroup")}</TableHead>
            <TableHead>{t("colRole")}</TableHead>
            <TableHead className="text-right">{t("colActions")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {groups.map((group) => {
            const manageable = canManage(group)
            return (
              <TableRow
                key={group.id}
                className="cursor-pointer"
                onClick={() => router.push(`/dashboard/group/${group.id}`)}
              >
                <TableCell className="font-medium">{group.name}</TableCell>
                <TableCell>
                  <Badge className={cn("w-32 justify-center rounded-full", GROUP_ROLE_BADGE[group.role])}>
                    {t(`role_${group.role}`)}
                  </Badge>
                </TableCell>
                <TableCell className="text-right text-sm font-medium text-primary">
                  {manageable ? t("manage") : t("open")}
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
