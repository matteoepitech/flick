import { ArrowUpRight, Check } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"

import { AsciinemaPlayer } from "@/components/asciinema-player"
import { GithubIcon } from "@/components/icons"
import { JsonLd } from "@/components/json-ld"
import { MouseMist } from "@/components/mouse-mist"
import { Button } from "@/components/ui/button"
import { Link } from "@/i18n/navigation"
import { localizedPath, SITE_NAME, SITE_URL } from "@/lib/seo"

export default function Page() {
  const locale = useLocale()
  const tHero = useTranslations("Hero")
  const tFeatures = useTranslations("Features")
  const tHow = useTranslations("HowItWorks")
  const tCTA = useTranslations("CallToAction")
  const tFooter = useTranslations("Footer")
  const tL = useTranslations("Landing")
  const tSeo = useTranslations("Seo")

  const comparPoints = [
    { title: tL("compar.row.selfhosted.title"), body: tL("compar.row.selfhosted.body") },
    { title: tL("compar.row.opensource.title"), body: tL("compar.row.opensource.body") },
    { title: tL("compar.row.nocloud.title"), body: tL("compar.row.nocloud.body") },
    { title: tL("compar.row.code.title"), body: tL("compar.row.code.body") },
    { title: tL("compar.row.noaccount.title"), body: tL("compar.row.noaccount.body") },
    { title: tL("compar.row.expiry.title"), body: tL("compar.row.expiry.body") },
  ]

  const faqs = [
    { q: tL("faq.q1"), a: tL("faq.a1") },
    { q: tL("faq.q2"), a: tL("faq.a2") },
    { q: tL("faq.q3"), a: tL("faq.a3") },
    { q: tL("faq.q4"), a: tL("faq.a4") },
    { q: tL("faq.q5"), a: tL("faq.a5") },
  ]

  const jsonLd = [
    {
      "@context": "https://schema.org",
      "@type": "SoftwareApplication",
      name: SITE_NAME,
      url: localizedPath(locale, "/"),
      applicationCategory: "Open-source self-hosted file transfer",
      operatingSystem: "Linux, Docker, Web",
      description: tSeo("description"),
      offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
      sameAs: ["https://github.com/Flick-Corp/flick"],
    },
    {
      "@context": "https://schema.org",
      "@type": "FAQPage",
      mainEntity: faqs.map((f) => ({
        "@type": "Question",
        name: f.q,
        acceptedAnswer: { "@type": "Answer", text: f.a },
      })),
    },
    {
      "@context": "https://schema.org",
      "@type": "Organization",
      name: SITE_NAME,
      url: SITE_URL,
      logo: `${SITE_URL}/assets/flick_logo.png`,
      sameAs: ["https://github.com/Flick-Corp/flick"],
    },
  ]

  const features = [
    { tag: tFeatures("instantTag"), title: tFeatures("instantTitle"), body: tFeatures("instantBody") },
    { tag: tFeatures("receiveTag"), title: tFeatures("receiveTitle"), body: tFeatures("receiveBody") },
    { tag: tFeatures("expirationTag"), title: tFeatures("expirationTitle"), body: tFeatures("expirationBody") },
    { tag: tFeatures("protectionTag"), title: tFeatures("protectionTitle"), body: tFeatures("protectionBody") },
  ]

  const steps = [
    { title: tHow("step1Title"), body: tHow("step1Body") },
    { title: tHow("step2Title"), body: tHow("step2Body") },
    { title: tHow("step3Title"), body: tHow("step3Body") },
  ]

  return (
    <main className="relative mx-auto max-w-6xl px-6 pb-24">
      <JsonLd data={jsonLd} />
      <MouseMist />

      <section className="flex flex-col items-center pt-20 text-center sm:pt-24">
        <h1 className="max-w-3xl font-heading text-5xl font-bold tracking-tight sm:text-6xl md:text-7xl">
          {tHero("titleStart")} <span className="text-primary">{tHero("titleHighlight")}</span>
        </h1>
        <p className="mt-6 max-w-xl text-lg leading-relaxed text-muted-foreground">{tHero("description")}</p>

        <div className="mt-9 flex flex-col gap-3 sm:flex-row">
          <Button asChild size="lg" className="h-13 px-6 text-base">
            <Link href="/send">
              <ArrowUpRight className="size-5" />
              {tHero("ctaSend")}
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="h-13 px-6 text-base">
            <Link href="/receive">{tHero("ctaReceive")}</Link>
          </Button>
        </div>
      </section>

      <section className="mx-auto mt-16 max-w-3xl">
        <div className="overflow-hidden rounded-xl border border-border bg-[#0d1117] shadow-2xl shadow-black/40">
          <div className="flex items-center gap-2 border-b border-white/5 px-4 py-3">
            <span className="size-3 rounded-full bg-[#ff5f57]" />
            <span className="size-3 rounded-full bg-[#febc2e]" />
            <span className="size-3 rounded-full bg-[#28c840]" />
            <span className="ml-2 font-mono text-xs text-muted-foreground">Flick CLI</span>
          </div>
          <div className="p-4">
            <AsciinemaPlayer src="/flick-demo.cast" />
          </div>
        </div>
      </section>

      <section className="mt-28 grid grid-cols-1 gap-y-10 sm:grid-cols-2 lg:grid-cols-4 lg:gap-y-0">
        {features.map((feature, index) => (
          <div
            key={feature.title}
            className="px-0 lg:border-border lg:px-8 lg:not-first:border-l lg:first:pl-0"
            data-index={index}
          >
            <p className="mb-3.5 font-heading text-xs font-semibold tracking-[0.14em] text-primary uppercase">
              {feature.tag}
            </p>
            <h3 className="font-heading text-lg font-bold">{feature.title}</h3>
            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{feature.body}</p>
          </div>
        ))}
      </section>

      <section className="mt-28">
        <div className="mx-auto max-w-2xl text-center">
          <p className="font-heading text-xs font-semibold tracking-[0.14em] text-primary uppercase">
            {tHow("eyebrow")}
          </p>
          <h2 className="mt-4 font-heading text-3xl font-bold tracking-tight sm:text-4xl">{tHow("title")}</h2>
        </div>

        <div className="mt-12 grid grid-cols-1 gap-12 md:grid-cols-3">
          {steps.map((step, index) => (
            <div key={step.title}>
              <div className="font-heading text-5xl leading-none font-extrabold text-primary">
                {String(index + 1).padStart(2, "0")}
              </div>
              <h3 className="mt-4 font-heading text-xl font-bold">{step.title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{step.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-28">
        <div className="mx-auto max-w-2xl text-center">
          <p className="font-heading text-xs font-semibold tracking-[0.14em] text-primary uppercase">
            {tL("compar.eyebrow")}
          </p>
          <h2 className="mt-4 font-heading text-3xl font-bold tracking-tight sm:text-4xl">{tL("compar.title")}</h2>
          <p className="mt-4 leading-relaxed text-muted-foreground">{tL("compar.body")}</p>
        </div>

        <div className="mx-auto mt-10 grid max-w-3xl grid-cols-1 gap-4 sm:grid-cols-2">
          {comparPoints.map((point) => (
            <div key={point.title} className="flex gap-3 rounded-xl border border-border bg-accent/20 p-5">
              <Check className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden />
              <div>
                <h3 className="font-heading font-semibold">{point.title}</h3>
                <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{point.body}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-28">
        <div className="mx-auto max-w-2xl text-center">
          <p className="font-heading text-xs font-semibold tracking-[0.14em] text-primary uppercase">
            {tL("faq.eyebrow")}
          </p>
          <h2 className="mt-4 font-heading text-3xl font-bold tracking-tight sm:text-4xl">{tL("faq.title")}</h2>
        </div>
        <div className="mx-auto mt-10 max-w-2xl divide-y divide-border">
          {faqs.map((item) => (
            <div key={item.q} className="py-5">
              <h3 className="font-heading text-lg font-semibold">{item.q}</h3>
              <p className="mt-2 leading-relaxed text-muted-foreground">{item.a}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-28">
        <div className="relative overflow-hidden rounded-2xl border border-border bg-gradient-to-b from-accent/60 to-card p-12 text-center sm:p-16">
          <h2 className="font-heading text-3xl font-bold tracking-tight sm:text-4xl">{tCTA("title")}</h2>
          <p className="mx-auto mt-4 max-w-md text-muted-foreground">{tCTA("body")}</p>
          <Button asChild size="lg" className="mt-8 h-13 px-6 text-base">
            <Link href="/register">{tCTA("button")}</Link>
          </Button>
        </div>
      </section>

      <footer className="mt-20 flex flex-col items-center justify-between gap-4 border-t border-border pt-7 text-sm text-muted-foreground sm:flex-row">
        <span>{tFooter("rights")}</span>
        <a
          href="https://github.com/Flick-Corp/flick"
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 transition-colors hover:text-foreground"
        >
          <GithubIcon className="size-4" />
          GitHub
        </a>
      </footer>
    </main>
  )
}
