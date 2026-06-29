import Image from "next/image"

import { cn } from "@/lib/utils"

type BrandLogoProps = {
  markOnly?: boolean
  className?: string
}

export function BrandLogo({ markOnly = false, className }: BrandLogoProps) {
  return (
    <span className={cn("flex items-center", className)}>
      <Image src="/assets/flick_logo.png" alt="Flick" width={32} height={32} priority />
      {!markOnly && <span className="translate-y-[9px] font-heading text-lg font-bold tracking-tight">lick</span>}
    </span>
  )
}
