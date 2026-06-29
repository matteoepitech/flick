"use client"

import { Boxes, LayoutDashboard, Settings, Users, UsersRound } from "lucide-react"
import { useTranslations } from "next-intl"
import { usePathname } from "next/navigation"
import { useEffect, useState } from "react"

import { BrandLogo } from "@/components/brand-logo"
import { UserAvatar } from "@/components/user-avatar"
import { Link } from "@/i18n/navigation"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { type AuthUser } from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { type DashboardSection, visibleSections } from "@/lib/permissions"

type NavItem = {
  href: string
  labelKey: "overview" | "users" | "groups" | "myGroups" | "settings"
  icon: React.ComponentType<{ className?: string }>
  section: DashboardSection
  separatedAbove?: boolean
}

type NavGroup = {
  labelKey: "groupAdministration" | "workspace"
  items: NavItem[]
}

const NAV_GROUPS: NavGroup[] = [
  {
    labelKey: "workspace",
    items: [{ href: "/dashboard/group", labelKey: "myGroups", icon: UsersRound, section: "group" }],
  },
  {
    labelKey: "groupAdministration",
    items: [
      { href: "/dashboard", labelKey: "overview", icon: LayoutDashboard, section: "overview" },
      { href: "/dashboard/users", labelKey: "users", icon: Users, section: "users" },
      { href: "/dashboard/groups", labelKey: "groups", icon: Boxes, section: "groups" },
      { href: "/dashboard/settings", labelKey: "settings", icon: Settings, section: "settings", separatedAbove: true },
    ],
  },
]

export function DashboardSidebar() {
  const t = useTranslations("Sidebar")
  const pathname = usePathname()
  const [user, setUser] = useState<AuthUser | null>(null)

  useEffect(() => {
    setUser(loadSession()?.user ?? null)
  }, [])

  const allowed = visibleSections(user)

  const isActive = (href: string) => {
    if (href === "/dashboard") return pathname === "/dashboard" || /\/[^/]+\/dashboard$/.test(pathname)

    return pathname.endsWith(href) || pathname.includes(href + "/")
  }

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <Link href="/" className="flex items-center px-1.5 py-1" aria-label="Flick">
          <BrandLogo className="group-data-[collapsible=icon]:[&>span:last-child]:hidden" />
        </Link>
      </SidebarHeader>
      <SidebarContent>
        {NAV_GROUPS.map((group) => {
          const groupItems = group.items.filter((item) => allowed.includes(item.section))
          if (groupItems.length === 0) return null
          return (
            <SidebarGroup key={group.labelKey}>
              <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu className="gap-2">
                  {groupItems.map((item) => (
                    <SidebarMenuItem
                      key={item.href}
                      className={item.separatedAbove ? "mt-2 border-t border-sidebar-border pt-2" : ""}
                    >
                      <SidebarMenuButton asChild isActive={isActive(item.href)} tooltip={t(item.labelKey)}>
                        <Link href={item.href}>
                          <item.icon className="h-4 w-4" />
                          <span>{t(item.labelKey)}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          )
        })}
      </SidebarContent>
      <SidebarFooter>
        {user && (
          <Link
            href="/profile"
            className="flex items-center gap-2.5 rounded-lg p-1.5 transition-colors group-data-[collapsible=icon]:justify-center hover:bg-sidebar-accent"
          >
            <UserAvatar name={user.username || "?"} className="size-8" />
            <div className="min-w-0 group-data-[collapsible=icon]:hidden">
              <div className="truncate text-sm font-semibold">{user.username}</div>
              <div className="text-xs text-muted-foreground capitalize">
                {user.role === "admin" ? t("groupAdministration") : t("groupGroup")}
              </div>
            </div>
          </Link>
        )}
      </SidebarFooter>
    </Sidebar>
  )
}
