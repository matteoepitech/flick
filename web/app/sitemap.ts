import type { MetadataRoute } from "next"

import { routing } from "@/i18n/routing"
import { alternatesFor, localizedPath } from "@/lib/seo"

// Public, indexable, locale-agnostic paths. Dashboard/auth routes are omitted on purpose.
const PUBLIC_PATHS = [
  { path: "/", priority: 1 },
  { path: "/send", priority: 0.8 },
  { path: "/receive", priority: 0.8 },
]

export default function sitemap(): MetadataRoute.Sitemap {
  return PUBLIC_PATHS.flatMap(({ path, priority }) =>
    routing.locales.map((locale) => ({
      url: localizedPath(locale, path),
      changeFrequency: "weekly" as const,
      priority,
      alternates: { languages: alternatesFor(path) },
    })),
  )
}
