"use client"

import { ArrowUpRight, LayoutDashboard, Settings, Users, UsersRound } from "lucide-react"
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
  label: string
  icon: React.ComponentType<{ className?: string }>
  section: DashboardSection
  separatedAbove?: boolean
}

// Every dashboard entry maps to a section so it can be filtered by the user's
// role. Admins see the administration items; maintainers see only their group.
const navItems: NavItem[] = [
  { href: "/dashboard", label: "Overview", icon: LayoutDashboard, section: "overview" },
  { href: "/dashboard/users", label: "Users", icon: Users, section: "users" },
  { href: "/dashboard/group", label: "My group", icon: UsersRound, section: "group" },
  { href: "/dashboard/settings", label: "Settings", icon: Settings, section: "settings", separatedAbove: true },
]

export function DashboardSidebar() {
  const pathname = usePathname()
  const [user, setUser] = useState<AuthUser | null>(null)

  useEffect(() => {
    setUser(loadSession()?.user ?? null)
  }, [])

  const allowed = visibleSections(user)
  const items = navItems.filter((item) => allowed.includes(item.section))

  const isActive = (href: string) => {
    if (href === "/dashboard") return pathname === "/dashboard" || /\/[^/]+\/dashboard$/.test(pathname)
    return pathname.endsWith(href)
  }

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <Link href="/" className="flex items-center gap-2 px-2 py-1">
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <ArrowUpRight className="h-5 w-5" />
          </span>
          <span className="text-lg font-semibold group-data-[collapsible=icon]:hidden">flick</span>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{user?.role === "admin" ? "Administration" : "Group"}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu className="gap-2">
              {items.map((item) => (
                <SidebarMenuItem
                  key={item.href}
                  className={item.separatedAbove ? "mt-2 border-t border-sidebar-border pt-2" : ""}
                >
                  <SidebarMenuButton asChild isActive={isActive(item.href)} tooltip={item.label}>
                    <Link href={item.href}>
                      <item.icon className="h-4 w-4" />
                      <span>{item.label}</span>
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
