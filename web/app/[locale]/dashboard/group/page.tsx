import { useTranslations } from "next-intl"

import { MyGroupsList } from "@/components/my-groups-list"
import { SectionGuard } from "@/components/section-guard"

export default function GroupPage() {
  const t = useTranslations("MyGroups")
  return (
    <SectionGuard section="group">
      <div className="space-y-6">
        <div>
          <p className="mb-2 font-heading text-xs font-semibold tracking-[0.12em] text-muted-foreground uppercase">
            {t("eyebrow")}
          </p>
          <h2 className="font-heading text-3xl font-bold tracking-tight">{t("title")}</h2>
        </div>
        <MyGroupsList />
      </div>
    </SectionGuard>
  )
}
