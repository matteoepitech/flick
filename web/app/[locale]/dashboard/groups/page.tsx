import { useTranslations } from "next-intl"

import { GroupsManager } from "@/components/groups-manager"
import { SectionGuard } from "@/components/section-guard"

export default function GroupsPage() {
  const t = useTranslations("Groups")
  return (
    <SectionGuard section="groups">
      <div className="space-y-6">
        <h2 className="font-heading text-3xl font-bold tracking-tight">{t("title")}</h2>
        <GroupsManager />
      </div>
    </SectionGuard>
  )
}
