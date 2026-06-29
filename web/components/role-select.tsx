"use client"

import { ChevronDown, Loader2 } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { cn } from "@/lib/utils"

export type RoleOption<T extends string> = {
  value: T
  label: string

  badgeClass: string

  dotClass: string
}

type RoleSelectProps<T extends string> = {
  value: T
  options: RoleOption<T>[]
  onSelect: (value: T) => void
  disabled?: boolean
  busy?: boolean
  ariaLabel?: string

  widthClassName?: string
}

export function RoleSelect<T extends string>({
  value,
  options,
  onSelect,
  disabled = false,
  busy = false,
  ariaLabel,
  widthClassName = "w-32",
}: RoleSelectProps<T>) {
  const current = options.find((option) => option.value === value)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild disabled={disabled || busy}>
        <button
          type="button"
          aria-label={ariaLabel}
          className="rounded-full disabled:cursor-not-allowed disabled:opacity-60"
        >
          <Badge
            className={cn(
              "cursor-pointer justify-between gap-1.5 rounded-full px-2.5",
              widthClassName,
              current?.badgeClass
            )}
          >
            <span className="inline-flex min-w-0 items-center gap-1.5">
              {busy && <Loader2 className="size-3 shrink-0 animate-spin" />}
              <span className="truncate">{current?.label ?? value}</span>
            </span>
            <ChevronDown className="size-3.5 shrink-0 opacity-80" />
          </Badge>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {options.map((option) => (
          <DropdownMenuItem key={option.value} onClick={() => onSelect(option.value)}>
            <span className={cn("mr-2 size-2 rounded-full", option.dotClass)} />
            {option.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
