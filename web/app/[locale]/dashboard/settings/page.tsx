import { useTranslations } from "next-intl"

import { SectionGuard } from "@/components/section-guard"
import { SettingsForm } from "@/components/settings-form"

export default function SettingsPage() {
  const t = useTranslations("Settings")
  return (
    <SectionGuard section="settings">
      <div className="mx-auto max-w-3xl space-y-6">
        <h2 className="font-heading text-3xl font-bold tracking-tight">{t("title")}</h2>
        <SettingsForm />
      </div>
    </SectionGuard>
  )
}
