<div align="center">

# 🚦 Glight

**A Claude Code desktop status traffic-light widget · liquid-glass look**

**A liquid-glass traffic light that floats on your desktop — one glance from the corner of your eye tells you what your AI is doing while you vibe code.**

![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)
![Language](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![Tech](https://img.shields.io/badge/D3D11_+_DirectComposition_+_HLSL-5C2D91)
![License](https://img.shields.io/badge/license-MIT-green)
[![Download](https://img.shields.io/badge/⬇_Download-Releases-success)](https://github.com/kiryusento2017/Glight/releases)

[简体中文](../../README.md) · **English** · [日本語](README.ja.md) · [한국어](README.ko.md) · [繁體中文](README.zh-TW.md)

<img src="../../assets/screenshot-wide.jpg" width="600" alt="Glight (Claude Code Light): a liquid-glass traffic light floating over VS Code, green light lit" />

<sub>A liquid-glass capsule that genuinely refracts your desktop, with three lights that track Claude Code's status in real time.</sub>

</div>

---

## What is this

**Glight** (a.k.a. Claude Code Light / Claude Code traffic-light status indicator) is a Windows desktop widget: an **always-on-top liquid-glass capsule** that genuinely refracts the desktop behind it. Embedded in the capsule are three traffic lights that **mirror [Claude Code](https://claude.ai/code)'s working status in real time** —

> Red blinking = running a tool; Yellow blinking = thinking; Green solid = idle, waiting for you; All off = not running.

No need to keep staring at the terminal — **a glance at the widget tells you whether the AI is busy, thinking, or stopped.** A single native `.exe`, no runtime dependencies, plug-and-play from a USB stick.

## ✨ Design philosophy

Inspired by **Apple's "Liquid Glass."** Unlike approaches that just smear a blur over the background, this is **real refraction**:

- **Refracts the real desktop in real time** — it grabs actual screen pixels and runs a custom HLSL refraction shader, so the windows, code, and wallpaper behind it get warped inward like through real glass (clear in the center, strong refraction at the edges) — not a frozen screenshot or a flat frosted blur.
- **Draggable** — grab the capsule and drop it anywhere; it remembers its position.
- **Squishy, jelly-like deformation on press** 🍮 — this is the soul of it. Press and the glass **squishes wide and stretches tall** like jelly; drag faster and it gets "flung" narrower; release and it bounces back via **second-order spring physics**, with a slight overshoot before settling. Every deformation parameter (stiffness / damping / overshoot) is hot-tunable in `glass-tuning.json` — save and it takes effect instantly.
- **Free scaling** — right-click "Resize" for stepless 100%–2000% zoom; size and position are remembered.

## 🚦 Status reference

| Light | Meaning | Triggering Claude Code event |
|---|---|---|
| 🔴 **Red blinking** | Running a tool | `PreToolUse` |
| 🟡 **Yellow blinking** | Thinking | `UserPromptSubmit` / `PostToolUse` |
| 🟢 **Green solid** | Idle, waiting | `Stop` |
| ⚫ **All off** | Not running | `claude.exe` process is gone |

Priority: **Red > Yellow > Green > Grey**. With multiple concurrent Claude sessions, if any one is busy it shows busy — one session ending won't falsely flip it to idle.

## 💻 Requirements

| Requirement | Notes |
|---|---|
| **OS** | **Windows 10 (2004+) / 11 only.** **No macOS / Linux** — it relies on Windows-exclusive DirectComposition + Desktop Duplication, with no cross-platform plans. |
| **Claude Code** | Requires [Claude Code](https://claude.ai/code) CLI installed locally. This is a status indicator **built specifically for Claude Code** — not for Cursor / Copilot / other AI tools. It still runs without it, but the lights stay grey. |
| **Runtime** | **None.** A single native `.exe` — no Node / .NET / Electron / any framework. |

## 📦 Install (3 steps)

**1️⃣ Download**

Grab `claude-traffic-light.exe` from [Releases](https://github.com/kiryusento2017/Glight/releases) and put it in any folder.

**2️⃣ First launch as administrator**

> Right-click the exe → "Run as administrator". **Only needed the first time** — after that, just double-click.

On first launch the widget writes two config files next to the exe: `config.json` (position/scale) and `glass-tuning.json` (visual/deformation params).

**3️⃣ Grant read + write permission to those two files**

If the widget lives on the **C: drive** (especially protected dirs like `C:\Program Files\`), Windows may block it from writing config files. In that case, manually grant **Read + Write** to those two files: right-click the file → "Properties" → "Security" → "Edit", check "Write".

> 💡 **This step is usually only needed on the C: drive.** On other drives (e.g. D:), the widget can normally write its config files without admin rights — you can skip this.

![Granting read/write permission to config.json / glass-tuning.json](../../assets/admin-permission.png)

Once set up, open Claude Code and start vibe coding — the lights will follow along.

## 🔒 What files does it touch on your computer?

Running an unfamiliar `.exe`, the thing you should care about most is "what exactly does it write to my computer?" So here's every file it touches and how it works, all laid bare — it **modifies no system files and makes no network connections whatsoever.**

### How it works: Claude Code's "Hooks"

Claude Code ships with a "hooks" mechanism: at key moments — when it **starts running a tool, is thinking, or stops** — it automatically runs commands you've registered in advance. This widget rides on that — on first launch it **adds 4 hooks** to Claude Code's own config file `~/.claude/settings.json`, so that whenever Claude Code's status changes it "pings" the widget, which switches the lights accordingly. The command those 4 hooks call **is the widget itself** (`claude-traffic-light.exe hook <state>`) — no third-party programs needed (unlike approaches that require Node).

We're conservative when merging the hooks:

- **Back up the entire `settings.json`** to `settings.json.bak` before touching it;
- **Add only our own 4 entries — never delete or alter any of your existing config**;
- Don't add them again if already present (idempotent);
- **If you don't have Claude Code (the `~/.claude/` folder doesn't exist), it writes nothing — it won't even create the folder.**

### Every file it creates / writes

| File | Where | What for | When written |
|---|---|---|---|
| `config.json` | **next to the exe** | Remembers the widget's position / size / toggles | On drag, resize, menu actions |
| `glass-tuning.json` | **next to the exe** | Glass appearance & deformation params for you to tweak | Generated once on first launch |
| `settings.json` (**adds 4 hooks**) | `~/.claude/` | Lets Claude Code notify the widget on status change | First launch; backed up first |
| `settings.json.bak` | `~/.claude/` | A **full backup** of the above, pre-change | Only when the hooks change |
| `agent-light-state-<session-id>` | `~/.claude/agent-light/` | Each Claude session's current status (**just one word**: `running`/`thinking`/`idle`) | Overwritten on each status change |
| Registry `Run` entry | `HKCU\...\Run` (user-level, **no UAC**) | Implements "Start on boot" | **Only if you enable** autostart; removed when unchecked |

> The first two (`config.json` / `glass-tuning.json`) sit right next to the exe — plainly visible, deletable anytime. The rest live under your own `~/.claude/` user folder. It **never touches C: system directories or sensitive registry areas.**

### Multiple Claudes at once won't confuse it

Every Claude session gets its **own little state file** (one word inside). When several run at once, the widget aggregates these files every 0.1s: **as long as any session is busy, the light shows busy; only when all are idle does it turn green.** So when one agent finishes while another is still running, the light **won't be falsely flipped to green.** These small files are auto-cleaned if not updated for over 10 minutes, so they never pile up.

### Want to wipe it completely?

Delete the two `.json` files next to the exe + the `~/.claude/agent-light/` folder, then remove those 4 hooks from `~/.claude/settings.json` (or just restore from `.bak`); turning off "Start on boot" auto-removes the registry entry. **Clean uninstall, no leftovers.**

## 🎮 Usage

- **Drag** — hold the visible capsule and drag to reposition (clicks on the transparent area outside the capsule do nothing, so no accidental drags).
- **Right-click / tray menu**:
  | Item | Effect |
  |---|---|
  | Show / Hide | Toggle visibility; restore from the tray icon |
  | Lock position | Locked = can't be dragged, prevents accidental moves |
  | Start on boot | Writes a registry Run entry (user scope, no UAC prompt); unchecking removes it |
  | Resize… | Slider window, stepless 100%–2000% zoom |
  | Reset size & position | Back to 100%, centered at top of screen |
  | Restart | One-click restart for stuck frames/glitches (new instance takes over cleanly) |
  | Exit | — |
- **System tray** — right-click the tray icon = same menu; double-click = show/hide.
- **Tuning** — edit `glass-tuning.json` next to the exe; **save and it hot-reloads (~0.5s)**, no restart needed.

---

<details>
<summary><b>🔧 Tuning: all parameters & defaults</b> (click to expand)</summary>

Edit `glass-tuning.json` next to the exe; **save and it hot-reloads, no restart or recompile**. This file is per-machine and not version-controlled; delete it and the defaults below regenerate on next launch. The authoritative defaults live in `DefaultTuning()` in `config/tuning.go`.

**Visual parameters**

| Param | Effect | Default | Range |
|---|---|---|---|
| `cornerR` | Corner radius (px); bigger = rounder | 48 | 0–115 |
| `cornerN` | Corner curvature exponent; 2 = standard circle, higher = squarer (Apple-style G2) | 2.1 | 2.0–4.0 |
| `refractBand` | Refraction band depth (px); only refracts within this depth of the edge | 3 | 1–30 |
| `edgeSqueeze` | Edge contraction; 0 = strongest refraction, 1 = none | 0.25 | 0–1 |
| `contrast` | Contrast | 1.2 | 0.5–2.0 |
| `brightness` | Brightness | 0.9 | 0.5–2.0 |
| `saturate` | Saturation | 1.5 | 0–3 |
| `lampR` | Lamp radius (px) | 19 | 6–30 |
| `lampGap` | Lamp spacing (px), red↔yellow↔green center distance | 64 | 38–90 |
| `glow` | Glow intensity when lit | 0 | 0–1 |

**Physics / deformation parameters**

| Param | Effect | Default | Range |
|---|---|---|---|
| `springK` | Spring stiffness; higher = faster, harder snap-back | 120 | 30–300 |
| `springC` | Damping; lower = more overshoot/bounce | 8 | 1–20 |
| `steadyX` | Resting horizontal scale; <1 = slightly narrow | 0.91 | 0.8–1.04 |
| `steadyY` | Resting vertical scale; >1 = slightly tall | 1.11 | 0.9–1.30 |
| `pressX` | Pressed horizontal scale; <1 = narrower | 0.82 | 0.5–1.04 |
| `pressY` | Pressed vertical scale; >1 = taller | 1.22 | 0.8–1.30 |
| `dragK` | Drag deformation strength; higher = more violent | 0.02 | 0.001–0.05 |
| `dragMin` | Drag deformation floor; 0.5 = shrinks to at most 50% | 0.5 | 0.3–1.0 |
| `releaseImpulse` | Release overshoot multiplier; >1 = stronger bounce | 1.5 | 1.0–3.0 |

> ⚠️ **Deformation has a hard cap**: canvas is 240×144, glass is 230×96, so at any moment horizontal scale ≤ 1.04 and vertical scale ≤ 1.50 — exceed it and the capsule top/bottom gets clipped by the canvas. Vertical drag has a built-in `maxDragScaleY=1.4` clamp. For more dramatic stretch, change `CANVAS`/`PILL` in `ui/glass.hlsl` (requires recompile).

</details>

<details>
<summary><b>🏗️ Architecture & how it works</b> (click to expand)</summary>

### Render pipeline (the core)

```
Desktop Duplication grabs the full-screen desktop texture (GPU-resident)
  → HLSL squircle SDF + refraction core (clear center, strong edge refraction)
  → DXGI composition swapchain → DirectComposition transparent topmost window
The window sets WDA_EXCLUDEFROMCAPTURE to exclude itself from capture, breaking the "refracting itself" feedback loop
```

Why not CSS `backdrop-filter` / WebView2? Because those can only sample inside the WebView document — they **can't reach the OS desktop**, so the old version could only show as a black box. Windows also has no system API for "refractive displacement of a window's background" (DWM Acrylic only blurs, no refraction). The only way is to grab desktop pixels yourself and write your own refraction shader.

### Module layout

```
main.go             Entry: single-instance mutex → load config → autostart sync → install hooks → window + watcher
hookinstall.go      Idempotently merges status hooks into ~/.claude/settings.json (backup first, additive only)
config/             config.json (position/scale/autostart) + glass-tuning.json (hot-reloaded visuals)
  autostart.go      Registry HKCU Run read/write/delete + path self-correction
state/              Four-state enum (grey/green/yellow/red) and priority aggregation
watcher/            Aggregates per-session state files every 100ms (any busy = busy) + 3s process check fallback
ui/
  window.go           DComp transparent topmost window, message loop, custom drag, spring deformation state machine
  render.go           D3D11 render pipeline: device/swapchain/shader compile + per-frame draw
  glass.hlsl          Pixel shader: squircle SDF shape + refraction core + three lamps
  capture.go          Desktop Duplication capture (Resize on zoom + throttled rebuild after invalidation)
  com.go              D3D11/DXGI/DComp COM bindings
  win32.go            Win32 API bindings and constants
  physics.go          Second-order spring physics (per-frame Euler integration)
  sizedialog.go       Resize slider window (comctl32 trackbar)
```

### Status detection: Claude Code Hooks (real-time)

No transcript polling — status is **pushed in real time** by Claude Code's 4 lifecycle hooks. On first launch the widget idempotently merges these 4 hooks into `~/.claude/settings.json`:

| Claude Code event | State word written | Light |
|---|---|---|
| `UserPromptSubmit` / `PostToolUse` | `thinking` | 🟡 thinking |
| `PreToolUse` | `running` | 🔴 running |
| `Stop` | `idle` | 🟢 idle |

Each hook calls the widget itself in exec form `claude-traffic-light.exe hook <state-word>` (spawned directly, no shell, avoiding Windows shell ambiguity), and does just one thing: write the state word into a **per-session state file**.

#### One state file per session: `agent-light-state-<session_id>`

When Claude Code fires a hook it passes a **JSON blob over stdin** containing the current session's `session_id`. The widget reads it and writes to:

```
~/.claude/agent-light/agent-light-state-<session_id>
```

The file's content is just one state word (`running` / `thinking` / `idle`). Key points:

- **One file per session, never overwriting each other** — this is the foundation for multi-agent support (below).
- The session_id is sanitized into a safe filename (letters/digits/hyphens only); if unavailable, it falls back to `default`.
- Reading stdin has a **500ms hard timeout**, so the hook always returns within milliseconds and **never slows Claude Code down**.
- Everything goes under the `agent-light/` subdirectory, instead of littering `~/.claude/`'s root.

#### Multi-agent concurrency: any session busy → globally busy

You may well run **multiple Claudes at once** (several terminals, or a main session spawning sub-agents), each writing its own state file. The watcher reads every `agent-light-state-*` file in `agent-light/` every 100ms and aggregates them into one global state by priority **Red > Yellow > Green > Grey**:

```
Session A: running  🔴 ┐
Session B: idle     🟢 ├─ take highest ─→ 🔴 Red (A is still working → show busy)
Session C: thinking 🟡 ┘
```

**As long as any session is busy, the widget shows busy.** So when one agent finishes first (writes `idle`) while another is still running, the light **won't be falsely flipped to green**; only when all sessions are `idle` does it go green, and only with no files at all does it go grey.

#### Other fallbacks

- **Busy state trusts only the hook content**, with no time-window timeout downgrade (long thinking has no tool calls → no hook, so a timeout would falsely mark it idle).
- **Process-check fallback**: every 3s, `CreateToolhelp32Snapshot` checks for `claude.exe`; if gone, force grey — crashes / force-kills / boot leftovers all fall back to grey via this.
- **Periodic cleanup**: every 30s, leftover session files not updated in over 10 minutes are deleted (pure disk reclamation, decoupled from state logic).
- **The hook handler is the widget itself** — zero external dependencies (unlike approaches needing node).

</details>

<details>
<summary><b>🛠️ Build from source</b> (click to expand)</summary>

Requires the Go toolchain + Windows:

```powershell
# Debug (with console output, no exe produced)
go run .

# Build exe (the one and only command, all four safeguards together)
go build -trimpath -buildvcs=false -ldflags="-H windowsgui" -o dist/claude-traffic-light.exe .

# Test
go test ./...
```

> **Build rule**: use `go run .` for debugging; produce an exe only with the command above — `-trimpath` (strips local paths/username), `-buildvcs=false` (strips git info), `-ldflags="-H windowsgui"` (no console window), `-o dist/` (isolation), and it auto-embeds `rsrc_windows_amd64.syso` (icon + version info). **Never use `-s -w`** (triggers AV false positives), **never pack** (UPX etc.).

**Recording a demo**: the normal exe is invisible to OBS and other screen recorders (by design, via `WDA_EXCLUDEFROMCAPTURE`). Launch with `--demo` to lift the exclusion so OBS can capture it — but then the glass refracts itself. For the cleanest demo footage, just film the screen with a phone/camera.

</details>

## 🚧 Limitations / out of scope

- **Windows only**, no macOS / Linux
- **Claude Code only**, no other AI tools
- **Click-through**: a DComp transparent topmost window can't pass clicks to the layer below, so the transparent area outside the capsule blocks clicks to windows underneath (known limitation)
- No sound alerts, no multi-monitor auto-positioning

## 📄 License

[MIT](../../LICENSE)
