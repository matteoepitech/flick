"use client"

import { ArrowDownLeft, ArrowUpRight, LayoutDashboard, LogIn } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useState } from "react"

import { BrandLogo } from "@/components/brand-logo"
import { ThemeToggle } from "@/components/theme-toggle"
import { UserAvatar } from "@/components/user-avatar"
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

  useEffect(() => {
    const stored = loadSession()
    setSession(stored)
    if (!stored) return

    const controller = new AbortController()
    verifySession(stored, controller.signal).then((status) => {
      if (status === "invalid") {
        setSession(null)
      } else if (status === "blocked") {
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
    <header className="sticky top-0 z-50 border-b bg-background/70 backdrop-blur-xl before:pointer-events-none before:absolute before:inset-x-0 before:bottom-full before:h-[100vh] before:bg-background before:content-['']">
      <div className="mx-auto flex h-[66px] max-w-6xl items-center justify-between gap-4 px-4 sm:px-7">
        <Link href="/" aria-label="Flick">
          <BrandLogo />
        </Link>

        <nav className="flex items-center gap-1.5 sm:gap-2">
          {canAccessDashboard(session?.user ?? null) && (
            <>
              <Link
                href="/dashboard"
                className="mr-1 hidden text-[14.5px] text-muted-foreground transition-colors hover:text-foreground md:inline"
              >
                {t("dashboard")}
              </Link>
              <Button asChild variant="ghost" size="icon" className="md:hidden">
                <Link href="/dashboard" aria-label={t("dashboard")}>
                  <LayoutDashboard className="size-4" />
                </Link>
              </Button>
            </>
          )}

          <Button asChild variant="outline" className="rounded-full px-3 sm:px-4">
            <Link href="/receive">
              <ArrowDownLeft className="size-4" />
              <span className="hidden sm:inline">{t("receive")}</span>
            </Link>
          </Button>
          <Button asChild className="rounded-full px-3 sm:px-4">
            <Link href="/send">
              <ArrowUpRight className="size-4" />
              <span className="hidden sm:inline">{t("send")}</span>
            </Link>
          </Button>

          {session ? (
            <Link
              href="/profile"
              aria-label={session.user.username || t("profile")}
              className="ml-1 rounded-full ring-1 ring-border transition-all hover:ring-ring/50"
            >
              <UserAvatar name={session.user.username || "?"} className="size-[34px]" />
            </Link>
          ) : (
            <Button asChild variant="ghost" size="icon" className="ml-1">
              <Link href="/login" aria-label={t("login")}>
                <LogIn className="size-4" />
              </Link>
            </Button>
          )}
          <ThemeToggle />
        </nav>
      </div>
    </header>
  )
}
