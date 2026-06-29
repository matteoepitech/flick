import { ArrowUpRight } from "lucide-react"
import { useTranslations } from "next-intl"

import { AsciinemaPlayer } from "@/components/asciinema-player"
import { GithubIcon } from "@/components/icons"
import { MouseMist } from "@/components/mouse-mist"
import { Button } from "@/components/ui/button"
import { Link } from "@/i18n/navigation"

export default function Page() {
  const tHero = useTranslations("Hero")
  const tFeatures = useTranslations("Features")
  const tHow = useTranslations("HowItWorks")
  const tCTA = useTranslations("CallToAction")
  const tFooter = useTranslations("Footer")

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
