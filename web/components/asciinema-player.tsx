"use client"

import { useEffect, useRef } from "react"

import "asciinema-player/dist/bundle/asciinema-player.css"

type AsciinemaPlayerProps = {
  src: string
}

export function AsciinemaPlayer({ src }: AsciinemaPlayerProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    let disposed = false
    let player: { dispose: () => void } | undefined

    import("asciinema-player").then((mod) => {
      if (disposed || !el) return
      player = mod.create(src, el, {
        autoPlay: true,
        loop: true,
        controls: false,
        terminalFontSize: "14px",
        terminalFontFamily: "var(--font-mono), ui-monospace, monospace",
        terminalLineHeight: 1.45,
        theme: "flick",
        fit: "width",

        idleTimeLimit: 7.5,
      })
    })

    return () => {
      disposed = true
      player?.dispose()
    }
  }, [src])

  return <div ref={containerRef} aria-hidden />
}
