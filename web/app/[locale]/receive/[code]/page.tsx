"use client"

import { ChevronLeft, Download, FileText, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { use, useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { CodeNotFoundError, type DownloadedFile, downloadByCode, triggerBlobDownload } from "@/lib/api"
import { Link } from "@/i18n/navigation"

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 o"
  const units = ["o", "Ko", "Mo", "Go"]
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** i).toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

type State =
  | { status: "loading" }
  | { status: "ready"; files: DownloadedFile[] }
  | { status: "not_found" }
  | { status: "error" }

export default function ReceiveCodePage({ params }: { params: Promise<{ code: string }> }) {
  const t = useTranslations("ReceiveCode")
  const { code: rawCode } = use(params)
  const code = decodeURIComponent(rawCode)

  const [state, setState] = useState<State>({ status: "loading" })

  useEffect(() => {
    const controller = new AbortController()
    setState({ status: "loading" })

    downloadByCode(code, controller.signal)
      .then((files) => setState({ status: "ready", files }))
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        if (err instanceof CodeNotFoundError) {
          setState({ status: "not_found" })
        } else {
          console.error(err)
          setState({ status: "error" })
        }
      })

    return () => controller.abort()
  }, [code])

  return (
    <main className="mx-auto flex w-full max-w-xl flex-col items-center px-6 py-16">
      <Link
        href="/receive"
        className="mb-8 inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("back")}
      </Link>

      <div className="w-full text-center">
        <h1 className="text-3xl font-bold tracking-tight md:text-4xl">{t("title")}</h1>
        <p className="mt-3 font-mono text-sm text-muted-foreground">{t("subtitle", { code })}</p>
      </div>

      <div className="mt-10 w-full">
        {state.status === "loading" && (
          <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            {t("loading")}
          </div>
        )}

        {state.status === "not_found" && (
          <p className="rounded-lg bg-destructive/10 px-4 py-3 text-center text-sm text-destructive">{t("notFound")}</p>
        )}

        {state.status === "error" && (
          <p className="rounded-lg bg-destructive/10 px-4 py-3 text-center text-sm text-destructive">{t("error")}</p>
        )}

        {state.status === "ready" && (
          <div className="flex flex-col gap-4">
            <Card className="p-2">
              <ul className="flex flex-col">
                {state.files.map((file, index) => (
                  <li key={`${file.name}-${index}`} className="flex items-center gap-3 rounded-lg px-3 py-2.5">
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                      <FileText className="h-4 w-4" />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{file.name}</p>
                      <p className="text-xs text-muted-foreground">{formatBytes(file.size)}</p>
                    </div>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => triggerBlobDownload(file.blob, file.name)}
                    >
                      <Download className="size-4" />
                      {t("download")}
                    </Button>
                  </li>
                ))}
              </ul>
            </Card>

            {state.files.length > 1 && (
              <Button
                type="button"
                size="lg"
                className="h-12 w-full"
                onClick={() => state.files.forEach((file) => triggerBlobDownload(file.blob, file.name))}
              >
                <Download className="size-5" />
                {t("downloadAll")}
              </Button>
            )}
          </div>
        )}
      </div>
    </main>
  )
}
