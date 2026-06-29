import { writeFileSync } from "node:fs"

const ESC = String.fromCharCode(27)
const ORANGE = `${ESC}[38;2;232;132;58m`
const GREEN = `${ESC}[38;2;16;185;129m`
const DIM = `${ESC}[38;2;139;148;158m`
const FG = `${ESC}[38;2;230;237;243m`
const BOLD = `${ESC}[1m`
const R = `${ESC}[0m`

const events = []
let t = 0.6
const push = (data) => events.push([Number(t.toFixed(3)), "o", data])
const wait = (s) => {
  t += s
}

push(`${GREEN}$${R} `)
wait(0.4)
for (const ch of "flick README.md") {
  push(`${FG}${ch}${R}`)
  wait(0.05 + Math.random() * 0.05)
}
wait(0.5)
push("\r\n")

wait(0.3)
push(`${FG}This upload contains:${R}\r\n`)
wait(0.15)
push(`${DIM}  • README.md (6.6 KB)${R}\r\n`)
wait(0.35)

push(`${DIM}Quota: [${"░".repeat(20)}] 0 B / 1000.0 MB used (0%)${R}\r\n`)
wait(0.5)

push(`${FG}Upload these files? ${DIM}[y/N]:${R} `)
wait(0.7)
push(`${FG}y${R}`)
wait(0.45)
push("\r\n")

const BAR = 22
function progress(label, totalKb, rate, frames, frameTime) {
  for (let i = 0; i <= frames; i++) {
    const pct = Math.round((i / frames) * 100)
    const filled = Math.round((i / frames) * BAR)
    const done = ((pct / 100) * totalKb).toFixed(1)
    const bar = `${ORANGE}${"█".repeat(filled)}${DIM}${"░".repeat(BAR - filled)}${R}`
    push(`\r${FG}${label}${R} ${DIM}${pct}%${R} |${bar}| ${DIM}(${done}/${totalKb} kB, ${rate})${R}`)
    wait(frameTime)
  }
  push("\r\n")
}

wait(0.2)
progress("Zipping...", "6.8", "1.8 MB/s", 12, 0.06)
wait(0.15)
progress("Uploading...", "3.2", "40 kB/s", 16, 0.07)
wait(0.4)

push(`${ORANGE}${BOLD}Code:${R}${ORANGE} bank-music-709${R}${DIM}  [15m left]${R}\r\n`)
wait(0.35)
push(`${GREEN}Code copied to clipboard.${R}\r\n`)

wait(7)
push(R)

const header = {
  version: 2,
  width: 74,
  height: 12,
  timestamp: 0,
  env: { TERM: "xterm-256color", SHELL: "/bin/zsh" },
  title: "Flick CLI",
}

const out = [JSON.stringify(header), ...events.map((e) => JSON.stringify(e))].join("\n") + "\n"
writeFileSync(new URL("../public/flick-demo.cast", import.meta.url), out)
console.log(`wrote public/flick-demo.cast (${events.length} events, ${t.toFixed(1)}s)`)
