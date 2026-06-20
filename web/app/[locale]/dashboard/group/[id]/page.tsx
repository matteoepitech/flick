import { MyGroupManager } from "@/components/my-group-manager"
import { SectionGuard } from "@/components/section-guard"

export default async function MyGroupDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return (
    <SectionGuard section="group">
      <MyGroupManager groupId={id} />
    </SectionGuard>
  )
}
