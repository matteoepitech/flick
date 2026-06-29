"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Loader2, UserPlus } from "lucide-react"

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
import { ApiError, registerUser, type AdminUser } from "@/lib/api"

interface CreateUserSheetProps {
  onCreated: (user: AdminUser) => void
}

export function CreateUserSheet({ onCreated }: CreateUserSheetProps) {
  const t = useTranslations("Users")

  const tRegister = useTranslations("Register")
  const [open, setOpen] = useState(false)
  const [username, setUsername] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function handleOpenChange(next: boolean) {
    if (next) {
      setUsername("")
      setEmail("")
      setPassword("")
      setConfirmPassword("")
      setError(null)
    }
    setOpen(next)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    const name = username.trim()
    const mail = email.trim()
    if (!name || !mail || !password) {
      setError(t("createValidation"))
      return
    }

    if (password !== confirmPassword) {
      setError(tRegister("passwordMismatch"))
      return
    }

    setSaving(true)
    setError(null)
    try {
      const created = await registerUser(name, mail, password)
      onCreated({
        id: created.id,
        username: created.username,
        email: created.email,
        role: created.role,
        blocked: created.blocked,
        createdAt: created.createdAt,
      })
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
          <UserPlus />
          {t("createAction")}
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle className="font-heading text-xl font-bold">{t("createTitle")}</SheetTitle>
          <SheetDescription>{t("createSubtitle")}</SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          <div className="space-y-2">
            <Label htmlFor="create-username">{t("fieldUsername")}</Label>
            <Input
              id="create-username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="off"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-email">{t("fieldEmail")}</Label>
            <Input
              id="create-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="off"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-password">{t("fieldPassword")}</Label>
            <Input
              id="create-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t("createPasswordPlaceholder")}
              autoComplete="new-password"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-confirm-password">{tRegister("confirmPassword")}</Label>
            <Input
              id="create-confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder={tRegister("confirmPasswordPlaceholder")}
              autoComplete="new-password"
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
