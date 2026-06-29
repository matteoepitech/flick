import { ArrowDownLeft, ArrowUpRight } from "lucide-react"
import { useTranslations } from "next-intl"

import { DashboardGuard } from "@/components/dashboard-guard"
import { DashboardSidebar } from "@/components/dashboard-sidebar"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { TooltipProvider } from "@/components/ui/tooltip"
import { Link } from "@/i18n/navigation"

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const t = useTranslations("Sidebar")
  const tHeader = useTranslations("Header")
  return (
    <DashboardGuard>
      <TooltipProvider>
        <SidebarProvider>
          <DashboardSidebar />
          <SidebarInset>
            <header className="sticky top-0 z-10 flex h-16 items-center gap-2 border-b bg-background/80 px-5 backdrop-blur-xl sm:px-6">
              <SidebarTrigger className="-ml-1.5" />
              <h1 className="font-heading text-lg font-bold tracking-tight">{t("headerTitle")}</h1>
              <div className="ml-auto flex items-center gap-2">
                <Button asChild variant="outline" size="sm" className="rounded-full">
                  <Link href="/receive">
                    <ArrowDownLeft className="size-4" />
                    <span className="hidden sm:inline">{tHeader("receive")}</span>
                  </Link>
                </Button>
                <Button asChild size="sm" className="rounded-full">
                  <Link href="/send">
                    <ArrowUpRight className="size-4" />
                    <span className="hidden sm:inline">{tHeader("send")}</span>
                  </Link>
                </Button>
                <ThemeToggle />
              </div>
            </header>
            <main className="flex-1 p-6 sm:p-8">{children}</main>
          </SidebarInset>
        </SidebarProvider>
      </TooltipProvider>
    </DashboardGuard>
  )
}
