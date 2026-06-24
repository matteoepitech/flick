"use client"

import { useTranslations } from "next-intl"
import { useEffect, useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"

import { SectionGuard } from "@/components/section-guard"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import { Skeleton } from "@/components/ui/skeleton"
import { fetchStats, type StatsSnapshot } from "@/lib/api"

type Point = { time: string; activeCodes: number }

const MAX_POINTS = 60
const POLL_INTERVAL_MS = 5000

// formatBytes: Render a byte count as a human-readable size (e.g. "1.5 MB").
function formatBytes(bytes: number): string {
  if (bytes <= 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / Math.pow(1024, exponent)
  return `${value.toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`
}

export default function DashboardPage() {
  const t = useTranslations("Dashboard")
  const chartConfig = {
    activeCodes: {
      label: t("activeCodes"),
      color: "var(--primary)",
    },
  } satisfies ChartConfig
  const [points, setPoints] = useState<Point[]>([])
  const [latest, setLatest] = useState<StatsSnapshot | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    let cancelled = false

    const tick = async () => {
      try {
        const snap = await fetchStats(controller.signal)
        if (cancelled) return
        setLatest(snap)
        setPoints((prev) => [
          ...prev.slice(-(MAX_POINTS - 1)),
          {
            time: new Date(snap.timestamp).toLocaleTimeString(),
            activeCodes: snap.activeCodes,
          },
        ])
      } catch {
        /* ignore transient errors */
      }
    }

    tick()
    const id = setInterval(tick, POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      controller.abort()
      clearInterval(id)
    }
  }, [])

  const current = points.at(-1)?.activeCodes ?? 0

  const summary = [
    { key: "totalUploads", label: t("totalUploads"), value: latest ? String(latest.totalUploads) : null },
    { key: "totalDownloads", label: t("totalDownloads"), value: latest ? String(latest.totalDownloads) : null },
    { key: "activeLinks", label: t("activeLinks"), value: latest ? String(current) : null },
    { key: "storageUsed", label: t("storageUsed"), value: latest ? formatBytes(latest.storageBytes) : null },
    { key: "usersCount", label: t("usersCount"), value: latest ? String(latest.userCount) : null },
  ]

  return (
    <SectionGuard section="overview">
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
        <p className="text-muted-foreground">{t("subtitle")}</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {summary.map((s) => (
          <Card key={s.key}>
            <CardHeader className="pb-2">
              <CardDescription>{s.label}</CardDescription>
              <CardTitle className="text-3xl">
                {s.value === null ? <Skeleton className="h-9 w-20" /> : s.value}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">{t("cardPlaceholder")}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("activeCodesValue", { count: current })}</CardTitle>
        </CardHeader>
        <CardContent>
          <ChartContainer config={chartConfig} className="h-64 w-full">
            <LineChart data={points} margin={{ left: 12, right: 12 }}>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="time" tickLine={false} axisLine={false} minTickGap={32} />
              <YAxis allowDecimals={false} tickLine={false} axisLine={false} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Line
                dataKey="activeCodes"
                type="monotone"
                stroke="var(--color-activeCodes)"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ChartContainer>
        </CardContent>
      </Card>
    </div>
    </SectionGuard>
  )
}
