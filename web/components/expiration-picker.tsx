"use client"

import { useTranslations } from "next-intl"
import { type KeyboardEvent, useMemo, useState } from "react"

import { parseDurationMinutes } from "@/lib/api"
import { cn } from "@/lib/utils"

interface ExpirationPickerProps {
  value: string
  onChange: (value: string) => void
  maxExpiration?: string
}

const PRESETS = ["30m", "1h", "2h", "4h", "24h", "7d"]

const DURATION_PATTERN = /^\d+[mhd]$/

export function ExpirationPicker({ value, onChange, maxExpiration }: ExpirationPickerProps) {
  const t = useTranslations("Send")
  const presetSet = new Set(PRESETS)
  const [custom, setCustom] = useState(!presetSet.has(value))

  const maxMinutes = useMemo(() => (maxExpiration ? parseDurationMinutes(maxExpiration) : Infinity), [maxExpiration])

  const availablePresets = useMemo(() => PRESETS.filter((p) => parseDurationMinutes(p) <= maxMinutes), [maxMinutes])

  const overMax = Boolean(value && maxExpiration && parseDurationMinutes(value) > maxMinutes)

  function handlePreset(preset: string) {
    setCustom(false)
    onChange(preset)
  }

  function handleCustomToggle() {
    setCustom(true)
    onChange("")
  }

  function handleCustomInput(val: string) {
    onChange(val)
  }

  function handleCustomBlur(val: string) {
    if (val && DURATION_PATTERN.test(val) && !(maxExpiration && parseDurationMinutes(val) > maxMinutes)) {
      onChange(val)
    }
  }

  function handleCustomKeyDown(event: KeyboardEvent<HTMLInputElement>, val: string) {
    if (
      event.key === "Enter" &&
      DURATION_PATTERN.test(val) &&
      !(maxExpiration && parseDurationMinutes(val) > maxMinutes)
    ) {
      onChange(val)
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-2">
        {availablePresets.map((preset) => {
          const active = !custom && value === preset
          return (
            <button
              key={preset}
              type="button"
              onClick={() => handlePreset(preset)}
              className={cn(
                "rounded-full border px-3.5 py-1.5 font-mono text-sm font-medium transition-colors",
                active
                  ? "border-primary bg-primary/8 text-primary"
                  : "border-border text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
            >
              {preset}
            </button>
          )
        })}
        <button
          type="button"
          onClick={handleCustomToggle}
          className={cn(
            "rounded-full border px-3.5 py-1.5 text-sm font-medium transition-colors",
            custom
              ? "border-primary bg-primary/8 text-primary"
              : "border-border text-muted-foreground hover:bg-muted hover:text-foreground"
          )}
        >
          {t("expirationCustom")}
        </button>
      </div>
      {custom && (
        <div className="relative">
          <input
            type="text"
            value={value}
            onChange={(e) => handleCustomInput(e.target.value)}
            onBlur={(e) => handleCustomBlur(e.target.value)}
            onKeyDown={(e) => handleCustomKeyDown(e, value)}
            placeholder={t("expirationCustomPlaceholder")}
            className={cn(
              "h-9 w-full rounded-lg border bg-background px-3 text-sm transition-colors outline-none placeholder:text-muted-foreground",
              value && !DURATION_PATTERN.test(value)
                ? "border-destructive/50 focus:border-destructive"
                : "border-border focus:border-primary"
            )}
            autoFocus
          />
          {value && !DURATION_PATTERN.test(value) && (
            <p className="mt-1 text-xs text-destructive">{t("expirationCustomInvalid")}</p>
          )}
        </div>
      )}
      {overMax && maxExpiration && (
        <p className="text-xs text-destructive">{t("expirationMaxExceeded", { max: maxExpiration })}</p>
      )}
    </div>
  )
}
