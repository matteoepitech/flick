import { routing } from "@/i18n/routing"

/**
 * Canonical origin for this instance. Read from FLICK_SITE_ADDRESS (the same
 * variable the rest of the stack uses) so every self-hosted deployment gets its
 * own domain in canonical URLs, sitemap, hreflang and JSON-LD. Falls back to
 * localhost when unset.
 */
export const SITE_URL = (process.env.FLICK_SITE_ADDRESS ?? "http://localhost").replace(/\/$/, "")

export const SITE_NAME = "Flick"

/**
 * Build a localized absolute URL. `path` is locale-agnostic (e.g. "/send").
 * Routing uses localePrefix "always", so every locale (including the default)
 * is served under its own prefix — keep canonical/hreflang URLs in sync.
 */
export function localizedPath(locale: string, path = "/"): string {
  const clean = path === "/" ? "" : path
  return `${SITE_URL}/${locale}${clean}`
}

/**
 * hreflang alternates for a given locale-agnostic path, including x-default.
 * Feed straight into Next.js `metadata.alternates`.
 */
export function alternatesFor(path = "/") {
  const languages: Record<string, string> = {}
  for (const locale of routing.locales) {
    languages[locale] = localizedPath(locale, path)
  }
  languages["x-default"] = localizedPath(routing.defaultLocale, path)
  return languages
}
