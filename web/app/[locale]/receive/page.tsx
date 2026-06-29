"use client"

import { ChevronLeft } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Link, useRouter } from "@/i18n/navigation"

export default function ReceivePage() {
  const t = useTranslations("Receive")
  const router = useRouter()
  const [code, setCode] = useState("")
  const [submitting, setSubmitting] = useState(false)

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const trimmed = code.trim()
    if (!trimmed) return
    setSubmitting(true)
    router.push(`/receive/${encodeURIComponent(trimmed)}`)
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-xl flex-col items-center justify-center px-6 py-16">
      <Link
        href="/"
        className="mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("back")}
      </Link>

      <div className="w-full text-center">
        <h1 className="font-heading text-3xl font-bold tracking-tight md:text-4xl">{t("title")}</h1>
        <p className="mt-3 text-base text-muted-foreground">{t("description")}</p>

        <form onSubmit={handleSubmit} className="mt-10 flex flex-col gap-5">
          <Input
            value={code}
            onChange={(event) => setCode(event.target.value.toLowerCase())}
            placeholder={t("placeholder")}
            autoFocus
            spellCheck={false}
            aria-label={t("title")}
            className="h-16 text-center font-mono text-lg tracking-[0.15em] text-primary md:text-xl"
          />

          <Button
            type="submit"
            size="lg"
            className="h-14 w-full text-base font-semibold"
            disabled={submitting || code.trim().length === 0}
          >
            {submitting ? t("loading") : t("submit")}
          </Button>
        </form>
      </div>
    </main>
  )
}
