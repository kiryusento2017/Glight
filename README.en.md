# Claude Code Light 🚦

A liquid-glass desktop traffic-light widget for Windows that shows your [Claude Code](https://claude.ai/code) status in real time. **Glance at it while vibe coding and you'll always know what the AI is up to.**

## System Requirements

| Requirement | Details |
|---|---|
| **OS** | **Windows only**. No macOS / Linux support (depends on DirectComposition + Desktop Duplication; no cross-platform plans) |
| **Claude Code** | Requires [Claude Code](https://claude.ai/code) CLI installed locally. This is a status indicator **purpose-built for Claude Code** — it does not support Cursor, Copilot, or other AI tools |
| **Runtime** | None. A single native `.exe` — no Node, .NET, or any framework required |

## What It Looks Like

A **liquid-glass pill** floating on your desktop. You can see the real desktop refracted and distorted through it, with three traffic lights in the center reflecting Claude Code's current state:

| Light | Meaning | Claude Code State |
|---|---|---|
| 🔴 Red blinking | Executing a tool | PreToolUse |
| 🟡 Yellow blinking | Thinking | UserPromptSubmit / PostToolUse |
| 🟢 Green steady | Idle | Stop |
| ⚫ All off | Not running | Claude Code process not found |

Press and hold the glass and it **deforms like jelly** (narrower horizontally, stretched vertically). The faster you drag, the narrower it gets — release and it snaps back with a bouncy spring. All physics parameters are hot-tunable in `glass-tuning.json`.

**Global scaling**: Right-click → "Resize…" opens a slider window for stepless 100%–2000% scaling of the entire widget (top edge fixed, expands downward). Size and position are auto-saved and restored on next launch. One-click reset to default is available.

## How It Works

When you double-click the `.exe`, the widget starts up in this order (**zero system file modifications**):

1. **Single-instance check** — exits immediately if already running.
2. **Load config** — reads `config.json` (position / scale / visibility / autostart) and `glass-tuning.json` (visual / physics parameters) from the exe's directory. If `glass-tuning.json` doesn't exist, it **auto-generates defaults** (for you to edit). If `config.json` doesn't exist, it uses **in-memory defaults** (no file written until your first save action).
3. **Sync autostart** — if autostart was previously enabled, corrects the registry path to the current exe location.
4. **Install hooks (branched)**:
   - **Claude Code installed** (`~/.claude/settings.json` exists) → idempotently merges 4 status hooks (backs up `.bak` first, append-only, no duplicates).
   - **Claude Code not installed** (`~/.claude/` doesn't exist) → **silently skips, creates no files or directories**.
5. **Create window + start monitoring** — creates the transparent topmost glass window; watcher polls status files every 100ms and checks for `claude.exe` every 3s.

**With Claude Code**: lights change in real time as Claude works (red = executing, yellow = thinking, green = idle). After closing Claude, all three lights go off within 3s.
**Without Claude Code**: the widget displays normally as liquid glass, but with no status detected, all three lights stay **grey** — and nothing is written to `~/.claude/`.

## About Claude Code

This widget uses **Claude Code's Hooks mechanism** for real-time status — no transcript polling:

- **On first run**, it checks `~/.claude/settings.json`: if present (Claude Code installed), it idempotently merges 4 hook rules (backs up first, append-only, no duplicates); **if absent, it silently skips — no files or directories are created** on machines without Claude Code.
- Every time Claude Code fires a lifecycle event, the hook calls the widget itself to write a status file. The widget reflects changes on the lights within 100ms.
- **Auto-off**: checks for the `claude.exe` process every 3s; after closing Claude Code, all lights go off within 3s at most.
- The hook handler is the widget `.exe` itself (`claude-traffic-light.exe hook <state>`) — **zero external dependencies**.

> If Claude Code isn't installed on your machine, the widget runs fine but stays grey (all lights off) since it can't detect any status.

## Install & Use

> ⚠️ **Run as administrator on first launch**: the widget auto-generates `glass-tuning.json` in the exe's directory on first run, and writes `config.json` and `~/.claude/settings.json` during subsequent use. If the exe's directory lacks write permissions (e.g., some `C:\Program Files\` paths), config files cannot be created or updated.
>
> **Three files need write access**:
>
> 1. **exe directory** — `config.json` and `glass-tuning.json` are read/written here
> 2. **`~/.claude/settings.json`** — the widget merges hook config into this file (idempotent, backs up first)
>
> Right-click the exe → "Run as administrator" — **only needed the first time**. After that, just double-click normally.
>
> ![Running as administrator](assets/admin-permission.png)

1. Download `claude-traffic-light.exe` from [Releases](../../releases)
2. **First launch**: right-click the exe → "Run as administrator" (to generate config files); after that, double-click normally
3. Open Claude Code, start vibe coding
4. **Drag**: click and hold the visible pill to move it (clicking the transparent area outside the pill does nothing)
5. **Right-click menu** (in code order):
   - **Show/Hide** — toggle widget visibility; restore from tray icon when hidden
   - **Lock Position** — when locked, dragging is disabled to prevent accidental moves
   - **Launch on Startup** — writes to `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` (no UAC prompt; unchecks deletes the entry)
   - **Resize…** — opens a slider window for stepless 100%–2000% scaling; saves on release, close manually
   - **Reset Size & Position** — back to 100%, centered at top of screen
   - **Restart** — one-click restart when the widget freezes or glitches (new instance polls until the old one releases its lock, guaranteed not to "close without reopening")
   - **Exit**

**Tuning**: edit `glass-tuning.json` in the exe's directory (auto-generated on first run); changes take effect immediately on save — no restart needed.

**System tray**: a widget icon appears in the taskbar after launch. **Right-click the tray icon** for the same menu as right-clicking the window; **double-click the tray icon** to show/hide the window.

## Files Written

On first run and during subsequent use, the widget reads/writes files in the following locations. **No system files are modified.**

| Path | Content | Notes |
|---|---|---|
| `~/.claude/settings.json` | 4 hook rules | Idempotently written on first launch (backup → merge → write back) |
| `~/.claude/settings.json.bak` | Pre-modification original | Created when hook config changes (first install / exe moved or renamed) |
| `~/.claude/agent-light/agent-light-state-<session_id>` | Status word (`idle`/`thinking`/`running`), one file per session | Overwritten on each Claude Code hook event |
| `./config.json` | Position + lock/visibility/scale/autostart | Saved on drag release / menu action / resize; exe directory |
| `./glass-tuning.json` | All visual and physics parameters | Auto-generated on first run; manually edited, hot-reloaded |

> If "Launch on Startup" is enabled, a registry entry is written to `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` (user space, no UAC prompt). Unchecking it deletes the entry.

## Tuning: Defaults & Ranges

Edit `glass-tuning.json` in the exe's directory — **changes hot-reload on save (~500ms), no restart or recompile needed**. This file is per-machine and not version-controlled; deleting it regenerates defaults on next launch.

> The authoritative source of defaults is `config/tuning.go`'s `DefaultTuning()` (compiled into the exe). New users without a json file get these defaults on first run.

**Visual Parameters**

| Parameter | Meaning | Default | Suggested Range |
|---|---|---|---|
| `cornerR` | Corner radius (px) | 48 | 0–48 true rounded rect; =48 short edge fully round; 48–115 approaches pill shape |
| `cornerN` | Corner curvature exponent | 2.1 | 2.0–4.0 (2 = standard round, larger = squarer / Apple G2 feel) |
| `refractBand` | Refraction band depth (px) | 3 | 1–30 (small = edge-only, large = deep refraction) |
| `edgeSqueeze` | Edge squeeze | 0.25 | 0–1 (0 = strongest refraction, 1 = no refraction) |
| `contrast` | Contrast | 1.2 | 0.5–2.0 (1 = unchanged) |
| `brightness` | Brightness | 0.9 | 0.5–2.0 (<1 = darker) |
| `saturate` | Saturation | 1.5 | 0–3 (0 = grayscale, 1 = unchanged) |
| `lampR` | Lamp radius (px) | 19 | 6–30 (>32 overlaps adjacent lamps) |
| `lampGap` | Lamp spacing (px) | 64 | 38–90 (<2×lampR overlaps, >95 out of pill) |
| `glow` | Lit glow | 0 | 0–1 (additive; high values on dark backgrounds may wash out into halos — use with caution) |

**Physics Parameters**

| Parameter | Meaning | Default | Suggested Range |
|---|---|---|---|
| `springK` | Spring stiffness | 120 | 30–300 (higher = faster, snappier return) |
| `springC` | Damping | 8 | 1–20 (lower = more overshoot / oscillation) |
| `steadyX` | Steady horizontal scale | 0.91 | 0.8–1.04 (<1 = narrower at rest) |
| `steadyY` | Steady vertical scale | 1.11 | 0.9–1.30 (>1 = taller at rest) |
| `pressX` | Press horizontal scale | 0.82 | 0.5–1.04 (<1 = narrower on press) |
| `pressY` | Press vertical scale | 1.22 | 0.8–1.30 (>1 = taller on press) |
| `dragK` | Drag deformation strength | 0.02 | 0.001–0.05 (higher = more deformation while dragging) |
| `dragMin` | Drag deformation floor | 0.5 | 0.3–1.0 (0.5 = at most 50% compression) |
| `releaseImpulse` | Release overshoot multiplier | 1.5 | 1.0–3.0 (>1 = stronger bounce) |

> ⚠️ **Deformation has hard caps — exceeding them clips the canvas**: the canvas is 240×144 and the glass is 230×96, so at any moment **horizontal scale ≤ 240/230 ≈ 1.04, vertical scale ≤ 144/96 = 1.50**. If the peak of `steadyY` + `pressY` + release overshoot exceeds 1.50, the pill top/bottom will be clipped. Vertical dragging is internally clamped at `maxDragScaleY=1.4` to prevent hitting the wall. For more extreme stretching, you'd need to modify `CANVAS`/`PILL` in `ui/glass.hlsl` (requires recompile, not hot-reloadable).

## Architecture

```
main.go             Entry: single-instance mutex (--restarted polling) → load config → sync autostart → install hooks (idempotent: exe real name) → create window → start monitoring
hookinstall.go      Idempotently merge status hooks into ~/.claude/settings.json
config/             Config read/write (config.json position/scale/autostart + glass-tuning.json visual hot-reload)
  autostart.go      HKCU Run registry read/write/delete + path self-correction (launch on startup)
state/              Four-state enum (Grey/Green/Yellow/Red) and priority aggregation
watcher/            Every 100ms aggregate per-session status files (any session busy = global busy) + every 3s process check fallback (procmon.go)
ui/
  window.go           DComp transparent topmost window, message loop, self-managed mouse drag, spring physics state machine, icon loading
  render.go           D3D11 render pipeline: device/swapchain/shader compilation + per-frame draw (dynamic viewport)
  glass.hlsl          Pixel shader: superellipse SDF shape mask, shuding refraction kernel, three-lamp overlay
  capture.go          Desktop Duplication capture (supports resize on scale change + rate-limited rebuild on session switch invalidation)
  com.go              D3D11/DXGI/DComp COM bindings (including swapchain ResizeBuffers)
  win32.go            Win32 API bindings and constants
  physics.go          Second-order spring physics (per-frame Euler integration driving deformation)
  sizedialog.go       Resize slider window (comctl32 trackbar, 100%–2000% stepless scaling)
```

### Status Detection: Claude Code Hooks

No transcript polling. Four lifecycle hooks push status in real time:

```
UserPromptSubmit → Yellow (thinking)
PostToolUse      → Yellow (thinking)
PreToolUse       → Red (executing)
Stop             → Green (idle)
```

Hooks are installed in exec form (`command` = exe path + `args`, spawned directly without a shell). Each hook reads `session_id` from stdin JSON and writes a **per-session status file** at `~/.claude/agent-light/agent-light-state-<sid>`. The watcher aggregates all session files every 100ms: **any session busy → global busy** (prevents one agent finishing from falsely pulling green during multi-agent concurrency); all idle → green; no files → grey.

- **Busy state trusts hook content only**: `running` → red, `thinking` → yellow, `idle` → green. No time-window staleness fallback (long thinking without tool calls → no hook events → time-based fallback would falsely downgrade to green).
- **Process detection as the sole fallback**: every 3s uses `CreateToolhelp32Snapshot` to check for `claude.exe`; if absent, forces grey — crash / force-kill / stale boot residue all resolve to grey via this path, eliminating stuck yellow/red on boot.
- **Scheduled cleanup (10min)**: every 30s deletes stale files not updated for over 10 minutes (pure disk reclamation).

### Render Pipeline

```
Desktop Duplication captures full-screen desktop texture (GPU-resident)
  → HLSL superellipse SDF + shuding refraction kernel (center clear, edges strongly refracted)
  → DXGI swapchain → DirectComposition transparent topmost window
WDA_EXCLUDEFROMCAPTURE excludes self from capture, breaking the feedback loop
```

### Spring Deformation

A second-order spring (stiffness K + damping C) integrates via Euler per frame. The main thread sets targets on mouse events; the render thread advances physics each frame:

- **Pressed**: narrower horizontally + taller vertically (`pressX` / `pressY`)
- **Steady state**: slightly narrower and taller (`steadyX` / `steadyY`)
- **Dragging**: additional compression proportional to speed (`dragK`, floor `dragMin`)
- **Released**: returns to steady state with velocity-based overshoot impulse (`releaseImpulse`)

All parameters hot-tunable in `glass-tuning.json`.

## Build

Requires Go toolchain + Windows:

```powershell
# Debug (console visible for output, no exe produced)
go run .

# Build exe (local testing / release — the one and only command; mkdir dist first if absent)
go build -trimpath -buildvcs=false -ldflags="-H windowsgui" -o dist/claude-traffic-light.exe .

# Test
go test ./...
```

> **Build rule**: debug with `go run .` (no exe); whenever producing an exe, use the single command above with all four safeguards — `-trimpath` (strips local paths/usernames), `-buildvcs=false` (strips git metadata), `-ldflags="-H windowsgui"` (no console window), `-o dist/` (isolated output), with icon + version info auto-embedded. **Never `-s -w`** (triggers Wacatac false positives), never pack with UPX. See [docs/编译构建发行.md](docs/编译构建发行.md) for the full workflow.

### Exe Icon + Version Info (Explorer Properties)

The window/tray icon is loaded at runtime by `main.go`'s `//go:embed claude-traffic-light.ico`, while the **exe file icon + version info in Explorer properties** (product name / version / attribution) is a separate mechanism — embedded at **link time** via `rsrc_windows_amd64.syso`.

- This syso is generated by **goversioninfo** from `versioninfo.json` (version metadata) + the ico **combined into one**; both `versioninfo.json` and the syso are **checked in**, so cloning and running `go build` directly yields an exe with icon + version info.
- **Only regenerate** when changing the icon / version number / attribution:
  ```powershell
  goversioninfo -icon=claude-traffic-light.ico -o=rsrc_windows_amd64.syso
  ```
  (Install once: `go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest`)
- ⚠️ **The filename must include `_windows_amd64`** so `go build` auto-selects it by platform. **Never** generate a second `.syso` with a resource section (e.g., `rsrc.syso`) — two `.rsrc` sections → linker fails with `too many .rsrc sections`.

## Tech Stack

Go + D3D11 + DirectComposition + HLSL (no WebView2, no Electron, no CGO)

## Limitations / Out of Scope

- **Windows only** — no macOS / Linux support
- **Claude Code only** — no support for other AI tools
- Click-through: DComp transparent topmost windows cannot pass clicks through; the transparent area outside the pill will block clicks to windows underneath (known limitation)
- No sound alerts, no multi-monitor auto-positioning

## License

MIT
