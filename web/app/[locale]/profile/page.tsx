"use client"

import { AtSign, Calendar, Check, ChevronLeft, Copy, Fingerprint, LogOut, UserRound } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { type AuthSession } from "@/lib/api"
import { clearSession, loadSession, verifySession } from "@/lib/auth"
import { Link, useRouter } from "@/i18n/navigation"

export default function ProfilePage() {
  const t = useTranslations("Profile")
  const locale = useLocale()
  const router = useRouter()

  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    const stored = loadSession()
    setSession(stored)
    setReady(true)
    if (!stored) return

    // Drop a ghost session whose account no longer exists on the server.
    const controller = new AbortController()
    verifySession(stored, controller.signal).then((valid) => {
      if (!valid) setSession(null)
    })

    return () => controller.abort()
  }, [])

  function handleLogout() {
    clearSession()
    router.push("/login")
  }

  async function copyId(id: string) {
    try {
      await navigator.clipboard.writeText(id)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard unavailable, ignore silently.
    }
  }

  return (
    <main className="mx-auto w-full max-w-3xl px-6 py-16">
      <Link
        href="/"
        className="mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("back")}
      </Link>

      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight">{t("title")}</h1>
        <p className="mt-2 text-muted-foreground">{t("subtitle")}</p>
      </div>

      {!ready ? null : !session ? (
        <Card>
          <CardContent className="flex flex-col items-center gap-4 py-12 text-center">
            <UserRound className="size-10 text-muted-foreground" />
            <p className="text-muted-foreground">{t("notSignedIn")}</p>
            <Button asChild>
              <Link href="/login">{t("signIn")}</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <ProfileCard session={session} locale={locale} copied={copied} onCopy={copyId} onLogout={handleLogout} />
      )}
    </main>
  )
}

function ProfileCard({
  session,
  locale,
  copied,
  onCopy,
  onLogout,
}: {
  session: AuthSession
  locale: string
  copied: boolean
  onCopy: (id: string) => void
  onLogout: () => void
}) {
  const t = useTranslations("Profile")
  const { user } = session

  const initials = (user.username || user.email || "?").slice(0, 2).toUpperCase()
  const memberSince = user.createdAt
    ? new Date(user.createdAt).toLocaleDateString(locale, { year: "numeric", month: "long", day: "numeric" })
    : t("unknown")

  const rows = [
    { icon: UserRound, label: t("username"), value: user.username || t("unknown") },
    { icon: AtSign, label: t("email"), value: user.email || t("unknown") },
    { icon: Calendar, label: t("memberSince"), value: memberSince },
  ]

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center gap-4">
          <span className="flex size-16 shrink-0 items-center justify-center rounded-full bg-primary text-xl font-semibold text-primary-foreground">
            {initials}
          </span>
          <div className="min-w-0">
            <CardTitle className="truncate text-xl">{user.username || t("unknown")}</CardTitle>
            <p className="truncate text-sm text-muted-foreground">{user.email}</p>
          </div>
        </CardHeader>

        <CardContent className="space-y-1">
          {rows.map((row) => (
            <div key={row.label} className="flex items-center gap-3 border-t border-border py-3 first:border-t-0">
              <row.icon className="size-4 shrink-0 text-muted-foreground" />
              <span className="w-32 shrink-0 text-sm text-muted-foreground">{row.label}</span>
              <span className="min-w-0 flex-1 truncate text-sm font-medium">{row.value}</span>
            </div>
          ))}

          <div className="flex items-center gap-3 border-t border-border py-3">
            <Fingerprint className="size-4 shrink-0 text-muted-foreground" />
            <span className="w-32 shrink-0 text-sm text-muted-foreground">{t("userId")}</span>
            <code className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">{user.id}</code>
            <Button
              variant="ghost"
              size="sm"
              className="shrink-0 gap-1.5"
              onClick={() => onCopy(user.id)}
              aria-label={t("copy")}
            >
              {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
              {copied ? t("copied") : t("copy")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button variant="destructive" onClick={onLogout} className="gap-2">
          <LogOut className="size-4" />
          {t("logout")}
        </Button>
      </div>
    </div>
  )
}
