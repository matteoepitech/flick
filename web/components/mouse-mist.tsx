"use client"

import { useEffect, useRef } from "react"

export function MouseMist() {
  const ref = useRef<HTMLDivElement>(null)
  const target = useRef({ x: 0, y: 0 })
  const current = useRef({ x: 0, y: 0 })
  const raf = useRef<number | null>(null)

  useEffect(() => {
    target.current = { x: window.innerWidth / 2, y: window.innerHeight / 3 }
    current.current = { ...target.current }

    const onMove = (e: PointerEvent) => {
      target.current = { x: e.clientX, y: e.clientY }
    }

    const tick = () => {
      const t = target.current
      const c = current.current
      c.x += (t.x - c.x) * 0.08
      c.y += (t.y - c.y) * 0.08
      if (ref.current) {
        ref.current.style.transform = `translate3d(${c.x}px, ${c.y}px, 0) translate(-50%, -50%)`
      }
      raf.current = requestAnimationFrame(tick)
    }

    window.addEventListener("pointermove", onMove, { passive: true })
    raf.current = requestAnimationFrame(tick)

    return () => {
      window.removeEventListener("pointermove", onMove)
      if (raf.current !== null) cancelAnimationFrame(raf.current)
    }
  }, [])

  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 -z-10 hidden overflow-hidden pointer-fine:block">
      <div
        ref={ref}
        className="absolute top-0 left-0 h-[40rem] w-[40rem] rounded-full bg-primary/40 opacity-70 blur-[120px] will-change-transform dark:bg-primary/25"
      />
    </div>
  )
}
