import { DashboardGuard } from "@/components/dashboard-guard"
import { DashboardSidebar } from "@/components/dashboard-sidebar"
import { ThemeToggle } from "@/components/theme-toggle"
import { Separator } from "@/components/ui/separator"
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { TooltipProvider } from "@/components/ui/tooltip"

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <DashboardGuard>
      <TooltipProvider>
        <SidebarProvider>
          <DashboardSidebar />
          <SidebarInset>
            <header className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b bg-background px-4">
              <SidebarTrigger />
              <Separator orientation="vertical" className="mr-2 h-4" />
              <h1 className="text-sm font-medium">Dashboard</h1>
              <div className="ml-auto">
                <ThemeToggle />
              </div>
            </header>
            <main className="flex-1 p-6">{children}</main>
          </SidebarInset>
        </SidebarProvider>
      </TooltipProvider>
    </DashboardGuard>
  )
}
