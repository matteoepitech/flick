"use client"

import { usePathname } from "next/navigation"
import type { ReactNode } from "react"

export function PageTransition({ children }: { children: ReactNode }) {
  const pathname = usePathname()

  return (
    <div key={pathname} className="animate-in duration-500 ease-out fade-in-0 slide-in-from-bottom-2">
      {children}
    </div>
  )
}
