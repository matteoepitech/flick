"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Loader2, Plus } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { ApiError, createGroup, type AdminGroup } from "@/lib/api"

interface CreateGroupSheetProps {
  token: string
  onCreated: (group: AdminGroup) => void
}

export function CreateGroupSheet({ token, onCreated }: CreateGroupSheetProps) {
  const t = useTranslations("Groups")
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Reset the form each time the sheet opens so a cancelled creation never
  // leaks into the next one.
  function handleOpenChange(next: boolean) {
    if (next) {
      setName("")
      setError(null)
    }
    setOpen(next)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    const trimmed = name.trim()
    if (!trimmed) {
      setError(t("nameRequired"))
      return
    }

    setSaving(true)
    setError(null)
    try {
      const created = await createGroup(token, trimmed)
      onCreated(created)
      setOpen(false)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t("createError"))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetTrigger asChild>
        <Button>
          <Plus />
          {t("createAction")}
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t("createTitle")}</SheetTitle>
          <SheetDescription>{t("createSubtitle")}</SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          <div className="space-y-2">
            <Label htmlFor="create-group-name">{t("fieldName")}</Label>
            <Input
              id="create-group-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoComplete="off"
              autoFocus
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <SheetFooter>
            <Button type="submit" disabled={saving}>
              {saving && <Loader2 className="animate-spin" />}
              {t("create")}
            </Button>
            <SheetClose asChild>
              <Button type="button" variant="outline">
                {t("cancel")}
              </Button>
            </SheetClose>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
