"use client"

import { AlertTriangle, RefreshCw } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

type ErrorStateProps = {
  title: string
  description: string
  details?: string
  retryLabel?: string
  onRetry?: () => void
}

export function ErrorState({ title, description, details, retryLabel, onRetry }: ErrorStateProps) {
  return (
    <div className="w-full">
      <Card className="ring-destructive/30">
        <CardHeader>
          <div className="flex items-start gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-destructive/10 text-destructive">
              <AlertTriangle className="size-5" />
            </span>
            <div className="space-y-1">
              <CardTitle className="font-heading font-bold">{title}</CardTitle>
              <CardDescription>{description}</CardDescription>
            </div>
          </div>
        </CardHeader>
        {(details || onRetry) && (
          <CardContent className="space-y-4">
            {details && (
              <pre className="overflow-auto rounded-lg border border-border bg-muted p-3 font-mono text-xs break-words whitespace-pre-wrap">
                {details}
              </pre>
            )}
            {onRetry && retryLabel && (
              <div className="flex justify-end">
                <Button variant="outline" onClick={onRetry}>
                  <RefreshCw className="size-4" />
                  {retryLabel}
                </Button>
              </div>
            )}
          </CardContent>
        )}
      </Card>
    </div>
  )
}
