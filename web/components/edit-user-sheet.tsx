"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Loader2, Pencil } from "lucide-react"

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
import { ApiError, updateUser, type AdminUser, type UserUpdate } from "@/lib/api"

interface EditUserSheetProps {
  user: AdminUser
  token: string
  onUpdated: (user: AdminUser) => void
}

export function EditUserSheet({ user, token, onUpdated }: EditUserSheetProps) {
  const t = useTranslations("Users")
  const [open, setOpen] = useState(false)
  const [username, setUsername] = useState(user.username)
  const [email, setEmail] = useState(user.email)
  const [password, setPassword] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function handleOpenChange(next: boolean) {
    if (next) {
      setUsername(user.username)
      setEmail(user.email)
      setPassword("")
      setError(null)
    }
    setOpen(next)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    const changes: UserUpdate = {}
    if (username.trim() && username.trim() !== user.username) changes.username = username.trim()
    if (email.trim() && email.trim() !== user.email) changes.email = email.trim()
    if (password) changes.password = password

    if (Object.keys(changes).length === 0) {
      setOpen(false)
      return
    }

    setSaving(true)
    setError(null)
    try {
      const updated = await updateUser(token, user.id, changes)
      onUpdated(updated)
      setOpen(false)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t("editError"))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t("editAction")}>
          <Pencil />
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle className="font-heading text-xl font-bold">{t("editTitle")}</SheetTitle>
          <SheetDescription>{t("editSubtitle", { username: user.username })}</SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          <div className="space-y-2">
            <Label htmlFor="edit-username">{t("fieldUsername")}</Label>
            <Input
              id="edit-username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="off"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-email">{t("fieldEmail")}</Label>
            <Input
              id="edit-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="off"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-password">{t("fieldPassword")}</Label>
            <Input
              id="edit-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t("passwordPlaceholder")}
              autoComplete="new-password"
            />
            <p className="text-xs text-muted-foreground">{t("passwordHint")}</p>
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <SheetFooter>
            <Button type="submit" disabled={saving}>
              {saving && <Loader2 className="animate-spin" />}
              {t("save")}
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
