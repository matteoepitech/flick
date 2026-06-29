"use client"

import { useEffect, useState } from "react"

import { loadSession } from "@/lib/auth"
import { canSee, landingPath, type DashboardSection } from "@/lib/permissions"
import { useRouter } from "@/i18n/navigation"

export function SectionGuard({ section, children }: { section: DashboardSection; children: React.ReactNode }) {
  const router = useRouter()
  const [allowed, setAllowed] = useState<boolean | null>(null)

  useEffect(() => {
    const user = loadSession()?.user ?? null
    if (canSee(user, section)) {
      setAllowed(true)
    } else {
      router.replace(landingPath(user))
    }
  }, [router, section])

  if (!allowed) return null

  return <>{children}</>
}
