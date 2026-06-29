"use client"

import { Activity, ArrowDownToLine, ArrowUpFromLine, HardDrive, Link2, Users } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"

import { SectionGuard } from "@/components/section-guard"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import { Skeleton } from "@/components/ui/skeleton"
import { fetchStats, type StatsSnapshot } from "@/lib/api"

type Point = { time: string; activeCodes: number }

const MAX_POINTS = 60
const POLL_INTERVAL_MS = 5000

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
      } catch {}
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
    {
      key: "totalUploads",
      label: t("totalUploads"),
      value: latest ? String(latest.totalUploads) : null,
      icon: ArrowUpFromLine,
    },
    {
      key: "totalDownloads",
      label: t("totalDownloads"),
      value: latest ? String(latest.totalDownloads) : null,
      icon: ArrowDownToLine,
    },
    { key: "activeLinks", label: t("activeLinks"), value: latest ? String(current) : null, icon: Link2 },
    {
      key: "storageUsed",
      label: t("storageUsed"),
      value: latest ? formatBytes(latest.storageBytes) : null,
      icon: HardDrive,
    },
    { key: "usersCount", label: t("usersCount"), value: latest ? String(latest.userCount) : null, icon: Users },
  ]

  return (
    <SectionGuard section="overview">
      <div className="space-y-6">
        <h2 className="font-heading text-3xl font-bold tracking-tight">{t("title")}</h2>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
          {summary.map((s) => {
            const Icon = s.icon
            return (
              <div key={s.key} className="rounded-xl border border-border bg-card p-5">
                <div className="flex items-center justify-between text-muted-foreground">
                  <span className="text-sm">{s.label}</span>
                  <Icon className="size-4" />
                </div>
                <div className="mt-3 font-heading text-3xl font-bold">
                  {s.value === null ? <Skeleton className="h-9 w-20" /> : <span>{s.value}</span>}
                </div>
              </div>
            )
          })}
        </div>

        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <Activity className="size-4 text-primary" />
            <CardTitle className="font-heading font-bold">{t("activeCodesValue", { count: current })}</CardTitle>
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
