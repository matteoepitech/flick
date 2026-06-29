import type { GroupRole } from "@/lib/api"

// The group roles in ascending order of privilege, for pickers and iteration.
export const GROUP_ROLES: GroupRole[] = ["member", "maintainer", "owner"]

// Distinctive colour per group role, shared by every badge so a role reads the
// same everywhere in the dashboard.
export const GROUP_ROLE_BADGE: Record<GroupRole, string> = {
  owner: "bg-amber-500/15 text-amber-600 border-amber-500/30 dark:text-amber-400",
  maintainer: "bg-sky-500/15 text-sky-600 border-sky-500/30 dark:text-sky-400",
  member: "bg-muted text-foreground/80 border-border",
}

// The matching dot colour used in role pickers.
export const GROUP_ROLE_DOT: Record<GroupRole, string> = {
  owner: "bg-amber-500",
  maintainer: "bg-sky-500",
  member: "bg-muted-foreground",
}
