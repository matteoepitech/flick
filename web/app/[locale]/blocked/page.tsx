"use client"

import { Ban, UserRound } from "lucide-react"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { clearSession } from "@/lib/auth"
import { Link, useRouter } from "@/i18n/navigation"

export default function BlockedPage() {
  const t = useTranslations("Blocked")
  const router = useRouter()

  function handleLogout() {
    clearSession()
    router.replace("/login")
  }

  return (
    <main className="mx-auto flex min-h-[60vh] w-full max-w-md flex-col items-center justify-center px-6 py-16">
      <Card className="w-full border-destructive/40">
        <CardHeader>
          <div className="flex flex-col items-center gap-3 text-center">
            <span className="flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
              <Ban className="size-6" />
            </span>
            <CardTitle className="font-heading text-xl font-bold">{t("title")}</CardTitle>
            <CardDescription>{t("description")}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-2">
          <Button asChild variant="outline" className="w-full">
            <Link href="/profile">
              <UserRound className="h-4 w-4" />
              {t("profile")}
            </Link>
          </Button>
          <Button variant="ghost" onClick={handleLogout} className="w-full">
            {t("logout")}
          </Button>
        </CardContent>
      </Card>
    </main>
  )
}
