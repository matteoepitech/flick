import { SectionGuard } from "@/components/section-guard"

export default function GroupPage() {
  return (
    <SectionGuard section="group">
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">My group</h2>
          <p className="text-muted-foreground">Manage the group you maintain.</p>
        </div>
      </div>
    </SectionGuard>
  )
}
