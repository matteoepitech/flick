import { ArrowDownLeft, ArrowUpRight } from "lucide-react"
import { useTranslations } from "next-intl"

import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { Link } from "@/i18n/navigation"

export default function SiteHeader() {
  const t = useTranslations("Header")

  return (
    <header className="border-b">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
        <Link href="/" className="flex items-center gap-2">
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <ArrowUpRight className="h-5 w-5" />
          </span>
          <span className="text-lg font-semibold">flick</span>
        </Link>

        <nav className="flex items-center gap-4">
          <Button asChild>
            <Link href="/send">
              <ArrowUpRight className="h-4 w-4" />
              {t("send")}
            </Link>
          </Button>
          <Button asChild variant="outline">
            <Link href="/receive">
              <ArrowDownLeft className="h-4 w-4" />
              {t("receive")}
            </Link>
          </Button>
          <ThemeToggle />
        </nav>
      </div>
    </header>
  )
}
