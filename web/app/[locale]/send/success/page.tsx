"use client"

import { Check, ChevronLeft, Copy, ExternalLink } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { Suspense, useState } from "react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Link } from "@/i18n/navigation"

export default function SendSuccessPage() {
  return (
    <Suspense fallback={null}>
      <SendSuccessContent />
    </Suspense>
  )
}

function SendSuccessContent() {
  const t = useTranslations("SendSuccess")
  const searchParams = useSearchParams()
  const codes = searchParams.getAll("code").filter(Boolean)
  const exp = searchParams.get("exp")
  const [copied, setCopied] = useState<string | null>(null)

  async function copyCode(code: string) {
    await navigator.clipboard.writeText(code)
    setCopied(code)
    setTimeout(() => setCopied((current) => (current === code ? null : current)), 2000)
  }

  return (
    <main className="mx-auto flex w-full max-w-xl flex-col items-center px-6 py-16">
      <Link
        href="/send"
        className="mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("back")}
      </Link>

      <div className="w-full text-center">
        <p className="font-heading text-xs font-semibold tracking-[0.12em] text-primary uppercase">{t("eyebrow")}</p>
        <h1 className="mt-4 font-heading text-3xl font-bold tracking-tight md:text-4xl">{t("title")}</h1>
        <p className="mt-3 text-base text-muted-foreground">{t("description")}</p>
      </div>

      <div className="mt-10 flex w-full flex-col gap-4">
        {codes.length === 0 && (
          <p className="rounded-xl border border-border bg-muted px-4 py-3 text-center text-sm text-muted-foreground">
            {t("noCode")}
          </p>
        )}

        {codes.map((code) => {
          const isCopied = copied === code
          return (
            <Card key={code} className="gap-4 p-6">
              <p className="font-heading text-xs font-semibold tracking-[0.12em] text-muted-foreground uppercase">
                {t("codeLabel")}
              </p>
              <p className="rounded-xl border border-dashed border-primary/30 bg-primary/5 px-5 py-5 text-center font-mono text-3xl font-bold tracking-[0.15em] break-all text-primary">
                {code}
              </p>
              {exp && <p className="text-center font-mono text-sm text-muted-foreground">{t("expiresIn", { exp })}</p>}
              <div className="flex flex-col gap-2 sm:flex-row">
                <Button
                  type="button"
                  variant="outline"
                  size="lg"
                  className="h-11 flex-1"
                  onClick={() => copyCode(code)}
                >
                  {isCopied ? <Check className="size-4" /> : <Copy className="size-4" />}
                  {isCopied ? t("copied") : t("copy")}
                </Button>
                <Button asChild size="lg" className="h-11 flex-1">
                  <Link href={`/receive/${encodeURIComponent(code)}`}>
                    <ExternalLink className="size-4" />
                    {t("openLink")}
                  </Link>
                </Button>
              </div>
            </Card>
          )
        })}
      </div>
    </main>
  )
}
