"use client"

import { useCallback, useEffect, useState } from "react"
import { useTranslations } from "next-intl"

import { ErrorState } from "@/components/error-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"
import { ApiError, getConfigureUrl, loadConfiguration, saveConfiguration } from "@/lib/api"
import { type SettingField, type SettingSection, settingsSections } from "@/lib/settings-config"

type Values = Record<string, string | number | boolean>

type LoadError = {
  message: string
  url: string
  status?: number
}

function buildInitialValues(): Values {
  const values: Values = {}
  for (const section of settingsSections) {
    for (const field of section.fields) {
      values[field.key] = field.defaultValue
    }
  }
  return values
}

export function SettingsForm() {
  const t = useTranslations("Settings")
  const [values, setValues] = useState<Values>(buildInitialValues)
  const [savedAt, setSavedAt] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<LoadError | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  const retry = useCallback(() => {
    setReloadKey((k) => k + 1)
  }, [])

  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    setLoadError(null)
    loadConfiguration(ctrl.signal)
      .then((remote) => {
        setValues((prev) => {
          const merged: Values = { ...prev }
          for (const key of Object.keys(prev)) {
            if (Object.prototype.hasOwnProperty.call(remote, key)) {
              merged[key] = remote[key]
            }
          }
          return merged
        })
      })
      .catch((err) => {
        if (err instanceof Error && err.name === "AbortError") return
        console.error("loadConfiguration failed", err)
        setLoadError({
          message: err instanceof Error ? err.message : String(err),
          url: getConfigureUrl(),
          status: err instanceof ApiError ? err.status : undefined,
        })
      })
      .finally(() => setLoading(false))
    return () => ctrl.abort()
  }, [reloadKey])

  const update = (key: string, value: string | number | boolean) => {
    setValues((prev) => ({ ...prev, [key]: value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setSaveError(null)
    try {
      await saveConfiguration(values)
      setSavedAt(new Date().toLocaleTimeString())
    } catch (err) {
      console.error("saveConfiguration failed", err)
      setSaveError(t("saveError"))
    } finally {
      setSaving(false)
    }
  }

  if (loadError) {
    const details = [`GET ${loadError.url}`, loadError.status ? `HTTP ${loadError.status}` : null, loadError.message]
      .filter(Boolean)
      .join("\n")

    return (
      <ErrorState
        title={t("loadErrorTitle")}
        description={t("loadErrorDescription")}
        details={details}
        retryLabel={t("retry")}
        onRetry={retry}
      />
    )
  }

  if (loading) {
    return <SettingsFormSkeleton />
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {settingsSections.map((section) => (
        <SectionCard key={section.id} section={section} values={values} onUpdate={update} />
      ))}

      <div className="flex items-center justify-end gap-3">
        {saveError && <span className="text-sm text-destructive">{saveError}</span>}
        {!saveError && savedAt && (
          <span className="text-sm text-muted-foreground">{t("savedAt", { time: savedAt })}</span>
        )}
        <Button type="submit" disabled={saving}>
          {saving ? t("saving") : t("save")}
        </Button>
      </div>
    </form>
  )
}

function SectionCard({
  section,
  values,
  onUpdate,
}: {
  section: SettingSection
  values: Values
  onUpdate: (key: string, value: string | number | boolean) => void
}) {
  const t = useTranslations("Settings")
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t(`sections.${section.id}.title`)}</CardTitle>
        {section.hasDescription && <CardDescription>{t(`sections.${section.id}.description`)}</CardDescription>}
      </CardHeader>
      <CardContent className="divide-y divide-border py-0">
        {section.fields.map((field) => {
          if (field.dependsOn && values[field.dependsOn.key] !== field.dependsOn.equals) {
            return null
          }
          return <FieldRow key={field.key} field={field} value={values[field.key]} onUpdate={onUpdate} />
        })}
      </CardContent>
    </Card>
  )
}

function FieldRow({
  field,
  value,
  onUpdate,
}: {
  field: SettingField
  value: string | number | boolean
  onUpdate: (key: string, value: string | number | boolean) => void
}) {
  const t = useTranslations("Settings")
  const label = t(`fields.${field.key}.label`)
  const description = field.hasDescription ? t(`fields.${field.key}.description`) : null
  const disabled = Boolean(field.notAvailable)
  const wrapperClass = disabled ? "pointer-events-none opacity-50 select-none" : ""
  const unavailableBadge = disabled ? (
    <span className="ml-2 rounded-full bg-muted px-2 py-0.5 font-heading text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase">
      {t("notAvailable")}
    </span>
  ) : null

  const labelBlock = (
    <div className="space-y-1 sm:flex-1">
      <Label htmlFor={field.key} className="font-medium">
        {label}
        {unavailableBadge}
      </Label>
      {description && <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>}
    </div>
  )

  if (field.type === "switch") {
    return (
      <div className={`flex items-center justify-between gap-4 py-5 first:pt-5 last:pb-5 ${wrapperClass}`}>
        {labelBlock}
        <Switch
          id={field.key}
          checked={Boolean(value)}
          disabled={disabled}
          onCheckedChange={(checked) => onUpdate(field.key, checked)}
        />
      </div>
    )
  }

  return (
    <div
      className={`flex flex-col gap-3 py-5 first:pt-5 last:pb-5 sm:flex-row sm:items-start sm:justify-between sm:gap-6 ${wrapperClass}`}
    >
      {labelBlock}
      <div className="sm:w-64 sm:shrink-0">
        {field.type === "textarea" ? (
          <Textarea
            id={field.key}
            value={String(value ?? "")}
            placeholder={field.placeholder}
            disabled={disabled}
            onChange={(e) => onUpdate(field.key, e.target.value)}
          />
        ) : field.type === "select" ? (
          <select
            id={field.key}
            value={String(value ?? "")}
            disabled={disabled}
            onChange={(e) => onUpdate(field.key, e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {field.options?.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        ) : (
          <Input
            id={field.key}
            type={field.type === "number" ? "number" : "text"}
            value={String(value ?? "")}
            placeholder={field.placeholder}
            disabled={disabled}
            className={field.type === "number" ? "tabular-nums" : undefined}
            onChange={(e) => onUpdate(field.key, field.type === "number" ? Number(e.target.value) : e.target.value)}
          />
        )}
      </div>
    </div>
  )
}

function SettingsFormSkeleton() {
  return (
    <div className="space-y-6" aria-busy="true">
      {settingsSections.map((section) => (
        <Card key={section.id}>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
            {section.hasDescription && <Skeleton className="mt-2 h-4 w-72 max-w-full" />}
          </CardHeader>
          <CardContent className="space-y-5">
            {section.fields.map((field) =>
              field.type === "switch" ? (
                <div key={field.key} className="flex items-center justify-between gap-4">
                  <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-48 max-w-full" />
                    {field.hasDescription && <Skeleton className="h-3 w-64 max-w-full" />}
                  </div>
                  <Skeleton className="h-5 w-9 rounded-full" />
                </div>
              ) : (
                <div key={field.key} className="space-y-2">
                  <Skeleton className="h-4 w-40 max-w-full" />
                  {field.hasDescription && <Skeleton className="h-3 w-72 max-w-full" />}
                  <Skeleton className="h-9 w-full" />
                </div>
              )
            )}
          </CardContent>
        </Card>
      ))}

      <div className="flex items-center justify-end gap-3">
        <Skeleton className="h-9 w-32" />
      </div>
    </div>
  )
}
