"use client"

import { ArrowDownLeft, ArrowUpRight, LayoutDashboard, LogIn, UserRound } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useState } from "react"

import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { type AuthSession } from "@/lib/api"
import { loadSession, verifySession } from "@/lib/auth"
import { canAccessDashboard } from "@/lib/permissions"
import { Link, usePathname, useRouter } from "@/i18n/navigation"

export default function SiteHeader() {
  const t = useTranslations("Header")
  const pathname = usePathname()
  const router = useRouter()

  const [session, setSession] = useState<AuthSession | null>(null)

  // Re-read the session on every navigation so the header reflects login/logout
  // without a full page reload, drop it if the server no longer knows the
  // account (e.g. a deleted user), and send blocked users to the blocked page.
  useEffect(() => {
    const stored = loadSession()
    setSession(stored)
    if (!stored) return

    const controller = new AbortController()
    verifySession(stored, controller.signal).then((status) => {
      if (status === "invalid") {
        setSession(null)
      } else if (status === "blocked") {
        // A blocked user keeps read-only access to their own profile (and the
        // blocked page); everywhere else sends them back to the blocked page.
        const allowed = pathname.startsWith("/blocked") || pathname.startsWith("/profile")
        if (!allowed) router.replace("/blocked")
      }
    })

    return () => controller.abort()
  }, [pathname, router])

  if (pathname.startsWith("/dashboard")) {
    return null
  }

  return (
    <header className="border-b">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2">
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <ArrowUpRight className="h-5 w-5" />
          </span>
          <span className="text-lg font-semibold">flick</span>
        </Link>

        <nav className="flex items-center gap-1 sm:gap-2 md:gap-4">
          {canAccessDashboard(session?.user ?? null) && (
            <Button asChild variant="ghost" size="icon" className="sm:w-auto sm:gap-1.5 sm:px-4">
              <Link href="/dashboard">
                <LayoutDashboard className="h-4 w-4" />
                <span className="hidden sm:inline">{t("dashboard")}</span>
              </Link>
            </Button>
          )}
          <Button asChild size="icon" className="sm:w-auto sm:gap-1.5 sm:px-4">
            <Link href="/send">
              <ArrowUpRight className="h-4 w-4" />
              <span className="hidden sm:inline">{t("send")}</span>
            </Link>
          </Button>
          <Button asChild variant="outline" size="icon" className="sm:w-auto sm:gap-1.5 sm:px-4">
            <Link href="/receive">
              <ArrowDownLeft className="h-4 w-4" />
              <span className="hidden sm:inline">{t("receive")}</span>
            </Link>
          </Button>
          {session ? (
            <Button asChild variant="ghost" size="icon" className="sm:w-auto sm:gap-1.5 sm:px-4">
              <Link href="/profile">
                <UserRound className="h-4 w-4" />
                <span className="hidden max-w-[12ch] truncate sm:inline">{session.user.username || t("profile")}</span>
              </Link>
            </Button>
          ) : (
            <Button asChild variant="ghost" size="icon" className="sm:w-auto sm:gap-1.5 sm:px-4">
              <Link href="/login">
                <LogIn className="h-4 w-4" />
                <span className="hidden sm:inline">{t("login")}</span>
              </Link>
            </Button>
          )}
          <ThemeToggle />
        </nav>
      </div>
    </header>
  )
}
