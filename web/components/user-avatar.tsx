import { cn } from "@/lib/utils"

const AVATAR_COLORS = [
  "bg-red-500",
  "bg-orange-500",
  "bg-amber-500",
  "bg-green-500",
  "bg-teal-500",
  "bg-sky-500",
  "bg-indigo-500",
  "bg-purple-500",
  "bg-pink-500",
]

function colorFor(seed: string): string {
  let hash = 0
  for (let i = 0; i < seed.length; i++) {
    hash = (hash * 31 + seed.charCodeAt(i)) >>> 0
  }
  return AVATAR_COLORS[hash % AVATAR_COLORS.length]
}

function initials(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return "?"
  return trimmed.slice(0, 2).toUpperCase()
}

interface UserAvatarProps {
  name: string
  className?: string
}

export function UserAvatar({ name, className }: UserAvatarProps) {
  return (
    <span
      className={cn(
        "inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full font-mono text-xs font-semibold text-white",
        colorFor(name),
        className
      )}
      aria-hidden
    >
      {initials(name)}
    </span>
  )
}
