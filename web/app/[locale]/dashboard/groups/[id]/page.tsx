import { GroupMembersManager } from "@/components/group-members-manager"
import { SectionGuard } from "@/components/section-guard"

export default async function GroupDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return (
    <SectionGuard section="groups">
      <GroupMembersManager groupId={id} />
    </SectionGuard>
  )
}
