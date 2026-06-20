import type { AuthUser } from "@/lib/api"

// Dashboard sections gated by role. Admins manage the whole instance;
// maintainers only get a single view scoped to their group.
export type DashboardSection = "overview" | "users" | "groups" | "settings" | "group"

// visibleSections: The dashboard sections a user is allowed to see.
//
// - Admins (global role) get the instance-wide administration sections.
// - Anyone who belongs to at least one group also gets the "My groups" page,
//   where they see their groups and (as a maintainer/owner) manage members.
// - A user with neither admin rights nor any group gets nothing and is kept out
//   of the dashboard.
export function visibleSections(user: AuthUser | null): DashboardSection[] {
  if (!user) return []

  const sections: DashboardSection[] = []
  if (user.role === "admin") sections.push("overview", "users", "groups", "settings")
  if (user.groups.length > 0) sections.push("group")
  return sections
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
  groups: "/dashboard/groups",
  group: "/dashboard/group",
  settings: "/dashboard/settings",
}

// landingPath: Where to send a user inside the dashboard, their first allowed
// section, or home if they have no access at all.
export function landingPath(user: AuthUser | null): string {
  const first = visibleSections(user)[0]
  return first ? SECTION_PATHS[first] : "/"
}
