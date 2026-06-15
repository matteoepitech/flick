import { SectionGuard } from "@/components/section-guard"

export default function UsersPage() {
  return (
    <SectionGuard section="users">
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Users</h2>
          <p className="text-muted-foreground">User management</p>
        </div>
      </div>
    </SectionGuard>
  )
}
