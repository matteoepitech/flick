"use client"

import { useEffect, useState } from "react"

import { type AuthSession } from "@/lib/api"
import { loadSession, verifySession } from "@/lib/auth"
import { canAccessDashboard } from "@/lib/permissions"
import { useRouter } from "@/i18n/navigation"

export function DashboardGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [allowed, setAllowed] = useState<boolean | null>(null)

  useEffect(() => {
    const session: AuthSession | null = loadSession()
    if (!session) {
      router.replace("/login")
      return
    }

    let cancelled = false
    const controller = new AbortController()

    verifySession(session, controller.signal).then((status) => {
      if (cancelled) return
      if (status === "blocked") {
        router.replace("/blocked")
        return
      }
      if (status === "invalid") {
        router.replace("/login")
        return
      }

      const fresh = loadSession() ?? session
      if (!canAccessDashboard(fresh.user)) {
        router.replace("/")
        return
      }
      setAllowed(true)
    })

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [router])

  if (!allowed) return null

  return <>{children}</>
}
