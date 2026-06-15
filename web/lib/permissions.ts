import type { AuthUser } from "@/lib/api"

// Dashboard sections gated by role. Admins manage the whole instance;
// maintainers only get a single view scoped to their group.
export type DashboardSection = "overview" | "users" | "settings" | "group"

// visibleSections: The dashboard sections a user is allowed to see.
//
// - Admins (global role) see everything.
// - Maintainers/owners (group role) only see their group page.
// - Everyone else gets nothing and is kept out of the dashboard.
//
// The API does not expose group memberships yet, so `groupRole` is currently
// always undefined: in practice only admins reach the dashboard for now.
export function visibleSections(user: AuthUser | null): DashboardSection[] {
  if (!user) return []
  if (user.role === "admin") return ["overview", "users", "settings"]
  if (user.groupRole === "maintainer" || user.groupRole === "owner") return ["group"]
  return []
}

// canAccessDashboard: Whether a user may open the dashboard at all.
export function canAccessDashboard(user: AuthUser | null): boolean {
  return visibleSections(user).length > 0
}

// canSee: Whether a user may access a specific dashboard section.
export function canSee(user: AuthUser | null, section: DashboardSection): boolean {
  return visibleSections(user).includes(section)
}

// Route backing each dashboard section, the single source of truth shared by
// the sidebar and the redirect helpers.
export const SECTION_PATHS: Record<DashboardSection, string> = {
  overview: "/dashboard",
  users: "/dashboard/users",
  group: "/dashboard/group",
  settings: "/dashboard/settings",
}

// landingPath: Where to send a user inside the dashboard — their first allowed
// section, or home if they have no access at all.
export function landingPath(user: AuthUser | null): string {
  const first = visibleSections(user)[0]
  return first ? SECTION_PATHS[first] : "/"
}
