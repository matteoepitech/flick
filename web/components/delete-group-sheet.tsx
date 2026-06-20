"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Loader2, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
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
import { ApiError, deleteGroup, type AdminGroup } from "@/lib/api"

interface DeleteGroupSheetProps {
  group: AdminGroup
  token: string
  onDeleted: (id: string) => void
}

export function DeleteGroupSheet({ group, token, onDeleted }: DeleteGroupSheetProps) {
  const t = useTranslations("Groups")
  const [open, setOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function handleOpenChange(next: boolean) {
    if (next) setError(null)
    setOpen(next)
  }

  async function handleConfirm() {
    setDeleting(true)
    setError(null)
    try {
      await deleteGroup(token, group.id)
      onDeleted(group.id)
      setOpen(false)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t("deleteError", { name: group.name }))
    } finally {
      setDeleting(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t("deleteAction")}>
          <Trash2 className="text-destructive" />
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>{t("deleteTitle")}</SheetTitle>
          <SheetDescription>{t("deleteSubtitle", { name: group.name })}</SheetDescription>
        </SheetHeader>

        <div className="flex flex-1 flex-col gap-4 px-4">
          {error && <p className="text-sm text-destructive">{error}</p>}

          <SheetFooter>
            <Button type="button" variant="destructive" onClick={handleConfirm} disabled={deleting}>
              {deleting && <Loader2 className="animate-spin" />}
              {t("confirmDelete")}
            </Button>
            <SheetClose asChild>
              <Button type="button" variant="outline">
                {t("cancel")}
              </Button>
            </SheetClose>
          </SheetFooter>
        </div>
      </SheetContent>
    </Sheet>
  )
}
