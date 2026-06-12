"use client"

import { useEffect, useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"

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

const chartConfig = {
  activeCodes: {
    label: "Active codes",
    color: "var(--primary)",
  },
} satisfies ChartConfig

export default function DashboardPage() {
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
    { label: "Total uploads", value: latest ? String(latest.totalUploads) : null },
    { label: "Total downloads", value: latest ? String(latest.totalDownloads) : null },
    { label: "Active links", value: latest ? String(current) : null },
    { label: "Storage used", value: latest ? formatBytes(latest.storageBytes) : null },
    { label: "Users", value: latest ? String(latest.userCount) : null },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Overview</h2>
        <p className="text-muted-foreground">Quick glance at your Flick instance.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {summary.map((s) => (
          <Card key={s.label}>
            <CardHeader className="pb-2">
              <CardDescription>{s.label}</CardDescription>
              <CardTitle className="text-3xl">
                {s.value === null ? <Skeleton className="h-9 w-20" /> : s.value}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">Data will be wired to the Go API.</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Active codes: {current}</CardTitle>
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
  )
}
