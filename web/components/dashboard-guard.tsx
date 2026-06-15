"use client"

import { useEffect, useState } from "react"

import { type AuthSession } from "@/lib/api"
import { loadSession, verifySession } from "@/lib/auth"
import { canAccessDashboard } from "@/lib/permissions"
import { useRouter } from "@/i18n/navigation"

// DashboardGuard: Client-side gate for the dashboard. Reads the stored session,
// confirms it still maps to a real account, and only renders the dashboard for
// users whose role grants access. Anonymous visitors and ghost (deleted-account)
// sessions go to login; signed-in users without permission go back home.
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

    verifySession(session, controller.signal).then((valid) => {
      if (cancelled) return
      if (!valid) {
        router.replace("/login")
        return
      }
      if (!canAccessDashboard(session.user)) {
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

  // Render nothing while we resolve the session to avoid flashing the dashboard
  // to users who will be redirected away.
  if (!allowed) return null

  return <>{children}</>
}
