import { useTranslations } from "next-intl"

import { GroupsManager } from "@/components/groups-manager"
import { SectionGuard } from "@/components/section-guard"

export default function GroupsPage() {
  const t = useTranslations("Groups")
  return (
    <SectionGuard section="groups">
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
          <p className="text-muted-foreground">{t("subtitle")}</p>
        </div>
        <GroupsManager />
      </div>
    </SectionGuard>
  )
}
