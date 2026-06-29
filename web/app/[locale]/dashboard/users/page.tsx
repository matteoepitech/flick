import { useTranslations } from "next-intl"

import { SectionGuard } from "@/components/section-guard"
import { UsersManager } from "@/components/users-manager"

export default function UsersPage() {
  const t = useTranslations("Users")
  return (
    <SectionGuard section="users">
      <div className="space-y-6">
        <h2 className="font-heading text-3xl font-bold tracking-tight">{t("title")}</h2>
        <UsersManager />
      </div>
    </SectionGuard>
  )
}
