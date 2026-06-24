import { ArrowDownLeft, ArrowRight, ArrowUpRight, Clock, KeyRound, Upload, Zap } from "lucide-react"
import { useTranslations } from "next-intl"

import { MouseMist } from "@/components/mouse-mist"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Link } from "@/i18n/navigation"

export default function Page() {
  const tHero = useTranslations("Hero")
  const tFeatures = useTranslations("Features")
  const tHow = useTranslations("HowItWorks")

  const features = [
    { icon: Zap, title: tFeatures("instantTitle"), body: tFeatures("instantBody") },
    { icon: Upload, title: tFeatures("receiveTitle"), body: tFeatures("receiveBody") },
    { icon: Clock, title: tFeatures("expirationTitle"), body: tFeatures("expirationBody") },
    { icon: KeyRound, title: tFeatures("protectionTitle"), body: tFeatures("protectionBody") },
  ]

  const steps = [
    { title: tHow("step1Title"), body: tHow("step1Body") },
    { title: tHow("step2Title"), body: tHow("step2Body") },
    { title: tHow("step3Title"), body: tHow("step3Body") },
  ]

  return (
    <main className="relative mx-auto max-w-6xl px-4 py-12 sm:px-6 sm:py-20">
      <MouseMist />
      <section className="flex flex-col items-center text-center">
        <h1 className="text-4xl tracking-tight sm:text-5xl md:text-6xl">
          {tHero("titleStart")} <span className="font-bold text-primary">{tHero("titleHighlight")}</span>
        </h1>
        <p className="mt-6 max-w-xl text-lg text-muted-foreground">{tHero("description")}</p>

        <div className="mt-10 flex flex-col gap-3 sm:flex-row">
          <Button asChild size="lg" className="h-14 px-8 text-lg">
            <Link href="/send">
              <ArrowUpRight className="size-6" />
              {tHero("ctaSend")}
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="h-14 px-8 text-lg">
            <Link href="/receive">
              <ArrowDownLeft className="size-6" />
              {tHero("ctaReceive")}
            </Link>
          </Button>
        </div>
      </section>

      <section className="mt-20 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
        {features.map((feature) => {
          const Icon = feature.icon
          return (
            <Card key={feature.title} className="p-8">
              <div className="flex items-center gap-3">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Icon className="h-5 w-5" />
                </span>
                <h3 className="text-sm font-semibold">{feature.title}</h3>
              </div>
              <p className="mt-4 text-base text-muted-foreground">{feature.body}</p>
            </Card>
          )
        })}
      </section>

      <section className="mt-32 mb-10">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-semibold tracking-widest text-primary uppercase">{tHow("eyebrow")}</p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl md:text-5xl">{tHow("title")}</h2>
        </div>

        <div className="mt-14 grid grid-cols-1 gap-6 md:grid-cols-3">
          {steps.map((step, index) => (
            <div key={step.title} className="relative">
              <Card className="h-full p-8">
                <span className="font-mono text-5xl font-bold text-primary">{String(index + 1).padStart(2, "0")}</span>
                <h3 className="mt-4 text-lg font-semibold">{step.title}</h3>
                <p className="mt-2 text-base text-muted-foreground">{step.body}</p>
              </Card>
              {index < steps.length - 1 && (
                <ArrowRight
                  aria-hidden
                  className="absolute top-1/2 -right-5 hidden h-5 w-5 -translate-y-1/2 text-muted-foreground/40 md:block"
                />
              )}
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}
