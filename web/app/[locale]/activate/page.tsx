"use client"

import { ChevronLeft, CheckCircle2, MonitorSmartphone } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { Suspense, useEffect, useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ApiError, approveDevice } from "@/lib/api"
import { loadSession } from "@/lib/auth"
import { Link } from "@/i18n/navigation"
import type { AuthSession } from "@/lib/api"

// useSearchParams() forces this subtree to render on the client, so it must sit
// behind a Suspense boundary to keep the route prerenderable at build time.
export default function ActivatePage() {
  return (
    <Suspense>
      <ActivateForm />
    </Suspense>
  )
}

function ActivateForm() {
  const t = useTranslations("Activate")
  const searchParams = useSearchParams()

  const [session, setSession] = useState<AuthSession | null>(null)
  const [ready, setReady] = useState(false)
  const [userCode, setUserCode] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [approved, setApproved] = useState(false)

  // Resolve the signed-in session and pre-fill the code from the URL (the CLI
  // opens /activate?code=<user_code>).
  useEffect(() => {
    setSession(loadSession())
    setReady(true)
    const code = searchParams.get("code")
    if (code) setUserCode(code)
  }, [searchParams])

  const canSubmit = userCode.trim().length > 0 && !submitting && session !== null

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canSubmit || !session) return

    setSubmitting(true)
    setError(null)

    try {
      await approveDevice(userCode.trim(), session.token)
      setApproved(true)
    } catch (err) {
      console.error(err)
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
        <p className="mt-3 text-base text-muted-foreground">
          {t("description")}
        </p>
      </div>

      <Card className="mt-10 w-full gap-6 p-6">
        {!ready ? null : approved ? (
          <div className="flex flex-col items-center gap-3 text-center">
            <CheckCircle2 className="size-12 text-primary" />
            <p className="text-lg font-semibold">{t("approvedTitle")}</p>
            <p className="text-sm text-muted-foreground">
              {t("approvedBody")}
            </p>
          </div>
        ) : !session ? (
          <div className="flex flex-col items-center gap-3 text-center">
            <p className="text-sm text-muted-foreground">{t("notSignedIn")}</p>
            <Button asChild size="lg" className="h-12 w-full text-base font-semibold">
              <Link href="/login">{t("signIn")}</Link>
            </Button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-5 text-left">
            <div className="flex flex-col gap-2">
              <Label htmlFor="user-code" className="text-sm font-semibold text-foreground">
                {t("codeLabel")}
              </Label>
              <Input
                id="user-code"
                value={userCode}
                onChange={(event) => setUserCode(event.target.value)}
                placeholder={t("codePlaceholder")}
                autoComplete="off"
                autoFocus
                spellCheck={false}
              />
              <p className="text-xs text-muted-foreground">{t("signedInAs", { email: session.user.email })}</p>
            </div>

            {error && <p className="rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">{error}</p>}

            <Button type="submit" size="lg" className="h-12 w-full text-base font-semibold" disabled={!canSubmit}>
              <MonitorSmartphone className="size-5" />
              {submitting ? t("submitting") : t("submit")}
            </Button>
          </form>
        )}
      </Card>
    </main>
  )
}
