"use client"

import { ChevronLeft, Eye, EyeOff, LogIn } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ApiError, isAccountBlocked, loginUser } from "@/lib/api"
import { saveSession } from "@/lib/auth"
import { canAccessDashboard, landingPath } from "@/lib/permissions"
import { Link, useRouter } from "@/i18n/navigation"

export default function LoginPage() {
  const t = useTranslations("Login")
  const router = useRouter()

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const canSubmit = email.trim().length > 0 && password.length > 0 && !submitting

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canSubmit) return

    setSubmitting(true)
    setError(null)

    try {
      const session = await loginUser(email.trim(), password)
      saveSession(session)
      // A blocked account still logs in (so it keeps a session for the profile
      // page) but is sent straight to the blocked page.
      if (session.user.blocked) {
        router.replace("/blocked")
        return
      }
      // Admins and maintainers land in the dashboard; everyone else goes home.
      router.push(canAccessDashboard(session.user) ? landingPath(session.user) : "/")
    } catch (err) {
      console.error(err)
      // A blocked account is rejected at login: send them to the blocked page
      // rather than showing a raw credentials error.
      if (isAccountBlocked(err)) {
        router.replace("/blocked")
        return
      }
      setError(err instanceof ApiError && err.message ? err.message : t("error"))
      setSubmitting(false)
    }
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-md flex-col items-center justify-center px-6 py-16">
      <Link
        href="/"
        className="mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("back")}
      </Link>

      <div className="w-full text-center">
        <h1 className="text-3xl font-bold tracking-tight md:text-4xl">{t("title")}</h1>
        <p className="mt-3 text-base text-muted-foreground">{t("description")}</p>
      </div>

      <Card className="mt-10 w-full gap-6 p-6">
        <form onSubmit={handleSubmit} className="flex flex-col gap-5 text-left">
          <div className="flex flex-col gap-2">
            <Label htmlFor="email" className="text-sm font-semibold text-foreground">
              {t("email")}
            </Label>
            <Input
              id="email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder={t("emailPlaceholder")}
              autoComplete="email"
              autoFocus
              spellCheck={false}
            />
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="password" className="text-sm font-semibold text-foreground">
              {t("password")}
            </Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? "text" : "password"}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder={t("passwordPlaceholder")}
                autoComplete="current-password"
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPassword((value) => !value)}
                aria-label={showPassword ? t("hidePassword") : t("showPassword")}
                className="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>

          {error && <p className="rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">{error}</p>}

          <Button type="submit" size="lg" className="h-12 w-full text-base font-semibold" disabled={!canSubmit}>
            <LogIn className="size-5" />
            {submitting ? t("submitting") : t("submit")}
          </Button>
        </form>
      </Card>

      <p className="mt-6 text-sm text-muted-foreground">
        {t("noAccount")}{" "}
        <Link href="/register" className="font-medium text-primary underline-offset-2 hover:underline">
          {t("registerLink")}
        </Link>
      </p>
    </main>
  )
}
