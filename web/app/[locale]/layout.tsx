import type { Metadata } from "next"
import { notFound } from "next/navigation"
import { Bricolage_Grotesque, Hanken_Grotesk, JetBrains_Mono } from "next/font/google"
import { hasLocale, NextIntlClientProvider } from "next-intl"
import { setRequestLocale } from "next-intl/server"

import "../globals.css"
import { PageTransition } from "@/components/page-transition"
import SiteHeader from "@/components/site-header"
import { ThemeProvider } from "@/components/theme-provider"
import { routing } from "@/i18n/routing"
import { cn } from "@/lib/utils"

const fontSans = Hanken_Grotesk({
  subsets: ["latin"],
  variable: "--font-sans",
  weight: ["400", "500", "600", "700", "800"],
})

const fontHeading = Bricolage_Grotesque({
  subsets: ["latin"],
  variable: "--font-heading",
  weight: ["400", "500", "600", "700", "800"],
})

const fontMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  weight: ["400", "500", "600", "700"],
})

export const metadata: Metadata = {
  icons: {
    icon: "/assets/flick_logo.png",
    shortcut: "/favicon.ico",
    apple: "/assets/flick_logo.png",
  },
}

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }))
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  if (!hasLocale(routing.locales, locale)) {
    notFound()
  }
  setRequestLocale(locale)

  return (
    <html
      lang={locale}
      suppressHydrationWarning
      className={cn("antialiased", "font-sans", fontSans.variable, fontHeading.variable, fontMono.variable)}
    >
      <body>
        <NextIntlClientProvider>
          <ThemeProvider>
            <SiteHeader />
            <PageTransition>{children}</PageTransition>
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
