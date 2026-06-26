"use client"

import { ArrowUpRight, Boxes, LayoutDashboard, Settings, Users, UsersRound } from "lucide-react"
import { useTranslations } from "next-intl"
import Image from "next/image"
import { usePathname } from "next/navigation"
import { useEffect, useState } from "react"

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

// Every dashboard entry maps to a section so it can be filtered by the user's
// role. Admins see the administration items; maintainers see only their group.
const navItems: NavItem[] = [
  { href: "/dashboard", labelKey: "overview", icon: LayoutDashboard, section: "overview" },
  { href: "/dashboard/users", labelKey: "users", icon: Users, section: "users" },
  { href: "/dashboard/groups", labelKey: "groups", icon: Boxes, section: "groups" },
  { href: "/dashboard/group", labelKey: "myGroups", icon: UsersRound, section: "group" },
  { href: "/dashboard/settings", labelKey: "settings", icon: Settings, section: "settings", separatedAbove: true },
]

export function DashboardSidebar() {
  const t = useTranslations("Sidebar")
  const pathname = usePathname()
  const [user, setUser] = useState<AuthUser | null>(null)

  useEffect(() => {
    setUser(loadSession()?.user ?? null)
  }, [])

  const allowed = visibleSections(user)
  const items = navItems.filter((item) => allowed.includes(item.section))

  const isActive = (href: string) => {
    if (href === "/dashboard") return pathname === "/dashboard" || /\/[^/]+\/dashboard$/.test(pathname)
    // Match the section page itself or any of its sub-routes (e.g. a group
    // detail page), while keeping "/dashboard/group" distinct from
    // "/dashboard/groups" thanks to the trailing slash.
    return pathname.endsWith(href) || pathname.includes(href + "/")
  }

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <Link href="/" className="flex items-center gap-0 px-2 py-1">
          <Image src="/assets/flick_logo.png" alt="Flick" width={32} height={32} />
          <span className="translate-y-2 text-lg font-semibold group-data-[collapsible=icon]:hidden">lick</span>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{user?.role === "admin" ? t("groupAdministration") : t("groupGroup")}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu className="gap-2">
              {items.map((item) => (
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
      </SidebarContent>
      <SidebarFooter></SidebarFooter>
    </Sidebar>
  )
}
