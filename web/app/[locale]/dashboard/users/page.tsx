import { useTranslations } from "next-intl"

import { SectionGuard } from "@/components/section-guard"
import { UsersManager } from "@/components/users-manager"

export default function UsersPage() {
  const t = useTranslations("Users")
  return (
    <SectionGuard section="users">
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
          <p className="text-muted-foreground">{t("subtitle")}</p>
        </div>
        <UsersManager />
      </div>
    </SectionGuard>
  )
}
