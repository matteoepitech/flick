import { useTranslations } from "next-intl"

import { SectionGuard } from "@/components/section-guard"
import { SettingsForm } from "@/components/settings-form"

export default function SettingsPage() {
  const t = useTranslations("Settings")
  return (
    <SectionGuard section="settings">
      <div className="mx-auto max-w-3xl space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
          <p className="text-muted-foreground">{t("subtitle")}</p>
        </div>
        <SettingsForm />
      </div>
    </SectionGuard>
  )
}
